package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/helmcode/finops-cli/internal/analysis"
)

// --- JSON output structs ---

// JSONSummaryReport is the top-level JSON output for the summary report.
type JSONSummaryReport struct {
	ReportType         string               `json:"report_type"`
	GeneratedAt        string               `json:"generated_at"`
	Period             JSONPeriod            `json:"period"`
	TotalSpend         float64              `json:"total_spend"`
	Currency           string               `json:"currency"`
	ActiveServices     int                  `json:"active_services"`
	ResourcesDiscovered int64               `json:"resources_discovered"`
	CostByAccount      []JSONAccountDetail  `json:"cost_by_account"`
	MonthlySpend       []JSONMonthlySpend   `json:"monthly_spend"`
	TopServices        []JSONServiceCost    `json:"top_services"`
	CostByRegion       []JSONRegionDetail   `json:"cost_by_region"`
	Commitments        *JSONCommitments     `json:"commitments,omitempty"`
}

// JSONPeriod represents the report time range.
type JSONPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// JSONAccountDetail holds full cost and resource data for an account.
type JSONAccountDetail struct {
	AccountID     string              `json:"account_id"`
	TotalAmount   float64             `json:"total_amount"`
	Currency      string              `json:"currency"`
	Percentage    float64             `json:"percentage"`
	ResourceCount int64               `json:"resource_count"`
	TopServices   []JSONSimpleCost    `json:"top_services"`
}

// JSONSimpleCost is a service-amount pair.
type JSONSimpleCost struct {
	Service string  `json:"service"`
	Amount  float64 `json:"amount"`
}

// JSONMonthlySpend is a period-amount pair for the monthly trend.
type JSONMonthlySpend struct {
	Period string  `json:"period"`
	Amount float64 `json:"amount"`
}

// JSONServiceCost is a service with cost and percentage of total.
type JSONServiceCost struct {
	Service     string  `json:"service"`
	TotalAmount float64 `json:"total_amount"`
	Currency    string  `json:"currency"`
	Percentage  float64 `json:"percentage"`
}

// JSONRegionDetail holds a region's cost breakdown and resources.
type JSONRegionDetail struct {
	Region        string             `json:"region"`
	TotalAmount   float64            `json:"total_amount"`
	Currency      string             `json:"currency"`
	ResourceCount int                `json:"resource_count"`
	ServiceCosts  []JSONSimpleCost   `json:"service_costs,omitempty"`
	Resources     []JSONResource     `json:"resources,omitempty"`
}

