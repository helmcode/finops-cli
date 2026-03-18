package report

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"runtime"
	"time"
)

//go:embed templates/*.html
var templatesFS embed.FS

// funcMap provides helper functions for templates.
var funcMap = template.FuncMap{
	"add":  func(a, b int) int { return a + b },
	"divf": func(a, b float64) float64 { if b == 0 { return 0 }; return a / b },
	"mulf": func(a, b float64) float64 { return a * b },
	"max":  func(a, b float64) float64 { if a > b { return a }; return b },
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
