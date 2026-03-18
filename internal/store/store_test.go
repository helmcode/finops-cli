package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := OpenAt(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenAt(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")
	s, err := OpenAt(dbPath)
	require.NoError(t, err)
	defer s.Close()

	// Verify directory was created
	_, err = os.Stat(filepath.Dir(dbPath))
	assert.NoError(t, err)

	// Verify database file exists
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestRunMigrations(t *testing.T) {
	s := setupTestDB(t)

	// Verify tables exist by querying them
	var count int64
	err := s.DB.QueryRow("SELECT COUNT(*) FROM cost_records").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	err = s.DB.QueryRow("SELECT COUNT(*) FROM resources").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	err = s.DB.QueryRow("SELECT COUNT(*) FROM sync_history").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	err = s.DB.QueryRow("SELECT COUNT(*) FROM config").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestUpsertCostRecord(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	err := s.Queries.UpsertCostRecord(ctx, UpsertCostRecordParams{
		Provider:    "aws",
		AccountID:   "123456789012",
		Service:     "Amazon Elastic Compute Cloud",
		Region:      sql.NullString{String: "us-east-1", Valid: true},
		PeriodStart: "2026-01-01",
		PeriodEnd:   "2026-02-01",
		Granularity: "MONTHLY",
		Amount:      1234.56,
		Currency:    "USD",
		SyncedAt:    "2026-03-18T00:00:00Z",
	})
	require.NoError(t, err)

	// Verify insert
	records, err := s.Queries.GetCostRecordsByProvider(ctx, "aws")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "Amazon Elastic Compute Cloud", records[0].Service)
	assert.Equal(t, 1234.56, records[0].Amount)

	// Upsert same record with updated amount
	err = s.Queries.UpsertCostRecord(ctx, UpsertCostRecordParams{
		Provider:    "aws",
		AccountID:   "123456789012",
		Service:     "Amazon Elastic Compute Cloud",
		Region:      sql.NullString{String: "us-east-1", Valid: true},
		PeriodStart: "2026-01-01",
		PeriodEnd:   "2026-02-01",
		Granularity: "MONTHLY",
		Amount:      2000.00,
		Currency:    "USD",
		SyncedAt:    "2026-03-18T01:00:00Z",
	})
	require.NoError(t, err)

	// Verify upsert updated the record
	records, err = s.Queries.GetCostRecordsByProvider(ctx, "aws")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, 2000.00, records[0].Amount)
}

func TestGetTotalCostByService(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	// Insert multiple cost records
	services := []struct {
		name   string
		amount float64
	}{
		{"Amazon EC2", 1000.00},
		{"Amazon EC2", 1200.00},
		{"Amazon RDS", 500.00},
		{"Amazon S3", 50.00},
	}

	for i, svc := range services {
		period := "2026-01-01"
		if i == 1 {
			period = "2026-02-01"
		}
		err := s.Queries.UpsertCostRecord(ctx, UpsertCostRecordParams{
			Provider:    "aws",
			AccountID:   "123456789012",
			Service:     svc.name,
			Region:      sql.NullString{String: "us-east-1", Valid: true},
			PeriodStart: period,
			PeriodEnd:   "2026-03-01",
			Granularity: "MONTHLY",
			Amount:      svc.amount,
			Currency:    "USD",
			SyncedAt:    "2026-03-18T00:00:00Z",
		})
		require.NoError(t, err)
	}

	results, err := s.Queries.GetTotalCostByService(ctx, GetTotalCostByServiceParams{
		Provider:    "aws",
		PeriodStart: "2026-01-01",
		PeriodEnd:   "2026-03-01",
	})
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Results ordered by total_amount DESC
	assert.Equal(t, "Amazon EC2", results[0].Service)
	assert.True(t, results[0].TotalAmount.Valid)
	assert.Equal(t, 2200.00, results[0].TotalAmount.Float64)
}

func TestUpsertResource(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	err := s.Queries.UpsertResource(ctx, UpsertResourceParams{
		Provider:     "aws",
		AccountID:    "123456789012",
		Service:      "Amazon EC2",
		ResourceID:   "i-abc123def456",
		ResourceType: "ec2:instance",
		Name:         sql.NullString{String: "web-server-1", Valid: true},
		Region:       sql.NullString{String: "us-east-1", Valid: true},
		Spec:         sql.NullString{String: `{"instance_type":"m5.xlarge"}`, Valid: true},
		Tags:         sql.NullString{String: `{"env":"prod"}`, Valid: true},
		State:        sql.NullString{String: "running", Valid: true},
		DiscoveredAt: "2026-03-18T00:00:00Z",
	})
	require.NoError(t, err)

	resources, err := s.Queries.GetResourcesByService(ctx, GetResourcesByServiceParams{
		Provider: "aws",
		Service:  "Amazon EC2",
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "i-abc123def456", resources[0].ResourceID)
	assert.Equal(t, "web-server-1", resources[0].Name.String)
}

func TestSyncHistory(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	id, err := s.Queries.InsertSyncHistory(ctx, InsertSyncHistoryParams{
		Provider:    "aws",
		AccountID:   "123456789012",
		Region:      sql.NullString{},
		PeriodStart: "2025-10-01",
		PeriodEnd:   "2026-03-01",
		CostRecords: sql.NullInt64{},
		ResourcesFound: sql.NullInt64{},
		StartedAt:   "2026-03-18T00:00:00Z",
		CompletedAt: sql.NullString{},
	})
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	err = s.Queries.UpdateSyncHistoryCompleted(ctx, UpdateSyncHistoryCompletedParams{
		CostRecords:    sql.NullInt64{Int64: 42, Valid: true},
		ResourcesFound: sql.NullInt64{Int64: 15, Valid: true},
		CompletedAt:    sql.NullString{String: "2026-03-18T00:01:00Z", Valid: true},
		ID:             id,
	})
	require.NoError(t, err)

	sync, err := s.Queries.GetLatestSync(ctx, GetLatestSyncParams{
		Provider:  "aws",
		AccountID: "123456789012",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(42), sync.CostRecords.Int64)
	assert.Equal(t, int64(15), sync.ResourcesFound.Int64)
	assert.True(t, sync.CompletedAt.Valid)
}

func TestConfig(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	// Set config
	err := s.Queries.SetConfig(ctx, SetConfigParams{
		Key:   "retention_months",
		Value: "12",
	})
	require.NoError(t, err)

	// Get config
	value, err := s.Queries.GetConfig(ctx, "retention_months")
	require.NoError(t, err)
	assert.Equal(t, "12", value)

	// Update config
	err = s.Queries.SetConfig(ctx, SetConfigParams{
		Key:   "retention_months",
		Value: "24",
	})
	require.NoError(t, err)

	value, err = s.Queries.GetConfig(ctx, "retention_months")
	require.NoError(t, err)
	assert.Equal(t, "24", value)

	// List config
	configs, err := s.Queries.ListConfig(ctx)
	require.NoError(t, err)
	require.Len(t, configs, 1)

	// Delete config
	err = s.Queries.DeleteConfig(ctx, "retention_months")
	require.NoError(t, err)

	_, err = s.Queries.GetConfig(ctx, "retention_months")
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestPrune(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()

	// Insert old and recent records
	err := s.Queries.UpsertCostRecord(ctx, UpsertCostRecordParams{
		Provider:    "aws",
		AccountID:   "123456789012",
		Service:     "Amazon EC2",
		Region:      sql.NullString{String: "us-east-1", Valid: true},
		PeriodStart: "2020-01-01",
		PeriodEnd:   "2020-02-01",
		Granularity: "MONTHLY",
		Amount:      100.00,
		Currency:    "USD",
		SyncedAt:    "2026-03-18T00:00:00Z",
	})
	require.NoError(t, err)

	err = s.Queries.UpsertCostRecord(ctx, UpsertCostRecordParams{
		Provider:    "aws",
		AccountID:   "123456789012",
		Service:     "Amazon EC2",
		Region:      sql.NullString{String: "us-east-1", Valid: true},
		PeriodStart: "2026-02-01",
		PeriodEnd:   "2026-03-01",
		Granularity: "MONTHLY",
		Amount:      200.00,
		Currency:    "USD",
		SyncedAt:    "2026-03-18T00:00:00Z",
	})
	require.NoError(t, err)

	// Prune with 12-month retention
	deleted, err := s.Prune(12)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only recent record remains
	count, err := s.Queries.CountCostRecords(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestDBSize(t *testing.T) {
	s := setupTestDB(t)

	size, err := s.DBSize()
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))
}