// JSONResource is a discovered cloud resource.
type JSONResource struct {
	Service      string `json:"service"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Name         string `json:"name,omitempty"`
	State        string `json:"state,omitempty"`
	AccountID    string `json:"account_id"`
}

// JSONCommitments holds Savings Plans, Reserved Instances, and Spot data.
type JSONCommitments struct {
	TotalCommitted    float64                  `json:"total_committed"`
	TotalUsed         float64                  `json:"total_used"`
	TotalSavings      float64                  `json:"total_savings"`
	AvgUtilization    float64                  `json:"avg_utilization_pct"`
	Currency          string                   `json:"currency"`
	HasData           bool                     `json:"has_data"`
	PermissionWarning bool                     `json:"permission_warning,omitempty"`
	Types             []JSONCommitmentType     `json:"types,omitempty"`
	SpotInstanceCount int64                    `json:"spot_instance_count"`
}

// JSONCommitmentType holds data for one commitment type (SP or RI).
type JSONCommitmentType struct {
	Type               string  `json:"type"`
	TotalCommitment    float64 `json:"total_commitment"`
	UsedCommitment     float64 `json:"used_commitment"`
	OnDemandEquivalent float64 `json:"on_demand_equivalent"`
	NetSavings         float64 `json:"net_savings"`
}

// JSONTrendReport is the JSON output for the trend report.
type JSONTrendReport struct {
	ReportType string             `json:"report_type"`
	GeneratedAt string            `json:"generated_at"`
	Service    string             `json:"service,omitempty"`
	Direction  string             `json:"direction"`
	DataPoints []JSONMonthlySpend `json:"data_points"`
}

// JSONAnomalyReport is the JSON output for the anomalies report.
type JSONAnomalyReport struct {
	ReportType  string        `json:"report_type"`
	GeneratedAt string        `json:"generated_at"`
	Anomalies   []JSONAnomaly `json:"anomalies"`
}

// JSONAnomaly is a single detected cost anomaly.
type JSONAnomaly struct {
	Period    string  `json:"period"`
	Service   string  `json:"service"`
	Expected  float64 `json:"expected"`
	Actual    float64 `json:"actual"`
	Deviation float64 `json:"deviation"`
	Severity  string  `json:"severity"`
}

// JSONCompareReport is the JSON output for the compare report.
type JSONCompareReport struct {
	ReportType    string              `json:"report_type"`
	GeneratedAt   string              `json:"generated_at"`
	CurrentPeriod  JSONPeriod         `json:"current_period"`
	PreviousPeriod JSONPeriod         `json:"previous_period"`
	TotalCurrent  float64             `json:"total_current"`
	TotalPrevious float64             `json:"total_previous"`
	TotalChange   float64             `json:"total_change"`
	TotalChangePct float64            `json:"total_change_pct"`
	Currency      string              `json:"currency"`
	ServiceDeltas []JSONServiceDelta  `json:"service_deltas"`
}

// JSONServiceDelta is a service's cost change between two periods.
type JSONServiceDelta struct {
	Service        string  `json:"service"`
	PreviousAmount float64 `json:"previous_amount"`
	CurrentAmount  float64 `json:"current_amount"`
	AbsoluteChange float64 `json:"absolute_change"`
	PercentChange  float64 `json:"percent_change"`
	Currency       string  `json:"currency"`
}

// JSONResourcesReport is the JSON output for the resources report.
type JSONResourcesReport struct {
	ReportType  string                `json:"report_type"`
	GeneratedAt string                `json:"generated_at"`
	TotalCount  int                   `json:"total_count"`
	Resources   []JSONResourceDetail  `json:"resources"`
}

// JSONResourceDetail is a full resource record for the resources report.
type JSONResourceDetail struct {
	Service      string `json:"service"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Name         string `json:"name,omitempty"`
	Region       string `json:"region,omitempty"`
	State        string `json:"state,omitempty"`
	AccountID    string `json:"account_id"`
	Spec         string `json:"spec,omitempty"`
	Tags         string `json:"tags,omitempty"`
}

// --- Generation functions ---

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func jsonWriter(outputPath string) (io.Writer, func(), error) {
	if outputPath == "" || outputPath == "-" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("creating JSON file: %w", err)
	}
	return f, func() { f.Close() }, nil
}

