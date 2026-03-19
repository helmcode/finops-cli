package report

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/helmcode/finops-cli/internal/analysis"
	"github.com/helmcode/finops-cli/internal/store"
)

//go:embed templates/*.html
var templatesFS embed.FS

// formatMoney formats a float as a money string with thousand separators.
// e.g., 39990.5678 → "39,990.57"
func formatMoney(f float64) string {
	negative := f < 0
	if negative {
		f = -f
	}

	// Use Sprintf for correct rounding, then insert commas
	formatted := fmt.Sprintf("%.2f", f)
	parts := strings.SplitN(formatted, ".", 2)
	intStr := parts[0]
	decStr := parts[1]

	// Add thousand separators to integer part
	if len(intStr) > 3 {
		var groups []string
		for len(intStr) > 3 {
			groups = append([]string{intStr[len(intStr)-3:]}, groups...)
			intStr = intStr[:len(intStr)-3]
		}
		groups = append([]string{intStr}, groups...)
		intStr = strings.Join(groups, ",")
	}

	result := intStr + "." + decStr
	if negative {
		return "-" + result
	}
	return result
}

// funcMap provides helper functions for templates.
var funcMap = template.FuncMap{
	"add":         func(a, b int) int { return a + b },
	"divf":        func(a, b float64) float64 { if b == 0 { return 0 }; return a / b },
	"mulf":        func(a, b float64) float64 { return a * b },
	"max":         func(a, b float64) float64 { if a > b { return a }; return b },
	"formatMoney": formatMoney,
	"formatPct":   func(f float64) string { return fmt.Sprintf("%.1f", f) },
	"gt0":         func(f float64) bool { return f > 0 },
}

// RegionServiceCost holds the cost of a single service within a region.
type RegionServiceCost struct {
	Service string
	Amount  float64
}

// RegionDetail holds a region's cost and its associated resources.
type RegionDetail struct {
	Region       string
	TotalAmount  float64
	Currency     string
	Resources    []store.Resource
	ServiceCosts []RegionServiceCost
}

// AccountDetail holds cost and resource data for a single account.
type AccountDetail struct {
	AccountID     string
	TotalAmount   float64
	Currency      string
	ResourceCount int64
	TopServices   []AccountServiceCost
}

// AccountServiceCost holds the cost of a single service within an account.
type AccountServiceCost struct {
	Service string
	Amount  float64
}

// ReportData is the common data structure passed to all templates.
type ReportData struct {
	Title          string
	GeneratedAt    string
	PeriodStart    string
	PeriodEnd      string
	Data           interface{}
	TotalResources int64
	MonthCount     float64
	RegionDetails  []RegionDetail
	MonthlySpend       []analysis.MonthlyDataPoint
	AccountDetails     []AccountDetail
	CommitmentOverview interface{}
}

// GenerateHTML renders a report template to an HTML file.
func GenerateHTML(templateName, outputPath string, data ReportData) error {
	if data.GeneratedAt == "" {
		data.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/base.html", "templates/"+templateName+".html")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, templateName+".html", data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

// OpenInBrowser opens the given file in the default browser.
func OpenInBrowser(path string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "start"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return exec.Command(cmd, path).Start()
}
