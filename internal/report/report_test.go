package report

import (
	database_sql "database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmcode/finops-cli/internal/analysis"
	"github.com/helmcode/finops-cli/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHTML_Summary(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "summary.html")

	data := ReportData{
		Title:       "Cost Summary",
		PeriodStart: "2026-01-01",
		PeriodEnd:   "2026-03-01",
		Data: &analysis.SummaryData{
			TotalSpend: 5000.0,
			Currency:   "USD",
			TopServices: []analysis.ServiceCost{
				{Service: "Amazon EC2", TotalAmount: 3000.0, Currency: "USD"},
				{Service: "Amazon RDS", TotalAmount: 1500.0, Currency: "USD"},
				{Service: "Amazon S3", TotalAmount: 500.0, Currency: "USD"},
			},
			CostByRegion: []analysis.RegionCost{
				{Region: "us-east-1", TotalAmount: 4000.0, Currency: "USD"},
				{Region: "eu-west-1", TotalAmount: 1000.0, Currency: "USD"},
			},
		},
		TotalResources: 42,
		MonthCount:     2,
	}

	err := GenerateHTML("summary", outputPath, data)
	require.NoError(t, err)

	// Verify file exists and has content
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// Read and check content
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Cost Summary")
	assert.Contains(t, string(content), "Amazon EC2")
	assert.Contains(t, string(content), "5,000.00")
}

func TestGenerateHTML_Trend(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "trend.html")

	data := ReportData{
		Title:       "Cost Trend",
		PeriodStart: "2026-01-01",
		PeriodEnd:   "2026-03-01",
		Data: &analysis.TrendData{
			Service: "Amazon EC2",
			DataPoints: []analysis.MonthlyDataPoint{
				{Period: "2026-01-01", Amount: 1000},
				{Period: "2026-02-01", Amount: 1200},
			},
			Direction:  analysis.TrendUp,
			AvgMonthly: 1100,
		},
	}

	err := GenerateHTML("trend", outputPath, data)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Amazon EC2")
	assert.Contains(t, string(content), "1,100.00")
}

func TestGenerateSummaryCSV(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "summary.csv")

	rows := []store.GetCostByAccountAndServiceRow{
		{AccountID: "123456789012", Service: "Amazon EC2", Region: database_sql.NullString{String: "us-east-1", Valid: true}, TotalAmount: database_sql.NullFloat64{Float64: 3000.0, Valid: true}, Currency: "USD"},
		{AccountID: "123456789012", Service: "Amazon RDS", Region: database_sql.NullString{String: "us-west-2", Valid: true}, TotalAmount: database_sql.NullFloat64{Float64: 1500.0, Valid: true}, Currency: "USD"},
	}

	err := GenerateSummaryCSV(outputPath, rows)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Account ID,Service,Region,Total Cost,Currency")
	assert.Contains(t, string(content), "123456789012,Amazon EC2,us-east-1,3000.00,USD")
}

func TestGenerateTrendCSV(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "trend.csv")

	data := &analysis.TrendData{
		DataPoints: []analysis.MonthlyDataPoint{
			{Period: "2026-01-01", Amount: 1000},
			{Period: "2026-02-01", Amount: 1200},
		},
	}

	err := GenerateTrendCSV(outputPath, data)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Period,Amount")
	assert.Contains(t, string(content), "2026-01-01,1000.00")
}

func TestGenerateAnomaliesCSV(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "anomalies.csv")

	data := []analysis.AnomalyResult{
		{Period: "2026-03-01", Service: "Amazon EC2", Expected: 1100, Actual: 5000, Deviation: 2.5, Severity: analysis.SeverityHigh},
	}

	err := GenerateAnomaliesCSV(outputPath, data)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Period,Service,Expected,Actual,Deviation,Severity")
	assert.Contains(t, string(content), "Amazon EC2")
}

func TestGenerateCompareCSV(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "compare.csv")

	data := &analysis.CompareResult{
		ServiceDeltas: []analysis.ServiceDelta{
			{Service: "Amazon EC2", PreviousAmount: 1000, CurrentAmount: 1500, AbsoluteChange: 500, PercentChange: 50.0, Currency: "USD"},
		},
	}

	err := GenerateCompareCSV(outputPath, data)
	require.NoError(t, err)

	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Service,Previous,Current,Change,% Change,Currency")
	assert.Contains(t, string(content), "Amazon EC2")
}
