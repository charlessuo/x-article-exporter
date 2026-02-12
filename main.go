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

	basePath := cfg.Output
	if basePath == "" {
		basePath = sanitizeFilename(article.Title)
	} else {
		basePath = strings.TrimSuffix(basePath, ".pdf")
	}

	htmlPath := basePath + ".html"
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("writing HTML: %w", err)
	}
	log.Printf("HTML written to %s", htmlPath)

	log.Println("Generating PDF...")
	pdfBytes, err := render.PrintToPDF(ctx, htmlContent)
	if err != nil {
		return err
	}

	pdfPath := basePath + ".pdf"
	if err := os.WriteFile(pdfPath, pdfBytes, 0644); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Printf("PDF written to %s\n", pdfPath)
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
