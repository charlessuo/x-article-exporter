package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/extract"
	"github.com/annismckenzie/x-article-exporter/internal/images"
	"github.com/annismckenzie/x-article-exporter/internal/render"
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

	articleID, err := extract.ExtractArticleID(cfg.URL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	log.Println("Fetching article...")
	body, err := extract.FetchArticle(ctx, articleID, cfg)
	if err != nil {
		return err
	}

	// Debug: dump raw response to inspect structure
	if os.Getenv("DEBUG") != "" {
		os.WriteFile("debug_response.json", body, 0644)
		log.Println("Raw response written to debug_response.json")
	}

	article, err := extract.ParseArticle(body)
	if err != nil {
		return err
	}

	if len(article.Blocks) == 0 {
		fmt.Fprintln(os.Stderr, "warning: article has no content blocks (content_state may be empty)")
	}

	log.Println("Downloading images...")
	if err := images.DownloadImages(ctx, article); err != nil {
		return err
	}

	log.Println("Rendering HTML...")
	htmlContent := render.RenderHTML(article)

	if os.Getenv("DEBUG") != "" {
		os.WriteFile("debug_render.html", []byte(htmlContent), 0644)
		log.Println("HTML written to debug_render.html")
	}

	log.Println("Generating PDF...")
	pdfBytes, err := render.PrintToPDF(ctx, htmlContent)
	if err != nil {
		return err
	}

	outputPath := cfg.Output
	if outputPath == "" {
		outputPath = sanitizeFilename(article.Title) + ".pdf"
	}

	if err := os.WriteFile(outputPath, pdfBytes, 0644); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Printf("PDF written to %s\n", outputPath)
	return nil
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
