package analysis

import (
	"context"
	"fmt"

	"github.com/helmcode/finops-cli/internal/store"
)

// MonthlyDataPoint represents a single data point in a time series.
type MonthlyDataPoint struct {
	Period string
	Amount float64
}

// TrendDirection indicates the cost trend.
type TrendDirection string

const (
	TrendUp   TrendDirection = "up"
	TrendDown TrendDirection = "down"
	TrendFlat TrendDirection = "flat"
)

// TrendData contains trend information for cost analysis.
type TrendData struct {
	Service    string
	DataPoints []MonthlyDataPoint
	Direction  TrendDirection
	AvgMonthly float64
}

// GenerateTrend calculates cost trend for a specific service or all services.
func GenerateTrend(q *store.Queries, provider, service string) (*TrendData, error) {
	ctx := context.Background()

	var dataPoints []MonthlyDataPoint

	if service != "" {
		rows, err := q.GetMonthlyCostTrendByService(ctx, store.GetMonthlyCostTrendByServiceParams{
			Provider: provider,
			Service:  service,
		})
		if err != nil {
			return nil, fmt.Errorf("getting trend by service: %w", err)
		}
		for _, row := range rows {
			amount := 0.0
			if row.TotalAmount.Valid {
				amount = row.TotalAmount.Float64
			}
			dataPoints = append(dataPoints, MonthlyDataPoint{
				Period: row.PeriodStart,
				Amount: amount,
			})
		}
	} else {
		rows, err := q.GetMonthlyCostTrend(ctx, provider)
		if err != nil {
			return nil, fmt.Errorf("getting overall trend: %w", err)
		}
		for _, row := range rows {
			amount := 0.0
			if row.TotalAmount.Valid {
				amount = row.TotalAmount.Float64
			}
			dataPoints = append(dataPoints, MonthlyDataPoint{
				Period: row.PeriodStart,
				Amount: amount,
			})
		}
	}

	direction := calculateDirection(dataPoints)
	avg := calculateAverage(dataPoints)

	return &TrendData{
		Service:    service,
		DataPoints: dataPoints,
		Direction:  direction,
		AvgMonthly: avg,
	}, nil
}

func calculateDirection(points []MonthlyDataPoint) TrendDirection {
	if len(points) < 2 {
		return TrendFlat
	}

	last := points[len(points)-1].Amount
	prev := points[len(points)-2].Amount

	if prev == 0 {
		if last > 0 {
			return TrendUp
		}
		return TrendFlat
	}

	change := (last - prev) / prev
	switch {
	case change > 0.05:
		return TrendUp
	case change < -0.05:
		return TrendDown
	default:
		return TrendFlat
	}
}

func calculateAverage(points []MonthlyDataPoint) float64 {
	if len(points) == 0 {
		return 0
	}

	total := 0.0
	for _, p := range points {
		total += p.Amount
	}
	return total / float64(len(points))
}