// GenerateSummaryJSON writes the full summary report as JSON.
func GenerateSummaryJSON(outputPath string, data ReportData) error {
	summaryData, ok := data.Data.(*analysis.SummaryData)
	if !ok {
		return fmt.Errorf("unexpected data type for summary JSON")
	}

	report := JSONSummaryReport{
		ReportType:          "summary",
		GeneratedAt:         nowISO(),
		Period:              JSONPeriod{Start: data.PeriodStart, End: data.PeriodEnd},
		TotalSpend:          roundTo(summaryData.TotalSpend, 2),
		Currency:            summaryData.Currency,
		ActiveServices:      len(summaryData.TopServices),
		ResourcesDiscovered: data.TotalResources,
	}

	// Cost by account
	for _, acct := range data.AccountDetails {
		pct := 0.0
		if summaryData.TotalSpend > 0 {
			pct = (acct.TotalAmount / summaryData.TotalSpend) * 100
		}
		ja := JSONAccountDetail{
			AccountID:     acct.AccountID,
			TotalAmount:   roundTo(acct.TotalAmount, 2),
			Currency:      acct.Currency,
			Percentage:    roundTo(pct, 1),
			ResourceCount: acct.ResourceCount,
		}
		for _, svc := range acct.TopServices {
			ja.TopServices = append(ja.TopServices, JSONSimpleCost{
				Service: svc.Service,
				Amount:  roundTo(svc.Amount, 2),
			})
		}
		report.CostByAccount = append(report.CostByAccount, ja)
	}

	// Monthly spend
	for _, dp := range data.MonthlySpend {
		report.MonthlySpend = append(report.MonthlySpend, JSONMonthlySpend{
			Period: dp.Period,
			Amount: roundTo(dp.Amount, 2),
		})
	}

	// Top services
	for _, svc := range summaryData.TopServices {
		pct := 0.0
		if summaryData.TotalSpend > 0 {
			pct = (svc.TotalAmount / summaryData.TotalSpend) * 100
		}
		report.TopServices = append(report.TopServices, JSONServiceCost{
			Service:     svc.Service,
			TotalAmount: roundTo(svc.TotalAmount, 2),
			Currency:    svc.Currency,
			Percentage:  roundTo(pct, 1),
		})
	}

	// Cost by region
	for _, rd := range data.RegionDetails {
		region := rd.Region
		if region == "" {
			region = "NoRegion"
		}
		jr := JSONRegionDetail{
			Region:        region,
			TotalAmount:   roundTo(rd.TotalAmount, 2),
			Currency:      rd.Currency,
			ResourceCount: len(rd.Resources),
		}
		for _, sc := range rd.ServiceCosts {
			jr.ServiceCosts = append(jr.ServiceCosts, JSONSimpleCost{
				Service: sc.Service,
				Amount:  roundTo(sc.Amount, 2),
			})
		}
		for _, r := range rd.Resources {
			name := ""
			if r.Name.Valid {
				name = r.Name.String
			}
			state := ""
			if r.State.Valid {
				state = r.State.String
			}
			jr.Resources = append(jr.Resources, JSONResource{
				Service:      r.Service,
				ResourceType: r.ResourceType,
				ResourceID:   r.ResourceID,
				Name:         name,
				State:        state,
				AccountID:    r.AccountID,
			})
		}
		report.CostByRegion = append(report.CostByRegion, jr)
	}

	// Commitments
	if co, ok := data.CommitmentOverview.(*analysis.CommitmentOverview); ok && co != nil {
		jc := &JSONCommitments{
			TotalCommitted:    roundTo(co.TotalCommitted, 2),
			TotalUsed:         roundTo(co.TotalUsed, 2),
			TotalSavings:      roundTo(co.TotalSavings, 2),
			AvgUtilization:    roundTo(co.AvgUtilization, 1),
			Currency:          co.Currency,
			HasData:           co.HasData,
			PermissionWarning: co.PermissionWarning,
			SpotInstanceCount: co.SpotInstanceCount,
		}
		for _, t := range co.Types {
			jc.Types = append(jc.Types, JSONCommitmentType{
				Type:               t.Type,
				TotalCommitment:    roundTo(t.TotalCommitment, 2),
				UsedCommitment:     roundTo(t.UsedCommitment, 2),
				OnDemandEquivalent: roundTo(t.OnDemandEquivalent, 2),
				NetSavings:         roundTo(t.NetSavings, 2),
			})
		}
		report.Commitments = jc
	}

	w, closer, err := jsonWriter(outputPath)
	if err != nil {
		return err
	}
	defer closer()
	return writeJSON(w, report)
}

// GenerateTrendJSON writes trend data as JSON.
func GenerateTrendJSON(outputPath string, data *analysis.TrendData) error {
	report := JSONTrendReport{
		ReportType:  "trend",
		GeneratedAt: nowISO(),
		Service:     data.Service,
		Direction:   string(data.Direction),
	}
	for _, dp := range data.DataPoints {
		report.DataPoints = append(report.DataPoints, JSONMonthlySpend{
			Period: dp.Period,
			Amount: roundTo(dp.Amount, 2),
		})
	}

	w, closer, err := jsonWriter(outputPath)
	if err != nil {
		return err
	}
	defer closer()
	return writeJSON(w, report)
}

// GenerateAnomaliesJSON writes anomaly data as JSON.
func GenerateAnomaliesJSON(outputPath string, data []analysis.AnomalyResult) error {
	report := JSONAnomalyReport{
		ReportType:  "anomalies",
		GeneratedAt: nowISO(),
	}
	for _, a := range data {
		report.Anomalies = append(report.Anomalies, JSONAnomaly{
			Period:    a.Period,
			Service:   a.Service,
			Expected:  roundTo(a.Expected, 2),
			Actual:    roundTo(a.Actual, 2),
			Deviation: roundTo(a.Deviation, 2),
			Severity:  string(a.Severity),
		})
	}

	w, closer, err := jsonWriter(outputPath)
	if err != nil {
		return err
	}
	defer closer()
	return writeJSON(w, report)
}

