package report

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/helmcode/finops-cli/internal/analysis"
)

// GenerateSummaryCSV writes summary cost data to a CSV file.
func GenerateSummaryCSV(outputPath string, data *analysis.SummaryData) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Service", "Total Cost", "Currency"}); err != nil {
		return err
	}

	for _, svc := range data.TopServices {
		if err := w.Write([]string{
			svc.Service,
			fmt.Sprintf("%.2f", svc.TotalAmount),
			svc.Currency,
		}); err != nil {
			return err
		}
	}

	return nil
}

// GenerateTrendCSV writes trend data to a CSV file.
func GenerateTrendCSV(outputPath string, data *analysis.TrendData) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Period", "Amount"}); err != nil {
		return err
	}

	for _, dp := range data.DataPoints {
		if err := w.Write([]string{
			dp.Period,
			fmt.Sprintf("%.2f", dp.Amount),
		}); err != nil {
			return err
		}
	}

	return nil
}

// GenerateAnomaliesCSV writes anomaly data to a CSV file.
func GenerateAnomaliesCSV(outputPath string, data []analysis.AnomalyResult) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Period", "Service", "Expected", "Actual", "Deviation", "Severity"}); err != nil {
		return err
	}

	for _, a := range data {
		if err := w.Write([]string{
			a.Period,
			a.Service,
			fmt.Sprintf("%.2f", a.Expected),
			fmt.Sprintf("%.2f", a.Actual),
			fmt.Sprintf("%.2f", a.Deviation),
			string(a.Severity),
		}); err != nil {
			return err
		}
	}

	return nil
}

// GenerateCompareCSV writes comparison data to a CSV file.
func GenerateCompareCSV(outputPath string, data *analysis.CompareResult) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Service", "Previous", "Current", "Change", "% Change", "Currency"}); err != nil {
		return err
	}

	for _, d := range data.ServiceDeltas {
		if err := w.Write([]string{
			d.Service,
			fmt.Sprintf("%.2f", d.PreviousAmount),
			fmt.Sprintf("%.2f", d.CurrentAmount),
			fmt.Sprintf("%.2f", d.AbsoluteChange),
			fmt.Sprintf("%.1f", d.PercentChange),
			d.Currency,
		}); err != nil {
			return err
		}
	}

	return nil
}
