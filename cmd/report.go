package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/helmcode/finops-cli/internal/analysis"
	"github.com/helmcode/finops-cli/internal/report"
	"github.com/helmcode/finops-cli/internal/store"
)

var (
	reportOutput   string
	reportFile     string
	reportLimit    int
	reportService  string
	reportRegion   string
	reportCurrent  string
	reportPrevious string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate reports from local data",
}

var reportSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "General overview with charts",
	RunE:  runReportSummary,
}

var reportTopServicesCmd = &cobra.Command{
	Use:   "top-services",
	Short: "Top N services by cost",
	RunE:  runReportTopServices,
}

var reportTrendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Temporal trend for a service",
	RunE:  runReportTrend,
}

var reportAnomaliesCmd = &cobra.Command{
	Use:   "anomalies",
	Short: "Anomalous cost spike detection",
	RunE:  runReportAnomalies,
}

var reportCompareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare two periods",
	RunE:  runReportCompare,
}

var reportResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Discovered resources with cost context",
	RunE:  runReportResources,
}

func init() {
	// Shared flags
	reportCmd.PersistentFlags().StringVar(&reportOutput, "output", "html", "Output format: html, csv, pdf")
	reportCmd.PersistentFlags().StringVar(&reportFile, "file", "", "Output file path (default: auto-generated)")

	// Per-subcommand flags
	reportTopServicesCmd.Flags().IntVar(&reportLimit, "limit", 10, "Number of top services to show")
	reportTrendCmd.Flags().StringVar(&reportService, "service", "", "Filter by service name")
	reportResourcesCmd.Flags().StringVar(&reportService, "service", "", "Filter by service name")
	reportResourcesCmd.Flags().StringVar(&reportRegion, "region", "", "Filter by region")
	reportCompareCmd.Flags().StringVar(&reportCurrent, "current", "", "Current period (YYYY-MM-DD:YYYY-MM-DD)")
	reportCompareCmd.Flags().StringVar(&reportPrevious, "previous", "", "Previous period (YYYY-MM-DD:YYYY-MM-DD)")

	reportCmd.AddCommand(reportSummaryCmd)
	reportCmd.AddCommand(reportTopServicesCmd)
	reportCmd.AddCommand(reportTrendCmd)
	reportCmd.AddCommand(reportAnomaliesCmd)
	reportCmd.AddCommand(reportCompareCmd)
	reportCmd.AddCommand(reportResourcesCmd)
	rootCmd.AddCommand(reportCmd)
}

func openStore() (*store.Store, error) {
	s, err := store.Open()
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return s, nil
}

