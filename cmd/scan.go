package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	providerpkg "github.com/helmcode/finops-cli/internal/provider"
	awsprovider "github.com/helmcode/finops-cli/internal/provider/aws"
	"github.com/helmcode/finops-cli/internal/store"
)

var (
	scanProvider string
	scanRegion   string
	scanMonths   int
	scanFrom     string
	scanTo       string
	scanAccount  string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Download costs and discover resources from cloud providers",
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVar(&scanProvider, "provider", "", "Cloud provider (aws)")
	scanCmd.Flags().StringVar(&scanRegion, "region", "all", "Region to scan (default: all)")
	scanCmd.Flags().IntVar(&scanMonths, "months", 6, "Number of months to sync (1-12, default: 6)")
	scanCmd.Flags().StringVar(&scanFrom, "from", "", "Start date (YYYY-MM-DD)")
	scanCmd.Flags().StringVar(&scanTo, "to", "", "End date (YYYY-MM-DD)")
	scanCmd.Flags().StringVar(&scanAccount, "account", "", "Filter accounts (comma-separated IDs)")

	_ = scanCmd.MarkFlagRequired("provider")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	// Validate provider
	if scanProvider != "aws" {
		return fmt.Errorf("unsupported provider %q (supported: aws)", scanProvider)
	}

	// Validate months
	if scanMonths < 1 || scanMonths > 12 {
		return fmt.Errorf("--months must be between 1 and 12 (got %d)", scanMonths)
	}

	// Determine date range
	start, end, err := determineDateRange()
	if err != nil {
		return err
	}

	// Validate date range
	if err := awsprovider.ValidateDateRange(start, end); err != nil {
		return err
	}

	// Parse account filter
	var accountFilter []string
	if scanAccount != "" {
		accountFilter = strings.Split(scanAccount, ",")
		for i := range accountFilter {
			accountFilter[i] = strings.TrimSpace(accountFilter[i])
		}
	}

	// Open store
	s, err := store.Open()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create provider
	sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Suffix = " Connecting to AWS..."
	sp.Start()

	provider, err := awsprovider.NewAWSProvider(ctx)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("creating AWS provider: %w", err)
	}

	// Detect account mode
	sp.Suffix = " Detecting account mode..."
	mode, err := provider.DetectAccountMode()
	sp.Stop()
	if err != nil {
		return fmt.Errorf("detecting account mode: %w", err)
	}

	if mode.IsOrganization {
		fmt.Printf("AWS Organization detected (management account: %s)\n", mode.ManagementID)
	} else {
		fmt.Printf("Single account mode (account: %s)\n", mode.ManagementID)
	}

	// List accounts
	accounts, err := provider.ListAccounts(accountFilter)
	if err != nil {
		return fmt.Errorf("listing accounts: %w", err)
	}

	if len(accountFilter) > 0 && !mode.IsOrganization {
		slog.Warn("--account filter ignored in single account mode")
	}

	fmt.Printf("Scanning %d account(s) from %s to %s\n\n",
		len(accounts), start.Format("2006-01-02"), end.Format("2006-01-02"))

	// Process each account
	totalCostRecords := 0
	totalResources := 0
	accountsProcessed := 0
	accountsSkipped := 0
	var skippedAccounts []string

	for _, acct := range accounts {
		sp = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		sp.Suffix = fmt.Sprintf(" Fetching costs for account %s (%s)...", acct.ID, acct.Name)
		sp.Start()

		// Insert sync history
		syncID, err := s.Queries.InsertSyncHistory(ctx, store.InsertSyncHistoryParams{
			Provider:    "aws",
			AccountID:   acct.ID,
			Region:      sql.NullString{String: scanRegion, Valid: true},
			PeriodStart: start.Format("2006-01-02"),
			PeriodEnd:   end.Format("2006-01-02"),
			StartedAt:   time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			sp.Stop()
			slog.Warn("failed to create sync history", "account", acct.ID, "error", err)
		}

		// Fetch costs
		groupBy := []string{"SERVICE", "REGION"}
		costRecords, err := provider.FetchCosts(providerpkg.CostParams{
			AccountID:   acct.ID,
			Start:       start,
			End:         end,
			Granularity: "MONTHLY",
			GroupBy:     groupBy,
		})
		if err != nil {
			sp.Stop()
			if awsprovider.IsAccessDenied(err) {
				slog.Warn("access denied, skipping account", "account", acct.ID)
				accountsSkipped++
				skippedAccounts = append(skippedAccounts, acct.ID)
				continue
			}
			return fmt.Errorf("fetching costs for account %s: %w", acct.ID, err)
		}

		// Store cost records
		costCount := 0
		for _, record := range costRecords {
			err := s.Queries.UpsertCostRecord(ctx, store.UpsertCostRecordParams{
				Provider:    record.Provider,
				AccountID:   record.AccountID,
				Service:     record.Service,
				Region:      sql.NullString{String: record.Region, Valid: record.Region != ""},
				PeriodStart: record.PeriodStart,
				PeriodEnd:   record.PeriodEnd,
				Granularity: record.Granularity,
				Amount:      record.Amount,
				Currency:    record.Currency,
				SyncedAt:    time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				slog.Warn("failed to store cost record", "error", err)
				continue
			}
			costCount++
		}

		// Discover resources for each service+region pair with spend
		sp.Suffix = fmt.Sprintf(" Discovering resources for account %s...", acct.ID)
		resourceCount := 0
		discoveredPairs := make(map[string]bool)

		for _, record := range costRecords {
			// Skip zero-cost records to avoid unnecessary API calls
			if record.Amount <= 0 {
				continue
			}

			region := record.Region
			if region == "" || scanRegion != "all" {
				if scanRegion != "all" {
					region = scanRegion
				}
			}

			// Skip empty regions (e.g., Tax, Support) and deduplicate by service+region
			if region == "" || region == "NoRegion" {
				continue
			}
			pairKey := record.Service + "|" + region
			if discoveredPairs[pairKey] {
				continue
			}
			discoveredPairs[pairKey] = true

			resources, err := provider.DiscoverResources(record.Service, region)
			if err != nil {
				slog.Debug("resource discovery failed", "service", record.Service, "error", err)
				continue
			}

			for _, res := range resources {
				err := s.Queries.UpsertResource(ctx, store.UpsertResourceParams{
					Provider:     res.Provider,
					AccountID:    res.AccountID,
					Service:      res.Service,
					ResourceID:   res.ResourceID,
					ResourceType: res.ResourceType,
					Name:         sql.NullString{String: res.Name, Valid: res.Name != ""},
					Region:       sql.NullString{String: res.Region, Valid: res.Region != ""},
					Spec:         sql.NullString{String: res.Spec, Valid: res.Spec != ""},
					Tags:         sql.NullString{String: res.Tags, Valid: res.Tags != ""},
					State:        sql.NullString{String: res.State, Valid: res.State != ""},
					DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
				})
				if err != nil {
					slog.Debug("failed to store resource", "resource", res.ResourceID, "error", err)
					continue
				}
				resourceCount++
			}
		}

		sp.Stop()

		// Update sync history
		if syncID > 0 {
			_ = s.Queries.UpdateSyncHistoryCompleted(ctx, store.UpdateSyncHistoryCompletedParams{
				CostRecords:    sql.NullInt64{Int64: int64(costCount), Valid: true},
				ResourcesFound: sql.NullInt64{Int64: int64(resourceCount), Valid: true},
				CompletedAt:    sql.NullString{String: time.Now().UTC().Format(time.RFC3339), Valid: true},
				ID:             syncID,
			})
		}

		totalCostRecords += costCount
		totalResources += resourceCount
		accountsProcessed++

		if verbose {
			fmt.Printf("  Account %s: %d cost records, %d resources\n", acct.ID, costCount, resourceCount)
		}
	}

	// Fetch commitment data (Savings Plans, Reserved Instances)
	sp = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Suffix = " Fetching commitment data (Savings Plans, Reserved Instances)..."
	sp.Start()

	commitmentCount := 0
	for _, acct := range accounts {
		commitRecords, err := provider.FetchCommitments(providerpkg.CommitmentParams{
			AccountID: acct.ID,
			Start:     start,
			End:       end,
		})
		if err != nil {
			slog.Debug("could not fetch commitments", "account", acct.ID, "error", err)
			continue
		}
		for _, cr := range commitRecords {
			err := s.Queries.UpsertCommitment(ctx, store.UpsertCommitmentParams{
				Provider:           cr.Provider,
				AccountID:          cr.AccountID,
				CommitmentType:     cr.CommitmentType,
				PeriodStart:        cr.PeriodStart,
				PeriodEnd:          cr.PeriodEnd,
				TotalCommitment:    cr.TotalCommitment,
				UsedCommitment:     cr.UsedCommitment,
				OnDemandEquivalent: cr.OnDemandEquivalent,
				NetSavings:         cr.NetSavings,
				UtilizationPct:     cr.UtilizationPct,
				CoveragePct:        cr.CoveragePct,
				Currency:           cr.Currency,
				SyncedAt:           time.Now().UTC().Format(time.RFC3339),
			})
			if err != nil {
				slog.Debug("failed to store commitment record", "error", err)
				continue
			}
			commitmentCount++
		}
	}
	sp.Stop()

	if commitmentCount > 0 && verbose {
		fmt.Printf("  Commitment records: %d synced\n", commitmentCount)
	}

	// Auto-prune
	retentionStr, err := s.Queries.GetConfig(ctx, "retention_months")
	retention := 12
	if err == nil {
		if parsed, e := strconv.Atoi(retentionStr); e == nil {
			retention = parsed
		}
	}
	pruned, _ := s.Prune(retention)
	if pruned > 0 {
		slog.Info("auto-pruned old records", "deleted", pruned)
	}

	// Print summary
	printScanSummary(accountsProcessed, accountsSkipped, totalCostRecords, totalResources, skippedAccounts)

	return nil
}

func determineDateRange() (time.Time, time.Time, error) {
	now := time.Now()

	if scanFrom != "" && scanTo != "" {
		start, err := time.Parse("2006-01-02", scanFrom)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from date: %w", err)
		}
		end, err := time.Parse("2006-01-02", scanTo)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --to date: %w", err)
		}
		return start, end, nil
	}

	if scanFrom != "" {
		start, err := time.Parse("2006-01-02", scanFrom)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from date: %w", err)
		}
		return start, now, nil
	}

	// Default: last N months
	start := time.Date(now.Year(), now.Month()-time.Month(scanMonths), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, end, nil
}

func printScanSummary(processed, skipped, costRecords, resources int, skippedAccounts []string) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	fmt.Println()
	fmt.Println(headerStyle.Render("Scan completed"))
	fmt.Printf("%s %s\n",
		successStyle.Render(fmt.Sprintf("%d/%d accounts processed", processed, processed+skipped)),
		func() string {
			if skipped > 0 {
				return warnStyle.Render(fmt.Sprintf(", %d skipped (no permissions)", skipped))
			}
			return ""
		}(),
	)
	fmt.Printf("Cost records: %s | Resources: %s\n",
		successStyle.Render(fmt.Sprintf("%d synced", costRecords)),
		successStyle.Render(fmt.Sprintf("%d discovered", resources)),
	)

	if skipped > 0 && !verbose {
		fmt.Println(warnStyle.Render("Run with --verbose for details"))
	}

	if verbose && len(skippedAccounts) > 0 {
		fmt.Println(warnStyle.Render("Skipped accounts: " + strings.Join(skippedAccounts, ", ")))
	}
}
