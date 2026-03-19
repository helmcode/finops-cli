package aws

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	awstypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"

	"github.com/helmcode/finops-cli/internal/provider"
)

// FetchCommitments retrieves Savings Plans and Reserved Instance data.
func (p *AWSProvider) FetchCommitments(params provider.CommitmentParams) ([]provider.CommitmentRecord, error) {
	ctx := context.Background()

	startStr := params.Start.Format("2006-01-02")
	endStr := params.End.Format("2006-01-02")

	timePeriod := &awstypes.DateInterval{
		Start: strPtr(startStr),
		End:   strPtr(endStr),
	}

	// Build account filter if provided.
	var filter *awstypes.Expression
	if params.AccountID != "" {
		filter = &awstypes.Expression{
			Dimensions: &awstypes.DimensionValues{
				Key:    awstypes.DimensionLinkedAccount,
				Values: []string{params.AccountID},
			},
		}
	}

	accountID := params.AccountID
	if accountID == "" {
		accountID = p.accountID
	}

	// Fetch all four data sources, gracefully handling access denied errors.
	spUtil := p.fetchSPUtilization(ctx, timePeriod, filter)
	spCov := p.fetchSPCoverage(ctx, timePeriod, filter)
	riUtil := p.fetchRIUtilization(ctx, timePeriod, filter)
	riCov := p.fetchRICoverage(ctx, timePeriod, filter)

	// Merge SP utilization + coverage into records.
	spRecords := mergeSPRecords(spUtil, spCov, accountID)

	// Merge RI utilization + coverage into records.
	riRecords := mergeRIRecords(riUtil, riCov, accountID)

	records := make([]provider.CommitmentRecord, 0, len(spRecords)+len(riRecords))
	records = append(records, spRecords...)
	records = append(records, riRecords...)

	slog.Info("fetched commitment records", "count", len(records), "account", accountID)
	return records, nil
}

// spUtilPeriod holds parsed SP utilization data for a single time period.
type spUtilPeriod struct {
	PeriodStart    string
	PeriodEnd      string
	TotalCommitment float64
	UsedCommitment  float64
	UtilizationPct  float64
}

// spCovPeriod holds parsed SP coverage data for a single time period.
type spCovPeriod struct {
	PeriodStart    string
	PeriodEnd      string
	OnDemandCost   float64
	CoveragePct    float64
}

// riUtilPeriod holds parsed RI utilization data for a single time period.
type riUtilPeriod struct {
	PeriodStart    string
	PeriodEnd      string
	PurchasedHours float64
	ActualHours    float64
	UtilizationPct float64
	NetSavings     float64
}

// riCovPeriod holds parsed RI coverage data for a single time period.
type riCovPeriod struct {
	PeriodStart  string
	PeriodEnd    string
	OnDemandCost float64
	CoveragePct  float64
}

// fetchSPUtilization calls GetSavingsPlansUtilization and returns parsed period data.
func (p *AWSProvider) fetchSPUtilization(ctx context.Context, timePeriod *awstypes.DateInterval, filter *awstypes.Expression) []spUtilPeriod {
	input := &costexplorer.GetSavingsPlansUtilizationInput{
		TimePeriod:  timePeriod,
		Granularity: awstypes.GranularityMonthly,
		Filter:      filter,
	}

	output, err := p.ceClient.GetSavingsPlansUtilization(ctx, input)
	if err != nil {
		if IsAccessDenied(err) {
			slog.Warn("access denied fetching Savings Plans utilization, skipping")
			return nil
		}
		slog.Warn("error fetching Savings Plans utilization, skipping", "error", err)
		return nil
	}

	var periods []spUtilPeriod
	for _, result := range output.SavingsPlansUtilizationsByTime {
		period := spUtilPeriod{}
		if result.TimePeriod != nil {
			if result.TimePeriod.Start != nil {
				period.PeriodStart = *result.TimePeriod.Start
			}
			if result.TimePeriod.End != nil {
				period.PeriodEnd = *result.TimePeriod.End
			}
		}

		if result.Utilization != nil {
			period.TotalCommitment = parseFloat(result.Utilization.TotalCommitment)
			period.UsedCommitment = parseFloat(result.Utilization.UsedCommitment)
			period.UtilizationPct = parseFloat(result.Utilization.UtilizationPercentage)
		}

		periods = append(periods, period)
	}

	return periods
}