func checkDataFreshness(s *store.Store) error {
	ctx := context.Background()
	count, err := s.Queries.CountCostRecords(ctx)
	if err != nil {
		return fmt.Errorf("checking data: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("no data found. Run 'finops scan --provider aws' first")
	}
	return nil
}

func defaultDateRange() analysis.DateRange {
	now := time.Now()
	start := time.Date(now.Year(), now.Month()-6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return analysis.DateRange{
		Start: start.Format("2006-01-02"),
		End:   end.Format("2006-01-02"),
	}
}

func outputPath(name string) string {
	if reportFile != "" {
		return reportFile
	}
	ext := reportOutput
	if ext == "" {
		ext = "html"
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("finops_%s_%s.%s", name, time.Now().Format("20060102_150405"), ext))
}

func finalizeReport(path string) {
	fmt.Printf("Report saved to: %s\n", path)
	if reportOutput == "html" && reportFile == "" {
		if err := report.OpenInBrowser(path); err != nil {
			slog.Debug("could not open browser", "error", err)
		}
	}
}

func runReportSummary(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if err := checkDataFreshness(s); err != nil {
		return err
	}

	dr := defaultDateRange()
	summaryData, err := analysis.GenerateSummary(s.Queries, "aws", dr)
	if err != nil {
		return fmt.Errorf("generating summary: %w", err)
	}

	ctx := context.Background()
	resourceCount, _ := s.Queries.CountResourcesByProvider(ctx, "aws")

	// Build region details with associated resources and service cost breakdown
	var regionDetails []report.RegionDetail
	for _, rc := range summaryData.CostByRegion {
		rd := report.RegionDetail{
			Region:      rc.Region,
			TotalAmount: rc.TotalAmount,
			Currency:    rc.Currency,
		}
		if rc.Region != "" {
			resources, err := s.Queries.GetResourcesByRegion(ctx, store.GetResourcesByRegionParams{
				Provider: "aws",
				Region:   sql.NullString{String: rc.Region, Valid: true},
			})
			if err == nil {
				rd.Resources = resources
			}
		}
		// Always get service cost breakdown per region
		regionName := rc.Region
		if regionName == "" {
			regionName = "NoRegion"
		}
		serviceCosts, err := s.Queries.GetCostByServiceForRegion(ctx, store.GetCostByServiceForRegionParams{
			Provider:    "aws",
			Region:      sql.NullString{String: regionName, Valid: true},
			PeriodStart: dr.Start,
			PeriodEnd:   dr.End,
		})
		if err == nil {
			for _, sc := range serviceCosts {
				amount := 0.0
				if sc.TotalAmount.Valid {
					amount = sc.TotalAmount.Float64
				}
				if amount > 0 {
					rd.ServiceCosts = append(rd.ServiceCosts, report.RegionServiceCost{
						Service: sc.Service,
						Amount:  amount,
					})
				}
			}
		}
		regionDetails = append(regionDetails, rd)
	}

	// Get monthly spend data for the bar chart
	trendData, err := analysis.GenerateTrend(s.Queries, "aws", "")
	if err != nil {
		slog.Debug("could not generate monthly trend for summary", "error", err)
	}
	var monthlySpend []analysis.MonthlyDataPoint
	if trendData != nil {
		monthlySpend = trendData.DataPoints
	}

	// Build account details with top services per account
	var accountDetails []report.AccountDetail
	for _, acct := range summaryData.CostByAccount {
		ad := report.AccountDetail{
			AccountID:     acct.AccountID,
			TotalAmount:   acct.TotalAmount,
			Currency:      acct.Currency,
			ResourceCount: acct.ResourceCount,
		}
		topSvcs, err := s.Queries.GetTopServicesByAccount(ctx, store.GetTopServicesByAccountParams{
			Provider:    "aws",
			AccountID:   acct.AccountID,
			PeriodStart: dr.Start,
			PeriodEnd:   dr.End,
		})
		if err == nil {
			for _, svc := range topSvcs {
				amount := 0.0
				if svc.TotalAmount.Valid {
					amount = svc.TotalAmount.Float64
				}
				if amount > 0 {
					ad.TopServices = append(ad.TopServices, report.AccountServiceCost{
						Service: svc.Service,
						Amount:  amount,
					})
				}
			}
		}
		accountDetails = append(accountDetails, ad)
	}

	// Get commitment overview
	commitmentOverview, err := analysis.GenerateCommitmentOverview(s.Queries, "aws", dr)
	if err != nil {
		slog.Debug("could not generate commitment overview", "error", err)
	}

	reportData := report.ReportData{
		Title: "Cost Summary", PeriodStart: dr.Start, PeriodEnd: dr.End,
		Data: summaryData, TotalResources: resourceCount, MonthCount: 6,
		RegionDetails:     regionDetails,
		MonthlySpend:      monthlySpend,
		AccountDetails:    accountDetails,
		CommitmentOverview: commitmentOverview,
	}

	path := outputPath("summary")
	switch reportOutput {
	case "csv":
		if err := report.GenerateSummaryCSV(path, summaryData); err != nil {
			return err
		}
	case "pdf":
		htmlPath := path + ".html"
		if err := report.GenerateHTML("summary", htmlPath, reportData); err != nil {
			return err
		}
		if err := report.GeneratePDF(htmlPath, path); err != nil {
			return err
		}
		os.Remove(htmlPath)
	default:
		if err := report.GenerateHTML("summary", path, reportData); err != nil {
			return err
		}
	}

	finalizeReport(path)
	return nil
}

func runReportTopServices(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if err := checkDataFreshness(s); err != nil {
		return err
	}

	dr := defaultDateRange()
	summaryData, err := analysis.GenerateSummary(s.Queries, "aws", dr)
	if err != nil {
		return fmt.Errorf("generating summary: %w", err)
	}

	// Apply limit
	if reportLimit > 0 && reportLimit < len(summaryData.TopServices) {
		summaryData.TopServices = summaryData.TopServices[:reportLimit]
	}

	path := outputPath("top_services")
	switch reportOutput {
	case "csv":
		if err := report.GenerateSummaryCSV(path, summaryData); err != nil {
			return err
		}
	default:
		if err := report.GenerateHTML("top_services", path, report.ReportData{
			Title: "Top Services", PeriodStart: dr.Start, PeriodEnd: dr.End,
			Data: summaryData, MonthCount: 6,
		}); err != nil {
			return err
		}
	}

	finalizeReport(path)
	return nil
}

func runReportTrend(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if err := checkDataFreshness(s); err != nil {
		return err
	}

	trendData, err := analysis.GenerateTrend(s.Queries, "aws", reportService)
	if err != nil {
		return fmt.Errorf("generating trend: %w", err)
	}

	dr := defaultDateRange()
	path := outputPath("trend")
	switch reportOutput {
	case "csv":
		if err := report.GenerateTrendCSV(path, trendData); err != nil {
			return err
		}
	default:
		if err := report.GenerateHTML("trend", path, report.ReportData{
			Title: "Cost Trend", PeriodStart: dr.Start, PeriodEnd: dr.End,
			Data: trendData,
		}); err != nil {
			return err
		}
	}

	finalizeReport(path)
	return nil
}

func runReportAnomalies(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if err := checkDataFreshness(s); err != nil {
		return err
	}

	dr := defaultDateRange()
	anomalies, err := analysis.DetectAnomalies(s.Queries, "aws", dr, 2.0)
	if err != nil {
		return fmt.Errorf("detecting anomalies: %w", err)
	}

	path := outputPath("anomalies")
	switch reportOutput {
	case "csv":
		if err := report.GenerateAnomaliesCSV(path, anomalies); err != nil {
			return err
		}
	default:
		if err := report.GenerateHTML("anomalies", path, report.ReportData{
			Title: "Cost Anomalies", PeriodStart: dr.Start, PeriodEnd: dr.End,
			Data: anomalies,
		}); err != nil {
			return err
		}
	}

	finalizeReport(path)
	return nil
}

func runReportCompare(cmd *cobra.Command, args []string) error {
	if reportCurrent == "" || reportPrevious == "" {
		return fmt.Errorf("--current and --previous are required (format: YYYY-MM-DD:YYYY-MM-DD)")
	}

	currentDR, err := parseDateRange(reportCurrent)
	if err != nil {
		return fmt.Errorf("invalid --current: %w", err)
	}

	previousDR, err := parseDateRange(reportPrevious)
	if err != nil {
		return fmt.Errorf("invalid --previous: %w", err)
	}

	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	if err := checkDataFreshness(s); err != nil {
		return err
	}

	compareData, err := analysis.ComparePeriods(s.Queries, "aws", currentDR, previousDR)
	if err != nil {
		return fmt.Errorf("comparing periods: %w", err)
	}

	path := outputPath("compare")
	switch reportOutput {
	case "csv":
		if err := report.GenerateCompareCSV(path, compareData); err != nil {
			return err
		}
	default:
		if err := report.GenerateHTML("compare", path, report.ReportData{
			Title:       "Period Comparison",
			PeriodStart: currentDR.Start,
			PeriodEnd:   currentDR.End,
			Data:        compareData,
		}); err != nil {
			return err
		}
	}

	finalizeReport(path)
	return nil
}

func runReportResources(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := context.Background()

	var resources []store.Resource

	switch {
	case reportService != "" && reportRegion != "":
		resources, err = s.Queries.GetResourcesByServiceAndRegion(ctx, store.GetResourcesByServiceAndRegionParams{
			Provider: "aws",
			Service:  reportService,
			Region:   sql.NullString{String: reportRegion, Valid: true},
		})
	case reportService != "":
		resources, err = s.Queries.GetResourcesByService(ctx, store.GetResourcesByServiceParams{
			Provider: "aws",
			Service:  reportService,
		})
	case reportRegion != "":
		resources, err = s.Queries.GetResourcesByRegion(ctx, store.GetResourcesByRegionParams{
			Provider: "aws",
			Region:   sql.NullString{String: reportRegion, Valid: true},
		})
	default:
		resources, err = s.Queries.GetResourcesByProvider(ctx, "aws")
	}
	if err != nil {
		return fmt.Errorf("querying resources: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found. Run 'finops scan --provider aws' first.")
		return nil
	}

	dr := defaultDateRange()
	path := outputPath("resources")
	switch reportOutput {
	default:
		if err := report.GenerateHTML("resources", path, report.ReportData{
			Title:       "Discovered Resources",
			PeriodStart: dr.Start,
			PeriodEnd:   dr.End,
			Data:        resources,
		}); err != nil {
			return err
		}
	}

	finalizeReport(path)
	return nil
}

func parseDateRange(s string) (analysis.DateRange, error) {
	parts := filepath.SplitList(s)
	if len(parts) != 1 {
		return analysis.DateRange{}, fmt.Errorf("expected format YYYY-MM-DD:YYYY-MM-DD")
	}

	// Split on ":"
	idx := -1
	for i, c := range s {
		if c == ':' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return analysis.DateRange{}, fmt.Errorf("expected format YYYY-MM-DD:YYYY-MM-DD, got %q", s)
	}

	start := s[:idx]
	end := s[idx+1:]

	if _, err := time.Parse("2006-01-02", start); err != nil {
		return analysis.DateRange{}, fmt.Errorf("invalid start date: %w", err)
	}
	if _, err := time.Parse("2006-01-02", end); err != nil {
		return analysis.DateRange{}, fmt.Errorf("invalid end date: %w", err)
	}

	return analysis.DateRange{Start: start, End: end}, nil
}
