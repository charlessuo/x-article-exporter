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

	// PDFError is set when HTML rendering succeeded but the PDF/Typst step
	// failed. In that case PDFBytes is nil and the caller should still write
	// the HTML. It is nil when the PDF was produced successfully.
	PDFError error
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

	progress("Rendering HTML...")
	log.Println("Rendering HTML...")
	htmlContent := render.RenderHTML(article)

	// HTML is always kept. The PDF is best-effort: a Typst compile failure
	// (e.g. malformed markup in the article prose) must not lose the HTML, so
	// we record the error and continue instead of aborting the whole run.
	result := &Result{
		Article: article,
		HTML:    htmlContent,
		Title:   article.Title,
		Author:  article.Author,
	}

	progress("Generating PDF...")
	log.Println("Generating PDF...")
	pdfBytes, err := render.PrintToPDF(ctx, article, opts.DarkMode)
	if err != nil {
		result.PDFError = err
		return result, nil
	}
	result.PDFBytes = pdfBytes

	progress("Validating PDF...")
	log.Println("Validating PDF...")
	valResult, err := validate.ValidatePDF(pdfBytes, article)
	if err != nil {
		// A validator crash must not lose the HTML either — degrade the same way
		// a Typst/PDF generation failure does: drop the PDF, keep the HTML.
		result.PDFError = fmt.Errorf("PDF validation: %w", err)
		result.PDFBytes = nil
		return result, nil
	}

	result.PageCount = valResult.PageCount
	result.ImageCount = valResult.ImageCount
	result.WordCount = valResult.WordCount
	result.ValidationOK = valResult.OK()
	result.ValidationWarnings = valResult.Warnings
	result.ValidationErrors = valResult.Errors

	return result, nil
}
