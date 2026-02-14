package pipeline

import (
	"context"
	"fmt"
	"log"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/extract"
	"github.com/annismckenzie/x-article-exporter/internal/images"
	"github.com/annismckenzie/x-article-exporter/internal/model"
	"github.com/annismckenzie/x-article-exporter/internal/render"
	"github.com/annismckenzie/x-article-exporter/internal/translate"
	"github.com/annismckenzie/x-article-exporter/internal/validate"
)

// Options configures a single export run.
// Unlike config.Config, it omits Output (the caller writes files).
type Options struct {
	URL         string
	TranslateTo string
	AuthToken   string
	CT0         string
	QueryID     string
	OllamaModel string
	DarkMode    bool

	// OnProgress is called at the start of each pipeline step with a
	// human-readable message (e.g. "Fetching article..."). Nil means no-op.
	OnProgress func(message string)
}

// Result holds everything the caller needs after a successful pipeline run.
type Result struct {
	Article            *model.Article
	PDFBytes           []byte
	HTML               string
	Title, Author      string
	PageCount          int
	ImageCount         int
	WordCount          int
	ValidationOK       bool
	ValidationWarnings []string
	ValidationErrors   []string
}

// Run executes the full article export pipeline: fetch, translate, render, validate.
// It returns a Result with PDF bytes and metadata, or an error.
func Run(ctx context.Context, opts Options) (*Result, error) {
	progress := opts.OnProgress
	if progress == nil {
		progress = func(string) {}
	}

	articleID, err := extract.ExtractArticleID(opts.URL)
	if err != nil {
		return nil, err
	}

	queryID, err := extract.ResolveQueryID(ctx, opts.QueryID)
	if err != nil {
		return nil, err
	}

	progress("Fetching article...")
	log.Println("Fetching article...")
	cfg := &config.Config{AuthToken: opts.AuthToken, CT0: opts.CT0}
	body, err := extract.FetchArticle(ctx, articleID, queryID, cfg)
	if err != nil {
		return nil, err
	}

	article, err := extract.ParseArticle(body)
	if err != nil {
		return nil, err
	}

	if len(article.Blocks) == 0 {
		return nil, fmt.Errorf("article has no content blocks (content_state may be empty)")
	}

	if opts.TranslateTo != "" {
		progress("Translating to " + opts.TranslateTo + "...")
		client := translate.NewClient("", opts.OllamaModel)
		if err := client.Ping(ctx); err != nil {
			return nil, err
		}
		log.Printf("Translating to %s...", opts.TranslateTo)
		if err := translate.TranslateArticle(ctx, article, opts.TranslateTo, client); err != nil {
			return nil, err
		}
	}

	progress("Downloading images...")
	log.Println("Downloading images...")
	if err := images.DownloadImages(ctx, article); err != nil {
		return nil, err
	}

	progress("Rendering PDF...")
	log.Println("Rendering HTML...")
	htmlContent := render.RenderHTML(article)

	log.Println("Generating PDF...")
	pdfBytes, err := render.PrintToPDF(ctx, article, opts.DarkMode)
	if err != nil {
		return nil, err
	}

	progress("Validating PDF...")
	log.Println("Validating PDF...")
	valResult, err := validate.ValidatePDF(pdfBytes, article)
	if err != nil {
		return nil, fmt.Errorf("PDF validation: %w", err)
	}

	return &Result{
		Article:            article,
		PDFBytes:           pdfBytes,
		HTML:               htmlContent,
		Title:              article.Title,
		Author:             article.Author,
		PageCount:          valResult.PageCount,
		ImageCount:         valResult.ImageCount,
		WordCount:          valResult.WordCount,
		ValidationOK:       valResult.OK(),
		ValidationWarnings: valResult.Warnings,
		ValidationErrors:   valResult.Errors,
	}, nil
}
