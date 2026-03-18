package analysis

import (
	"context"
	"fmt"
	"math"

	"github.com/helmcode/finops-cli/internal/store"
)

// AnomalySeverity indicates how significant the anomaly is.
type AnomalySeverity string

const (
	SeverityLow    AnomalySeverity = "low"
	SeverityMedium AnomalySeverity = "medium"
	SeverityHigh   AnomalySeverity = "high"
)

// AnomalyResult represents a detected cost anomaly.
type AnomalyResult struct {
	Period    string
	Service   string
	Expected  float64
	Actual    float64
	Deviation float64 // Z-score
	Severity  AnomalySeverity
}

// DetectAnomalies finds cost spikes using z-score over moving average.
// threshold is the z-score cutoff (e.g., 2.0 for ~95% confidence).
func DetectAnomalies(q *store.Queries, provider string, dr DateRange, threshold float64) ([]AnomalyResult, error) {
	ctx := context.Background()

	// Get all cost records by service
	services, err := q.GetDistinctServices(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("getting distinct services: %w", err)
	}

	var anomalies []AnomalyResult

	for _, service := range services {
		rows, err := q.GetMonthlyCostTrendByService(ctx, store.GetMonthlyCostTrendByServiceParams{
			Provider: provider,
			Service:  service,
		})
		if err != nil {
			return nil, fmt.Errorf("getting trend for %s: %w", service, err)
		}

		if len(rows) < 3 {
			continue // Need at least 3 data points for meaningful detection
		}

		amounts := make([]float64, len(rows))
		periods := make([]string, len(rows))
		for i, row := range rows {
			if row.TotalAmount.Valid {
				amounts[i] = row.TotalAmount.Float64
			}
			periods[i] = row.PeriodStart
		}

		// Calculate z-scores using moving average and standard deviation
		detected := detectWithZScore(amounts, periods, service, threshold)
		anomalies = append(anomalies, detected...)
	}

	return anomalies, nil
}

func detectWithZScore(amounts []float64, periods []string, service string, threshold float64) []AnomalyResult {
	n := len(amounts)
	if n < 3 {
		return nil
	}

	mean := calculateMean(amounts)
	stddev := calculateStdDev(amounts, mean)

	if stddev == 0 {
		return nil // No variance, no anomalies
	}

	var results []AnomalyResult
	for i, amount := range amounts {
		zScore := (amount - mean) / stddev

		if math.Abs(zScore) >= threshold {
			severity := SeverityLow
			if math.Abs(zScore) >= threshold*2 {
				severity = SeverityHigh
			} else if math.Abs(zScore) >= threshold*1.5 {
				severity = SeverityMedium
			}

			results = append(results, AnomalyResult{
				Period:    periods[i],
				Service:   service,
				Expected:  mean,
				Actual:    amount,
				Deviation: zScore,
				Severity:  severity,
			})
		}
	}

	return results
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return math.Sqrt(sumSquares / float64(len(values)))
}
