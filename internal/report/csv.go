package report

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/helmcode/finops-cli/internal/analysis"
	"github.com/helmcode/finops-cli/internal/store"
)

// GenerateSummaryCSV writes detailed cost data to a CSV file with account and region breakdown.
func GenerateSummaryCSV(outputPath string, rows []store.GetCostByAccountAndServiceRow) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"Account ID", "Service", "Region", "Total Cost", "Currency"}); err != nil {
		return err
	}

	for _, row := range rows {
		amount := 0.0
		if row.TotalAmount.Valid {
			amount = row.TotalAmount.Float64
		}
		region := ""
		if row.Region.Valid {
			region = row.Region.String
		}
		if err := w.Write([]string{
			row.AccountID,
			row.Service,
			region,
			fmt.Sprintf("%.2f", amount),
			row.Currency,
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
