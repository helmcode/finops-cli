package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	awstypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmcode/finops-cli/internal/provider"
)

// Mock Cost Explorer client
type mockCostExplorer struct {
	output *costexplorer.GetCostAndUsageOutput
	err    error
}

func (m *mockCostExplorer) GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.output, nil
}

func (m *mockCostExplorer) GetSavingsPlansUtilization(ctx context.Context, params *costexplorer.GetSavingsPlansUtilizationInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetSavingsPlansUtilizationOutput, error) {
	return &costexplorer.GetSavingsPlansUtilizationOutput{}, nil
}

func (m *mockCostExplorer) GetSavingsPlansCoverage(ctx context.Context, params *costexplorer.GetSavingsPlansCoverageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetSavingsPlansCoverageOutput, error) {
	return &costexplorer.GetSavingsPlansCoverageOutput{}, nil
}

func (m *mockCostExplorer) GetReservationUtilization(ctx context.Context, params *costexplorer.GetReservationUtilizationInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetReservationUtilizationOutput, error) {
	return &costexplorer.GetReservationUtilizationOutput{}, nil
}

func (m *mockCostExplorer) GetReservationCoverage(ctx context.Context, params *costexplorer.GetReservationCoverageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetReservationCoverageOutput, error) {
	return &costexplorer.GetReservationCoverageOutput{}, nil
}

func TestValidateDateRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr string
	}{
		{
			name:  "valid range",
			start: now.AddDate(0, -6, 0),
			end:   now,
		},
		{
			name:    "start after end",
			start:   now,
			end:     now.AddDate(0, -6, 0),
			wantErr: "--from must be before --to",
		},
		{
			name:    "start in future",
			start:   now.AddDate(0, 1, 0),
			end:     now.AddDate(0, 2, 0),
			wantErr: "--from cannot be in the future",
		},
		{
			name:    "exceeds 12 months",
			start:   now.AddDate(-2, 0, 0),
			end:     now,
			wantErr: "range exceeds 12 months",
		},
		{
			name:  "exactly 12 months",
			start: now.AddDate(-1, 0, 0),
			end:   now,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDateRange(tc.start, tc.end)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFetchCosts_Success(t *testing.T) {
	start := "2026-01-01"
	end := "2026-02-01"
	svc := "Amazon Elastic Compute Cloud"
	region := "us-east-1"
	amount := "1234.56"
	unit := "USD"

	p := &AWSProvider{
		accountID: "123456789012",
		ceClient: &mockCostExplorer{
			output: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []awstypes.ResultByTime{
					{
						TimePeriod: &awstypes.DateInterval{
							Start: &start,
							End:   &end,
						},
						Groups: []awstypes.Group{
							{
								Keys: []string{svc, region},
								Metrics: map[string]awstypes.MetricValue{
									"UnblendedCost": {
										Amount: &amount,
										Unit:   &unit,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	now := time.Now()
	records, err := p.FetchCosts(provider.CostParams{
		AccountID:   "123456789012",
		Start:       now.AddDate(0, -6, 0),
		End:         now,
		Granularity: "MONTHLY",
		GroupBy:     []string{"SERVICE", "REGION"},
	})

	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "Amazon Elastic Compute Cloud", records[0].Service)
	assert.Equal(t, "us-east-1", records[0].Region)
	assert.Equal(t, 1234.56, records[0].Amount)
	assert.Equal(t, "USD", records[0].Currency)
}

func TestFetchCosts_SkipsZeroCost(t *testing.T) {
	start := "2026-01-01"
	end := "2026-02-01"
	svc := "Amazon S3"
	region := "us-east-1"
	zeroAmount := "0.0"
	unit := "USD"

	p := &AWSProvider{
		accountID: "123456789012",
		ceClient: &mockCostExplorer{
			output: &costexplorer.GetCostAndUsageOutput{
				ResultsByTime: []awstypes.ResultByTime{
					{
						TimePeriod: &awstypes.DateInterval{
							Start: &start,
							End:   &end,
						},
						Groups: []awstypes.Group{
							{
								Keys: []string{svc, region},
								Metrics: map[string]awstypes.MetricValue{
									"UnblendedCost": {
										Amount: &zeroAmount,
										Unit:   &unit,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	now := time.Now()
	records, err := p.FetchCosts(provider.CostParams{
		AccountID:   "123456789012",
		Start:       now.AddDate(0, -6, 0),
		End:         now,
		Granularity: "MONTHLY",
		GroupBy:     []string{"SERVICE", "REGION"},
	})

	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestFetchCosts_AccessDenied(t *testing.T) {
	p := &AWSProvider{
		accountID: "123456789012",
		ceClient: &mockCostExplorer{
			err: &smithy.GenericAPIError{Code: "AccessDeniedException", Message: "not authorized"},
		},
	}

	now := time.Now()
	_, err := p.FetchCosts(provider.CostParams{
		AccountID:   "999999999999",
		Start:       now.AddDate(0, -6, 0),
		End:         now,
		Granularity: "MONTHLY",
		GroupBy:     []string{"SERVICE", "REGION"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

func TestFetchCosts_InvalidRange(t *testing.T) {
	p := &AWSProvider{
		accountID: "123456789012",
		ceClient:  &mockCostExplorer{},
	}

	now := time.Now()
	_, err := p.FetchCosts(provider.CostParams{
		AccountID:   "123456789012",
		Start:       now.AddDate(-2, 0, 0),
		End:         now,
		Granularity: "MONTHLY",
		GroupBy:     []string{"SERVICE", "REGION"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "range exceeds 12 months")
}

func TestIsAccessDenied(t *testing.T) {
	assert.True(t, IsAccessDenied(&smithy.GenericAPIError{Code: "AccessDeniedException"}))
	assert.False(t, IsAccessDenied(&smithy.GenericAPIError{Code: "InternalServerError"}))
}