// GenerateCompareJSON writes comparison data as JSON.
func GenerateCompareJSON(outputPath string, data *analysis.CompareResult, currentDR, previousDR analysis.DateRange) error {
	report := JSONCompareReport{
		ReportType:     "compare",
		GeneratedAt:    nowISO(),
		CurrentPeriod:  JSONPeriod{Start: currentDR.Start, End: currentDR.End},
		PreviousPeriod: JSONPeriod{Start: previousDR.Start, End: previousDR.End},
		TotalCurrent:   roundTo(data.TotalCurrent, 2),
		TotalPrevious:  roundTo(data.TotalPrevious, 2),
		TotalChange:    roundTo(data.TotalChange, 2),
		TotalChangePct: roundTo(data.TotalPercent, 1),
		Currency:       data.Currency,
	}
	for _, d := range data.ServiceDeltas {
		report.ServiceDeltas = append(report.ServiceDeltas, JSONServiceDelta{
			Service:        d.Service,
			PreviousAmount: roundTo(d.PreviousAmount, 2),
			CurrentAmount:  roundTo(d.CurrentAmount, 2),
			AbsoluteChange: roundTo(d.AbsoluteChange, 2),
			PercentChange:  roundTo(d.PercentChange, 1),
			Currency:       d.Currency,
		})
	}

	w, closer, err := jsonWriter(outputPath)
	if err != nil {
		return err
	}
	defer closer()
	return writeJSON(w, report)
}

// GenerateResourcesJSON writes discovered resources as JSON.
func GenerateResourcesJSON(outputPath string, resources interface{}) error {
	// Accept []store.Resource via interface to avoid circular import
	type resourceLike struct {
		Service      string
		ResourceType string
		ResourceID   string
		Name         interface{ Valid() bool; StringVal() string }
		Region       interface{ Valid() bool; StringVal() string }
		State        interface{ Valid() bool; StringVal() string }
		AccountID    string
		Spec         interface{ Valid() bool; StringVal() string }
		Tags         interface{ Valid() bool; StringVal() string }
	}

	report := JSONResourcesReport{
		ReportType:  "resources",
		GeneratedAt: nowISO(),
	}

	// Use JSON marshal/unmarshal to convert store.Resource slice
	raw, err := json.Marshal(resources)
	if err != nil {
		return fmt.Errorf("marshaling resources: %w", err)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(raw, &items); err != nil {
		return fmt.Errorf("unmarshaling resources: %w", err)
	}

	for _, item := range items {
		r := JSONResourceDetail{
			Service:      strField(item, "service"),
			ResourceType: strField(item, "resource_type"),
			ResourceID:   strField(item, "resource_id"),
			AccountID:    strField(item, "account_id"),
		}
		if v, ok := item["name"]; ok && v != nil {
			if m, ok := v.(map[string]interface{}); ok {
				if valid, _ := m["Valid"].(bool); valid {
					r.Name, _ = m["String"].(string)
				}
			} else if s, ok := v.(string); ok {
				r.Name = s
			}
		}
		if v, ok := item["region"]; ok && v != nil {
			if m, ok := v.(map[string]interface{}); ok {
				if valid, _ := m["Valid"].(bool); valid {
					r.Region, _ = m["String"].(string)
				}
			} else if s, ok := v.(string); ok {
				r.Region = s
			}
		}
		if v, ok := item["state"]; ok && v != nil {
			if m, ok := v.(map[string]interface{}); ok {
				if valid, _ := m["Valid"].(bool); valid {
					r.State, _ = m["String"].(string)
				}
			} else if s, ok := v.(string); ok {
				r.State = s
			}
		}
		if v, ok := item["spec"]; ok && v != nil {
			if m, ok := v.(map[string]interface{}); ok {
				if valid, _ := m["Valid"].(bool); valid {
					r.Spec, _ = m["String"].(string)
				}
			} else if s, ok := v.(string); ok {
				r.Spec = s
			}
		}
		if v, ok := item["tags"]; ok && v != nil {
			if m, ok := v.(map[string]interface{}); ok {
				if valid, _ := m["Valid"].(bool); valid {
					r.Tags, _ = m["String"].(string)
				}
			} else if s, ok := v.(string); ok {
				r.Tags = s
			}
		}
		report.Resources = append(report.Resources, r)
	}
	report.TotalCount = len(report.Resources)

	w, closer, err := jsonWriter(outputPath)
	if err != nil {
		return err
	}
	defer closer()
	return writeJSON(w, report)
}

func strField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func roundTo(f float64, decimals int) float64 {
	switch decimals {
	case 1:
		return float64(int(f*10+0.5)) / 10
	case 2:
		return float64(int(f*100+0.5)) / 100
	default:
		return f
	}
}
