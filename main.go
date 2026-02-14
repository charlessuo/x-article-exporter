package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.ParseFlags(args)
	if err != nil {
		return err
	}

	ctx := context.Background()

	result, err := pipeline.Run(ctx, pipeline.Options{
		URL:         cfg.URL,
		TranslateTo: cfg.TranslateTo,
		AuthToken:   cfg.AuthToken,
		CT0:         cfg.CT0,
		QueryID:     cfg.QueryID,
		OllamaModel: cfg.OllamaModel,
		DarkMode:    cfg.DarkMode,
	})
	if err != nil {
		return err
	}

	log.Print(formatValidation(result))
	for _, w := range result.ValidationWarnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
	if !result.ValidationOK {
		for _, e := range result.ValidationErrors {
			fmt.Fprintf(os.Stderr, "validation error: %s\n", e)
		}
		return fmt.Errorf("PDF validation failed with %d error(s)", len(result.ValidationErrors))
	}

	basePath := cfg.Output
	if basePath == "" {
		basePath = sanitizeFilename(result.Title)
	} else {
		basePath = strings.TrimSuffix(basePath, ".pdf")
	}

	htmlPath := basePath + ".html"
	if err := os.WriteFile(htmlPath, []byte(result.HTML), 0644); err != nil {
		return fmt.Errorf("writing HTML: %w", err)
	}
	log.Printf("HTML written to %s", htmlPath)

	pdfPath := basePath + ".pdf"
	if err := os.WriteFile(pdfPath, result.PDFBytes, 0644); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Printf("PDF written to %s\n", pdfPath)
	return nil
}

func formatValidation(r *pipeline.Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PDF valid. %d pages. %d images. %d words.", r.PageCount, r.ImageCount, r.WordCount)
	if len(r.ValidationWarnings) > 0 {
		for _, w := range r.ValidationWarnings {
			fmt.Fprintf(&b, " Warning: %s.", w)
		}
	}
	if r.ValidationOK {
		b.WriteString(" OK.")
	}
	return b.String()
}

// sanitizeFilename replaces characters that are invalid in filenames.
var unsafeFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizeFilename(title string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		return "article"
	}
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	// Collapse multiple underscores.
	name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "article"
	}
	return name
}
