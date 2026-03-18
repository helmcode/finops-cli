package analysis

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmcode/finops-cli/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.OpenAt(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func seedCostData(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()

	records := []store.UpsertCostRecordParams{
		{Provider: "aws", AccountID: "123", Service: "Amazon EC2", Region: sql.NullString{String: "us-east-1", Valid: true}, PeriodStart: "2026-01-01", PeriodEnd: "2026-02-01", Granularity: "MONTHLY", Amount: 1000.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"},
		{Provider: "aws", AccountID: "123", Service: "Amazon EC2", Region: sql.NullString{String: "us-east-1", Valid: true}, PeriodStart: "2026-02-01", PeriodEnd: "2026-03-01", Granularity: "MONTHLY", Amount: 1200.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"},
		{Provider: "aws", AccountID: "123", Service: "Amazon EC2", Region: sql.NullString{String: "us-east-1", Valid: true}, PeriodStart: "2026-03-01", PeriodEnd: "2026-04-01", Granularity: "MONTHLY", Amount: 5000.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"}, // anomaly
		{Provider: "aws", AccountID: "123", Service: "Amazon RDS", Region: sql.NullString{String: "us-east-1", Valid: true}, PeriodStart: "2026-01-01", PeriodEnd: "2026-02-01", Granularity: "MONTHLY", Amount: 500.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"},
		{Provider: "aws", AccountID: "123", Service: "Amazon RDS", Region: sql.NullString{String: "us-east-1", Valid: true}, PeriodStart: "2026-02-01", PeriodEnd: "2026-03-01", Granularity: "MONTHLY", Amount: 500.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"},
		{Provider: "aws", AccountID: "123", Service: "Amazon RDS", Region: sql.NullString{String: "us-east-1", Valid: true}, PeriodStart: "2026-03-01", PeriodEnd: "2026-04-01", Granularity: "MONTHLY", Amount: 550.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"},
		{Provider: "aws", AccountID: "123", Service: "Amazon S3", Region: sql.NullString{String: "us-west-2", Valid: true}, PeriodStart: "2026-01-01", PeriodEnd: "2026-02-01", Granularity: "MONTHLY", Amount: 50.0, Currency: "USD", SyncedAt: "2026-03-18T00:00:00Z"},
	}

	for _, r := range records {
		err := s.Queries.UpsertCostRecord(ctx, r)
		require.NoError(t, err)
	}
}

func TestGenerateSummary(t *testing.T) {
	s := setupTestStore(t)
	seedCostData(t, s)

	dr := DateRange{Start: "2026-01-01", End: "2026-04-01"}
	summary, err := GenerateSummary(s.Queries, "aws", dr)
	require.NoError(t, err)

	assert.Greater(t, summary.TotalSpend, 0.0)
	assert.Equal(t, "USD", summary.Currency)
	assert.NotEmpty(t, summary.TopServices)

	// EC2 should be the top service
	assert.Equal(t, "Amazon EC2", summary.TopServices[0].Service)

	// Should have region data
	assert.NotEmpty(t, summary.CostByRegion)
}

func TestGenerateTrend_AllServices(t *testing.T) {
	s := setupTestStore(t)
	seedCostData(t, s)

	trend, err := GenerateTrend(s.Queries, "aws", "")
	require.NoError(t, err)

	assert.Empty(t, trend.Service)
	assert.NotEmpty(t, trend.DataPoints)
	assert.Greater(t, trend.AvgMonthly, 0.0)
}

func TestGenerateTrend_ByService(t *testing.T) {
	s := setupTestStore(t)
	seedCostData(t, s)

	trend, err := GenerateTrend(s.Queries, "aws", "Amazon EC2")
	require.NoError(t, err)

	assert.Equal(t, "Amazon EC2", trend.Service)
	assert.Len(t, trend.DataPoints, 3)
	assert.Equal(t, TrendUp, trend.Direction) // 1000 → 1200 → 5000
}

func TestCalculateDirection(t *testing.T) {
	tests := []struct {
		name     string
		points   []MonthlyDataPoint
		expected TrendDirection
	}{
		{"empty", nil, TrendFlat},
		{"single", []MonthlyDataPoint{{Amount: 100}}, TrendFlat},
		{"up", []MonthlyDataPoint{{Amount: 100}, {Amount: 200}}, TrendUp},
		{"down", []MonthlyDataPoint{{Amount: 200}, {Amount: 100}}, TrendDown},
		{"flat", []MonthlyDataPoint{{Amount: 100}, {Amount: 102}}, TrendFlat},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, calculateDirection(tc.points))
		})
	}
}

func TestDetectAnomalies(t *testing.T) {
	s := setupTestStore(t)
	seedCostData(t, s)

	dr := DateRange{Start: "2026-01-01", End: "2026-04-01"}
	anomalies, err := DetectAnomalies(s.Queries, "aws", dr, 1.0)
	require.NoError(t, err)

	// EC2's March spike (5000 vs avg ~2400) should be detected
	found := false
	for _, a := range anomalies {
		if a.Service == "Amazon EC2" && a.Period == "2026-03-01" {
			found = true
			assert.Equal(t, 5000.0, a.Actual)
			assert.Greater(t, a.Deviation, 0.0)
		}
	}
	assert.True(t, found, "expected anomaly for EC2 in March not found")
}

func TestComparePeriods(t *testing.T) {
	s := setupTestStore(t)
	seedCostData(t, s)

	current := DateRange{Start: "2026-02-01", End: "2026-04-01"}
	previous := DateRange{Start: "2026-01-01", End: "2026-02-01"}

	result, err := ComparePeriods(s.Queries, "aws", current, previous)
	require.NoError(t, err)

	assert.Greater(t, result.TotalCurrent, 0.0)
	assert.Greater(t, result.TotalPrevious, 0.0)
	assert.NotEmpty(t, result.ServiceDeltas)

	// Total current should be more than previous (EC2 grew significantly)
	assert.Greater(t, result.TotalCurrent, result.TotalPrevious)
	assert.Greater(t, result.TotalPercent, 0.0)
}

func TestCalculateMean(t *testing.T) {
	assert.Equal(t, 0.0, calculateMean(nil))
	assert.Equal(t, 2.0, calculateMean([]float64{1, 2, 3}))
	assert.Equal(t, 5.0, calculateMean([]float64{5}))
}

func TestCalculateStdDev(t *testing.T) {
	assert.Equal(t, 0.0, calculateStdDev(nil, 0))
	assert.Equal(t, 0.0, calculateStdDev([]float64{5}, 5))
	// stddev of [1, 2, 3] with mean 2 = sqrt(2/3) ≈ 0.8165
	stddev := calculateStdDev([]float64{1, 2, 3}, 2)
	assert.InDelta(t, 0.8165, stddev, 0.01)
}
