package aws

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	awstypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/smithy-go"

	"github.com/helmcode/finops-cli/internal/provider"
)

const maxCostExplorerMonths = 12

// ValidateDateRange checks that the date range is valid for Cost Explorer.
func ValidateDateRange(start, end time.Time) error {
	if start.After(end) {
		return fmt.Errorf("--from must be before --to")
	}

	if start.After(time.Now()) {
		return fmt.Errorf("--from cannot be in the future")
	}

	months := monthsBetween(start, end)
	if months > maxCostExplorerMonths {
		return fmt.Errorf("range exceeds 12 months (%d months requested, max %d)", months, maxCostExplorerMonths)
	}

	return nil
}

func monthsBetween(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months
}

// FetchCosts retrieves cost data from AWS Cost Explorer for the given parameters.
func (p *AWSProvider) FetchCosts(params provider.CostParams) ([]provider.CostRecord, error) {
	if err := ValidateDateRange(params.Start, params.End); err != nil {
		return nil, err
	}

	ctx := context.Background()

	groupBy := make([]awstypes.GroupDefinition, 0, len(params.GroupBy))
	for _, g := range params.GroupBy {
		groupBy = append(groupBy, awstypes.GroupDefinition{
			Type: awstypes.GroupDefinitionTypeDimension,
			Key:  strPtr(g),
		})
	}

	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &awstypes.DateInterval{
			Start: strPtr(params.Start.Format("2006-01-02")),
			End:   strPtr(params.End.Format("2006-01-02")),
		},
		Granularity: awstypes.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		GroupBy:     groupBy,
	}

	// If we have an account filter and this is an org, add a filter
	if params.AccountID != "" {
		input.Filter = &awstypes.Expression{
			Dimensions: &awstypes.DimensionValues{
				Key:    awstypes.DimensionLinkedAccount,
				Values: []string{params.AccountID},
			},
		}
	}

	slog.Debug("fetching costs",
		"account", params.AccountID,
		"start", params.Start.Format("2006-01-02"),
		"end", params.End.Format("2006-01-02"),
	)

	output, err := p.ceClient.GetCostAndUsage(ctx, input)
	if err != nil {
		if IsAccessDenied(err) {
			slog.Warn("access denied fetching costs", "account", params.AccountID)
			return nil, fmt.Errorf("access denied for account %s: %w", params.AccountID, err)
		}
		return nil, fmt.Errorf("getting cost and usage: %w", err)
	}

	var records []provider.CostRecord
	for _, result := range output.ResultsByTime {
		periodStart := ""
		periodEnd := ""
		if result.TimePeriod != nil {
			periodStart = *result.TimePeriod.Start
			periodEnd = *result.TimePeriod.End
		}

		for _, group := range result.Groups {
			service := ""
			region := ""

			for i, key := range group.Keys {
				if i == 0 {
					service = key
				}
				if i == 1 {
					region = key
				}
			}

			amount := 0.0
			currency := "USD"
			if metric, ok := group.Metrics["UnblendedCost"]; ok {
				if metric.Amount != nil {
					amount, _ = strconv.ParseFloat(*metric.Amount, 64)
				}
				if metric.Unit != nil {
					currency = *metric.Unit
				}
			}

			// Skip zero-cost entries
			if amount == 0 {
				continue
			}

			accountID := params.AccountID
			if accountID == "" {
				accountID = p.accountID
			}

			records = append(records, provider.CostRecord{
				Provider:    "aws",
				AccountID:   accountID,
				Service:     service,
				Region:      region,
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,
				Granularity: params.Granularity,
				Amount:      amount,
				Currency:    currency,
			})
		}
	}

	slog.Info("fetched cost records", "count", len(records), "account", params.AccountID)
	return records, nil
}

// IsAccessDenied checks if an error is an AWS AccessDeniedException.
func IsAccessDenied(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AccessDeniedException"
	}
	return false
}
