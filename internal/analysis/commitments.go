package analysis

import (
	"context"

	"github.com/helmcode/finops-cli/internal/store"
)

// CommitmentTypeSummary holds aggregated data for one commitment type.
type CommitmentTypeSummary struct {
	Type               string  // "savings_plan" or "reserved_instance"
	TotalCommitment    float64
	UsedCommitment     float64
	OnDemandEquivalent float64
	NetSavings         float64
	Currency           string
}

// CommitmentOverview holds the full commitment analysis.
type CommitmentOverview struct {
	TotalCommitted    float64
	TotalUsed         float64
	TotalSavings      float64
	AvgUtilization    float64
	Currency          string
	Types             []CommitmentTypeSummary
	SpotInstanceCount int64
	HasData           bool
	PermissionWarning bool
}

// GenerateCommitmentOverview aggregates commitment data for the report.
func GenerateCommitmentOverview(q *store.Queries, provider string, dr DateRange) (*CommitmentOverview, error) {
	ctx := context.Background()

	rows, err := q.GetCommitmentSummary(ctx, store.GetCommitmentSummaryParams{
		Provider:    provider,
		PeriodStart: dr.Start,
		PeriodEnd:   dr.End,
	})
	if err != nil {
		return &CommitmentOverview{HasData: false}, nil
	}

	overview := &CommitmentOverview{
		Currency: "USD",
	}

	for _, row := range rows {
		totalCommit := 0.0
		if row.TotalCommitment.Valid {
			totalCommit = row.TotalCommitment.Float64
		}
		usedCommit := 0.0
		if row.UsedCommitment.Valid {
			usedCommit = row.UsedCommitment.Float64
		}
		onDemand := 0.0
		if row.OnDemandEquivalent.Valid {
			onDemand = row.OnDemandEquivalent.Float64
		}
		savings := 0.0
		if row.NetSavings.Valid {
			savings = row.NetSavings.Float64
		}

		overview.Types = append(overview.Types, CommitmentTypeSummary{
			Type:               row.CommitmentType,
			TotalCommitment:    totalCommit,
			UsedCommitment:     usedCommit,
			OnDemandEquivalent: onDemand,
			NetSavings:         savings,
			Currency:           row.Currency,
		})

		overview.TotalCommitted += totalCommit
		overview.TotalUsed += usedCommit
		overview.TotalSavings += savings
		overview.Currency = row.Currency
	}

	if overview.TotalCommitted > 0 {
		overview.AvgUtilization = (overview.TotalUsed / overview.TotalCommitted) * 100
	}

	// Count spot instances
	spotCount, err := q.CountSpotInstances(ctx, provider)
	if err == nil {
		overview.SpotInstanceCount = spotCount
	}

	// Mark as having data if we have any commitment types or spot instances
	if len(overview.Types) > 0 || overview.SpotInstanceCount > 0 {
		overview.HasData = true
	}

	return overview, nil
}
