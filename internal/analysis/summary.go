package analysis

import (
	"context"
	"fmt"

	"github.com/helmcode/finops-cli/internal/store"
)

// ServiceCost represents cost data for a single service.
type ServiceCost struct {
	Service     string
	TotalAmount float64
	Currency    string
}

// RegionCost represents cost data for a single region.
type RegionCost struct {
	Region      string
	TotalAmount float64
	Currency    string
}

// ResourceCount represents the count of resources for a service.
type ResourceCount struct {
	Service string
	Count   int64
}

// SummaryData contains all aggregated data for a summary report.
type SummaryData struct {
	TotalSpend     float64
	Currency       string
	TopServices    []ServiceCost
	CostByRegion   []RegionCost
	ResourceCounts []ResourceCount
	TrendChange    float64 // Percentage change vs previous period
	PeriodStart    string
	PeriodEnd      string
}

// DateRange defines a time range for analysis.
type DateRange struct {
	Start string // "2026-01-01"
	End   string // "2026-03-01"
}

// GenerateSummary aggregates cost data for the given date range.
func GenerateSummary(q *store.Queries, provider string, dr DateRange) (*SummaryData, error) {
	ctx := context.Background()

	// Get top services by cost
	serviceRows, err := q.GetTotalCostByService(ctx, store.GetTotalCostByServiceParams{
		Provider:    provider,
		PeriodStart: dr.Start,
		PeriodEnd:   dr.End,
	})
	if err != nil {
		return nil, fmt.Errorf("getting costs by service: %w", err)
	}

	var topServices []ServiceCost
	totalSpend := 0.0
	currency := "USD"
	for _, row := range serviceRows {
		amount := 0.0
		if row.TotalAmount.Valid {
			amount = row.TotalAmount.Float64
		}
		topServices = append(topServices, ServiceCost{
			Service:     row.Service,
			TotalAmount: amount,
			Currency:    row.Currency,
		})
		totalSpend += amount
		currency = row.Currency
	}

	// Get costs by region
	regionRows, err := q.GetTotalCostByRegion(ctx, store.GetTotalCostByRegionParams{
		Provider:    provider,
		PeriodStart: dr.Start,
		PeriodEnd:   dr.End,
	})
	if err != nil {
		return nil, fmt.Errorf("getting costs by region: %w", err)
	}

	var costByRegion []RegionCost
	for _, row := range regionRows {
		region := ""
		if row.Region.Valid {
			region = row.Region.String
		}
		amount := 0.0
		if row.TotalAmount.Valid {
			amount = row.TotalAmount.Float64
		}
		costByRegion = append(costByRegion, RegionCost{
			Region:      region,
			TotalAmount: amount,
			Currency:    row.Currency,
		})
	}

	// Get resource counts by service
	countRows, err := q.CountResourcesByService(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("counting resources by service: %w", err)
	}

	var resourceCounts []ResourceCount
	for _, row := range countRows {
		resourceCounts = append(resourceCounts, ResourceCount{
			Service: row.Service,
			Count:   row.Count,
		})
	}

	return &SummaryData{
		TotalSpend:     totalSpend,
		Currency:       currency,
		TopServices:    topServices,
		CostByRegion:   costByRegion,
		ResourceCounts: resourceCounts,
		PeriodStart:    dr.Start,
		PeriodEnd:      dr.End,
	}, nil
}