// fetchSPCoverage calls GetSavingsPlansCoverage and returns parsed period data.
func (p *AWSProvider) fetchSPCoverage(ctx context.Context, timePeriod *awstypes.DateInterval, filter *awstypes.Expression) []spCovPeriod {
	input := &costexplorer.GetSavingsPlansCoverageInput{
		TimePeriod:  timePeriod,
		Granularity: awstypes.GranularityMonthly,
		Filter:      filter,
	}

	output, err := p.ceClient.GetSavingsPlansCoverage(ctx, input)
	if err != nil {
		if IsAccessDenied(err) {
			slog.Warn("access denied fetching Savings Plans coverage, skipping")
			return nil
		}
		slog.Warn("error fetching Savings Plans coverage, skipping", "error", err)
		return nil
	}

	var periods []spCovPeriod
	for _, result := range output.SavingsPlansCoverages {
		period := spCovPeriod{}
		if result.TimePeriod != nil {
			if result.TimePeriod.Start != nil {
				period.PeriodStart = *result.TimePeriod.Start
			}
			if result.TimePeriod.End != nil {
				period.PeriodEnd = *result.TimePeriod.End
			}
		}

		if result.Coverage != nil {
			period.CoveragePct = parseFloat(result.Coverage.CoveragePercentage)
			if result.Coverage.SpendCoveredBySavingsPlans != nil {
				period.OnDemandCost = parseFloat(result.Coverage.OnDemandCost)
			}
		}

		periods = append(periods, period)
	}

	return periods
}

// fetchRIUtilization calls GetReservationUtilization and returns parsed period data.
func (p *AWSProvider) fetchRIUtilization(ctx context.Context, timePeriod *awstypes.DateInterval, filter *awstypes.Expression) []riUtilPeriod {
	input := &costexplorer.GetReservationUtilizationInput{
		TimePeriod:  timePeriod,
		Granularity: awstypes.GranularityMonthly,
		Filter:      filter,
	}

	output, err := p.ceClient.GetReservationUtilization(ctx, input)
	if err != nil {
		if IsAccessDenied(err) {
			slog.Warn("access denied fetching RI utilization, skipping")
			return nil
		}
		slog.Warn("error fetching RI utilization, skipping", "error", err)
		return nil
	}

	var periods []riUtilPeriod
	for _, result := range output.UtilizationsByTime {
		period := riUtilPeriod{}
		if result.TimePeriod != nil {
			if result.TimePeriod.Start != nil {
				period.PeriodStart = *result.TimePeriod.Start
			}
			if result.TimePeriod.End != nil {
				period.PeriodEnd = *result.TimePeriod.End
			}
		}

		if result.Total != nil {
			period.PurchasedHours = parseFloat(result.Total.PurchasedHours)
			period.ActualHours = parseFloat(result.Total.TotalActualHours)
			period.UtilizationPct = parseFloat(result.Total.UtilizationPercentage)
			period.NetSavings = parseFloat(result.Total.NetRISavings)
		}

		periods = append(periods, period)
	}

	return periods
}

// fetchRICoverage calls GetReservationCoverage and returns parsed period data.
func (p *AWSProvider) fetchRICoverage(ctx context.Context, timePeriod *awstypes.DateInterval, filter *awstypes.Expression) []riCovPeriod {
	input := &costexplorer.GetReservationCoverageInput{
		TimePeriod:  timePeriod,
		Granularity: awstypes.GranularityMonthly,
		Filter:      filter,
	}

	output, err := p.ceClient.GetReservationCoverage(ctx, input)
	if err != nil {
		if IsAccessDenied(err) {
			slog.Warn("access denied fetching RI coverage, skipping")
			return nil
		}
		slog.Warn("error fetching RI coverage, skipping", "error", err)
		return nil
	}

	var periods []riCovPeriod
	for _, result := range output.CoveragesByTime {
		period := riCovPeriod{}
		if result.TimePeriod != nil {
			if result.TimePeriod.Start != nil {
				period.PeriodStart = *result.TimePeriod.Start
			}
			if result.TimePeriod.End != nil {
				period.PeriodEnd = *result.TimePeriod.End
			}
		}

		if result.Total != nil {
			if result.Total.CoverageHours != nil {
				period.CoveragePct = parseFloat(result.Total.CoverageHours.CoverageHoursPercentage)
				period.OnDemandCost = parseFloat(result.Total.CoverageHours.OnDemandHours)
			}
		}

		periods = append(periods, period)
	}

	return periods
}

