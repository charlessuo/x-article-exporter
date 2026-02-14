package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/extract"
	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
	"github.com/annismckenzie/x-article-exporter/internal/translate"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleExportArticle(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	url, err := request.RequireString("url")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	// Validate URL early.
	if _, err := extract.ExtractArticleID(url); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Invalid article URL: %s", err)), nil
	}

	translateTo := request.GetString("translate", "")
	customOutput := request.GetString("output", "")
	darkMode := request.GetBool("dark_mode", s.cfg.DarkMode)

	opts := pipeline.Options{
		URL:         url,
		TranslateTo: translateTo,
		AuthToken:   s.cfg.AuthToken,
		CT0:         s.cfg.CT0,
		OllamaModel: s.cfg.OllamaModel,
		DarkMode:    darkMode,
	}

	result, err := s.runFn(ctx, opts)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Export failed: %s", err)), nil
	}

	outputPath := s.resolveOutputPath(customOutput, result.Title)

	if err := ensureDir(outputPath); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to create output directory: %s", err)), nil
	}

	if err := os.WriteFile(outputPath, result.PDFBytes, 0644); err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to write PDF: %s", err)), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "PDF exported successfully.\n\n")
	fmt.Fprintf(&b, "File: %s\n", outputPath)
	fmt.Fprintf(&b, "Title: %s\n", result.Title)
	fmt.Fprintf(&b, "Author: %s\n", result.Author)
	fmt.Fprintf(&b, "Pages: %d\n", result.PageCount)
	fmt.Fprintf(&b, "Images: %d\n", result.ImageCount)
	fmt.Fprintf(&b, "Words: %d\n", result.WordCount)
	if translateTo != "" {
		fmt.Fprintf(&b, "Translated to: %s\n", translateTo)
	}
	if darkMode {
		b.WriteString("Mode: dark\n")
	}
	if !result.ValidationOK {
		fmt.Fprintf(&b, "\nValidation warnings: %s\n", strings.Join(result.ValidationWarnings, "; "))
		fmt.Fprintf(&b, "Validation errors: %s\n", strings.Join(result.ValidationErrors, "; "))
	}

	return mcplib.NewToolResultText(b.String()), nil
}

func (s *Server) handleGetArticleInfo(ctx context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	url, err := request.RequireString("url")
	if err != nil {
		return mcplib.NewToolResultError(err.Error()), nil
	}

	articleID, err := extract.ExtractArticleID(url)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Invalid article URL: %s", err)), nil
	}

	queryID, err := extract.ResolveQueryID(ctx, "")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to resolve query ID: %s", err)), nil
	}

	cfg := &config.Config{AuthToken: s.cfg.AuthToken, CT0: s.cfg.CT0}
	body, err := extract.FetchArticle(ctx, articleID, queryID, cfg)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to fetch article: %s", err)), nil
	}

	article, err := extract.ParseArticle(body)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to parse article: %s", err)), nil
	}

	// Count words across all text blocks.
	wordCount := 0
	for _, block := range article.Blocks {
		text := strings.TrimSpace(block.Text)
		if text != "" {
			wordCount += len(strings.Fields(text))
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", article.Title)
	fmt.Fprintf(&b, "Author: %s\n", article.Author)
	if !article.PublishedAt.IsZero() {
		fmt.Fprintf(&b, "Published: %s\n", formatTime(article.PublishedAt))
	}
	if !article.ModifiedAt.IsZero() {
		fmt.Fprintf(&b, "Modified: %s\n", formatTime(article.ModifiedAt))
	}
	fmt.Fprintf(&b, "Blocks: %d\n", len(article.Blocks))
	fmt.Fprintf(&b, "Images: %d\n", article.RenderedImageCount())
	fmt.Fprintf(&b, "Words: %d\n", wordCount)

	if article.CoverImageURL != "" {
		fmt.Fprintf(&b, "Cover image: %s\n", article.CoverImageURL)
	}

	blockCounts := article.BlockCounts()
	if len(blockCounts) > 0 {
		b.WriteString("\nBlock types:\n")
		for typ, count := range blockCounts {
			fmt.Fprintf(&b, "  %s: %d\n", typ, count)
		}
	}

	return mcplib.NewToolResultText(b.String()), nil
}

func (s *Server) handleListExports(_ context.Context, request mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	limit := int(request.GetFloat("limit", 20))
	if limit <= 0 {
		limit = 20
	}

	dir := s.cfg.OutputDir
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return mcplib.NewToolResultText(fmt.Sprintf("Output directory %s does not exist. No exports yet.", dir)), nil
		}
		return mcplib.NewToolResultError(fmt.Sprintf("Failed to read directory %s: %s", dir, err)), nil
	}

	// Filter to PDF files and collect info.
	type pdfFile struct {
		name    string
		size    int64
		modTime string
	}
	var pdfs []pdfFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		pdfs = append(pdfs, pdfFile{
			name:    entry.Name(),
			size:    info.Size(),
			modTime: formatTime(info.ModTime()),
		})
	}

	// Sort by modification time, newest first.
	sort.Slice(pdfs, func(i, j int) bool {
		return pdfs[i].modTime > pdfs[j].modTime
	})

	if len(pdfs) == 0 {
		return mcplib.NewToolResultText(fmt.Sprintf("No PDF files found in %s.", dir)), nil
	}

	if len(pdfs) > limit {
		pdfs = pdfs[:limit]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "PDF exports in %s (%d files):\n\n", dir, len(pdfs))
	for _, pdf := range pdfs {
		fmt.Fprintf(&b, "- %s (%s, %s)\n", pdf.name, formatFileSize(pdf.size), pdf.modTime)
	}

	return mcplib.NewToolResultText(b.String()), nil
}

func (s *Server) handleCheckTranslation(ctx context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	model := s.cfg.OllamaModel
	client := translate.NewClient("", model)

	if err := client.Ping(ctx); err != nil {
		return mcplib.NewToolResultText(fmt.Sprintf("Translation service is NOT available.\n\nError: %s\n\nMake sure Ollama is running (ollama serve) and the model is pulled (ollama pull %s).", err, model)), nil
	}

	return mcplib.NewToolResultText(fmt.Sprintf("Translation service is available.\n\nOllama: running\nModel: %s\nEndpoint: %s\n\nYou can use the 'translate' parameter with export_article to translate articles.", model, filepath.Join(client.BaseURL, "api/chat"))), nil
}
