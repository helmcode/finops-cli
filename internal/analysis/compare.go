package analysis

import (
	"context"
	"fmt"

	"github.com/helmcode/finops-cli/internal/store"
)

// ServiceDelta represents the cost change for a single service between two periods.
type ServiceDelta struct {
	Service         string
	PreviousAmount  float64
	CurrentAmount   float64
	AbsoluteChange  float64
	PercentChange   float64
	Currency        string
}

// CompareResult contains the comparison of two periods.
type CompareResult struct {
	CurrentPeriod   DateRange
	PreviousPeriod  DateRange
	TotalPrevious   float64
	TotalCurrent    float64
	TotalChange     float64
	TotalPercent    float64
	ServiceDeltas   []ServiceDelta
	Currency        string
}

// ComparePeriods compares costs between two date ranges.
func ComparePeriods(q *store.Queries, provider string, current, previous DateRange) (*CompareResult, error) {
	ctx := context.Background()

	// Get costs for current period
	currentRows, err := q.GetTotalCostByService(ctx, store.GetTotalCostByServiceParams{
		Provider:    provider,
		PeriodStart: current.Start,
		PeriodEnd:   current.End,
	})
	if err != nil {
		return nil, fmt.Errorf("getting current period costs: %w", err)
	}

	currentByService := make(map[string]float64)
	totalCurrent := 0.0
	currency := "USD"
	for _, row := range currentRows {
		amount := 0.0
		if row.TotalAmount.Valid {
			amount = row.TotalAmount.Float64
		}
		currentByService[row.Service] = amount
		totalCurrent += amount
		currency = row.Currency
	}

	// Get costs for previous period
	previousRows, err := q.GetTotalCostByService(ctx, store.GetTotalCostByServiceParams{
		Provider:    provider,
		PeriodStart: previous.Start,
		PeriodEnd:   previous.End,
	})
	if err != nil {
		return nil, fmt.Errorf("getting previous period costs: %w", err)
	}

	previousByService := make(map[string]float64)
	totalPrevious := 0.0
	for _, row := range previousRows {
		amount := 0.0
		if row.TotalAmount.Valid {
			amount = row.TotalAmount.Float64
		}
		previousByService[row.Service] = amount
		totalPrevious += amount
	}

	// Build service deltas (union of both periods)
	allServices := make(map[string]bool)
	for svc := range currentByService {
		allServices[svc] = true
	}
	for svc := range previousByService {
		allServices[svc] = true
	}

	var deltas []ServiceDelta
	for svc := range allServices {
		cur := currentByService[svc]
		prev := previousByService[svc]
		change := cur - prev
		pctChange := 0.0
		if prev != 0 {
			pctChange = (change / prev) * 100
		}

		deltas = append(deltas, ServiceDelta{
			Service:        svc,
			PreviousAmount: prev,
			CurrentAmount:  cur,
			AbsoluteChange: change,
			PercentChange:  pctChange,
			Currency:       currency,
		})
	}

	totalChange := totalCurrent - totalPrevious
	totalPercent := 0.0
	if totalPrevious != 0 {
		totalPercent = (totalChange / totalPrevious) * 100
	}

	return &CompareResult{
		CurrentPeriod:  current,
		PreviousPeriod: previous,
		TotalPrevious:  totalPrevious,
		TotalCurrent:   totalCurrent,
		TotalChange:    totalChange,
		TotalPercent:   totalPercent,
		ServiceDeltas:  deltas,
		Currency:       currency,
	}, nil
}