// mergeSPRecords combines SP utilization and coverage data into CommitmentRecords.
// It matches periods by PeriodStart to produce one record per period.
func mergeSPRecords(util []spUtilPeriod, cov []spCovPeriod, accountID string) []provider.CommitmentRecord {
	covMap := make(map[string]spCovPeriod, len(cov))
	for _, c := range cov {
		covMap[c.PeriodStart] = c
	}

	var records []provider.CommitmentRecord
	for _, u := range util {
		rec := provider.CommitmentRecord{
			Provider:        "aws",
			AccountID:       accountID,
			CommitmentType:  "savings_plan",
			PeriodStart:     u.PeriodStart,
			PeriodEnd:       u.PeriodEnd,
			TotalCommitment: u.TotalCommitment,
			UsedCommitment:  u.UsedCommitment,
			UtilizationPct:  u.UtilizationPct,
			Currency:        "USD",
		}

		if c, ok := covMap[u.PeriodStart]; ok {
			rec.OnDemandEquivalent = c.OnDemandCost
			rec.CoveragePct = c.CoveragePct
		}

		records = append(records, rec)
	}

	// Add coverage-only periods (periods with coverage data but no utilization).
	utilMap := make(map[string]struct{}, len(util))
	for _, u := range util {
		utilMap[u.PeriodStart] = struct{}{}
	}
	for _, c := range cov {
		if _, ok := utilMap[c.PeriodStart]; !ok {
			records = append(records, provider.CommitmentRecord{
				Provider:           "aws",
				AccountID:          accountID,
				CommitmentType:     "savings_plan",
				PeriodStart:        c.PeriodStart,
				PeriodEnd:          c.PeriodEnd,
				OnDemandEquivalent: c.OnDemandCost,
				CoveragePct:        c.CoveragePct,
				Currency:           "USD",
			})
		}
	}

	return records
}

// mergeRIRecords combines RI utilization and coverage data into CommitmentRecords.
// It matches periods by PeriodStart to produce one record per period.
func mergeRIRecords(util []riUtilPeriod, cov []riCovPeriod, accountID string) []provider.CommitmentRecord {
	covMap := make(map[string]riCovPeriod, len(cov))
	for _, c := range cov {
		covMap[c.PeriodStart] = c
	}

	var records []provider.CommitmentRecord
	for _, u := range util {
		rec := provider.CommitmentRecord{
			Provider:        "aws",
			AccountID:       accountID,
			CommitmentType:  "reserved_instance",
			PeriodStart:     u.PeriodStart,
			PeriodEnd:       u.PeriodEnd,
			TotalCommitment: u.PurchasedHours,
			UsedCommitment:  u.ActualHours,
			UtilizationPct:  u.UtilizationPct,
			NetSavings:      u.NetSavings,
			Currency:        "USD",
		}

		if c, ok := covMap[u.PeriodStart]; ok {
			rec.OnDemandEquivalent = c.OnDemandCost
			rec.CoveragePct = c.CoveragePct
		}

		records = append(records, rec)
	}

	// Add coverage-only periods (periods with coverage data but no utilization).
	utilMap := make(map[string]struct{}, len(util))
	for _, u := range util {
		utilMap[u.PeriodStart] = struct{}{}
	}
	for _, c := range cov {
		if _, ok := utilMap[c.PeriodStart]; !ok {
			records = append(records, provider.CommitmentRecord{
				Provider:           "aws",
				AccountID:          accountID,
				CommitmentType:     "reserved_instance",
				PeriodStart:        c.PeriodStart,
				PeriodEnd:          c.PeriodEnd,
				OnDemandEquivalent: c.OnDemandCost,
				CoveragePct:        c.CoveragePct,
				Currency:           "USD",
			})
		}
	}

	return records
}

// parseFloat safely parses a *string to float64, returning 0 on nil or error.
func parseFloat(s *string) float64 {
	if s == nil {
		return 0
	}
	v, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		slog.Warn("failed to parse float", "value", *s, "error", err)
		return 0
	}
	return v
}
