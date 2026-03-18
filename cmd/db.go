package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/helmcode/finops-cli/internal/store"
)

var (
	retentionMonths int
	forceRetention  bool
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Local database management",
}

var dbStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show database information (size, record counts, last sync)",
	RunE:  runDBStats,
}

var dbPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Modify data retention and prune old records",
	RunE:  runDBPrune,
}

func init() {
	dbPruneCmd.Flags().IntVar(&retentionMonths, "retention", 12, "Retention period in months (min: 1)")
	dbPruneCmd.Flags().BoolVar(&forceRetention, "force", false, "Skip confirmation for unusual retention periods")

	dbCmd.AddCommand(dbStatsCmd)
	dbCmd.AddCommand(dbPruneCmd)
	rootCmd.AddCommand(dbCmd)
}

func runDBStats(cmd *cobra.Command, args []string) error {
	s, err := store.Open()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	costCount, err := s.Queries.CountCostRecords(ctx)
	if err != nil {
		return fmt.Errorf("counting cost records: %w", err)
	}

	resourceCount, err := s.Queries.CountResources(ctx)
	if err != nil {
		return fmt.Errorf("counting resources: %w", err)
	}

	syncCount, err := s.Queries.CountSyncHistory(ctx)
	if err != nil {
		return fmt.Errorf("counting sync history: %w", err)
	}

	dbSize, err := s.DBSize()
	if err != nil {
		return fmt.Errorf("getting database size: %w", err)
	}

	// Get last sync date
	lastSync := "never"
	latestSync, err := s.Queries.GetLatestSyncByProvider(ctx, "aws")
	if err == nil && latestSync.CompletedAt.Valid {
		lastSync = latestSync.CompletedAt.String
	}

	// Get retention setting
	retention := "12 (default)"
	retVal, err := s.Queries.GetConfig(ctx, "retention_months")
	if err == nil {
		retention = retVal
	}

	// Style output
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle := lipgloss.NewStyle().Width(20)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	fmt.Println(headerStyle.Render("Database Statistics"))
	fmt.Println()
	fmt.Printf("%s %s\n", labelStyle.Render("Size:"), valueStyle.Render(formatBytes(dbSize)))
	fmt.Printf("%s %s\n", labelStyle.Render("Cost records:"), valueStyle.Render(strconv.FormatInt(costCount, 10)))
	fmt.Printf("%s %s\n", labelStyle.Render("Resources:"), valueStyle.Render(strconv.FormatInt(resourceCount, 10)))
	fmt.Printf("%s %s\n", labelStyle.Render("Sync history:"), valueStyle.Render(strconv.FormatInt(syncCount, 10)))
	fmt.Printf("%s %s\n", labelStyle.Render("Last sync:"), valueStyle.Render(lastSync))
	fmt.Printf("%s %s\n", labelStyle.Render("Retention:"), valueStyle.Render(retention+" months"))

	return nil
}

func runDBPrune(cmd *cobra.Command, args []string) error {
	// Validate retention
	if retentionMonths < 1 {
		return fmt.Errorf("minimum retention is 1 month")
	}

	if retentionMonths > 24 && !forceRetention {
		return fmt.Errorf("unusual retention period (%d months). Use --force to confirm", retentionMonths)
	}

	if retentionMonths > 24 {
		slog.Warn("unusual retention period", "months", retentionMonths)
	}

	s, err := store.Open()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Save retention setting
	err = s.Queries.SetConfig(ctx, store.SetConfigParams{
		Key:   "retention_months",
		Value: strconv.Itoa(retentionMonths),
	})
	if err != nil {
		return fmt.Errorf("saving retention setting: %w", err)
	}

	// Prune records
	deleted, err := s.Prune(retentionMonths)
	if err != nil {
		return fmt.Errorf("pruning records: %w", err)
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println(successStyle.Render(fmt.Sprintf("Retention set to %d months. Pruned %d old records.", retentionMonths, deleted)))

	return nil
}

func formatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
