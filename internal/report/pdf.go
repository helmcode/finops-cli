package report

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// GeneratePDF converts an HTML file to PDF using headless Chrome via chromedp.
// Returns an error with a clear message if Chrome/Chromium is not available.
func GeneratePDF(htmlPath, pdfPath string) error {
	if !isChromeAvailable() {
		return fmt.Errorf("PDF generation requires Chrome or Chromium to be installed. " +
			"Install Chrome and try again, or use --output html instead")
	}

	absHTML, err := filepath.Abs(htmlPath)
	if err != nil {
		return fmt.Errorf("resolving HTML path: %w", err)
	}

	// Use a wide window so Chart.js canvases render at full size.
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1400, 900),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1400, 900),
		chromedp.Navigate("file://"+absHTML),
		chromedp.WaitReady("body"),
		// Wait for Chart.js to load from CDN and render all canvas charts.
		chromedp.Sleep(3*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithLandscape(true).
				WithScale(0.75).
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("generating PDF: %w", err)
	}

	if err := os.WriteFile(pdfPath, buf, 0o644); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	return nil
}

func isChromeAvailable() bool {
	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	}

	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return true
		}
	}
	return false
}
