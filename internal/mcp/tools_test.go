package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func mockRunFunc(result *pipeline.Result, err error) RunFunc {
	return func(_ context.Context, _ pipeline.Options) (*pipeline.Result, error) {
		return result, err
	}
}

func newTestConfig(outputDir string) *config.MCPConfig {
	return &config.MCPConfig{
		AuthToken: "test-token",
		CT0:       "test-ct0",
		OutputDir: outputDir,
	}
}

func makeRequest(args map[string]any) mcplib.CallToolRequest {
	return mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Arguments: args,
		},
	}
}

func TestExportArticle_Success(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	result := &pipeline.Result{
		Title:        "Test Article",
		Author:       "Test Author (@test)",
		PDFBytes:     []byte("%PDF-1.4 test"),
		PageCount:    2,
		ImageCount:   1,
		WordCount:    500,
		ValidationOK: true,
	}

	srv := NewServer(cfg, mockRunFunc(result, nil))
	ctx := context.Background()

	req := makeRequest(map[string]any{
		"url": "https://x.com/user/article/12345",
	})

	res, err := srv.handleExportArticle(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}

	text := res.Content[0].(mcplib.TextContent).Text
	if !strings.Contains(text, "PDF exported successfully") {
		t.Errorf("expected success message, got: %s", text)
	}
	if !strings.Contains(text, "Test Article") {
		t.Errorf("expected title in output, got: %s", text)
	}

	// Verify file was written.
	pdfPath := filepath.Join(dir, "Test Article.pdf")
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("expected PDF at %s: %v", pdfPath, err)
	}
	if string(data) != "%PDF-1.4 test" {
		t.Errorf("PDF content = %q, want %%PDF-1.4 test", data)
	}
}

func TestExportArticle_CustomOutput(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	result := &pipeline.Result{
		Title:        "Test",
		PDFBytes:     []byte("pdf"),
		ValidationOK: true,
	}

	srv := NewServer(cfg, mockRunFunc(result, nil))
	ctx := context.Background()

	customPath := filepath.Join(dir, "custom.pdf")
	req := makeRequest(map[string]any{
		"url":    "https://x.com/user/article/12345",
		"output": customPath,
	})

	res, err := srv.handleExportArticle(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}

	if _, err := os.Stat(customPath); err != nil {
		t.Errorf("expected file at %s: %v", customPath, err)
	}
}

func TestExportArticle_InvalidURL(t *testing.T) {
	cfg := newTestConfig(".")
	srv := NewServer(cfg, mockRunFunc(nil, nil))

	req := makeRequest(map[string]any{
		"url": "https://google.com/not-an-article",
	})

	res, err := srv.handleExportArticle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("expected tool error for invalid URL")
	}
}

func TestExportArticle_MissingURL(t *testing.T) {
	cfg := newTestConfig(".")
	srv := NewServer(cfg, mockRunFunc(nil, nil))

	req := makeRequest(map[string]any{})

	res, err := srv.handleExportArticle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("expected tool error for missing URL")
	}
}

func TestExportArticle_DarkModeOverride(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	cfg.DarkMode = false

	var capturedOpts pipeline.Options
	runFn := func(_ context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		capturedOpts = opts
		return &pipeline.Result{
			Title:        "Test",
			PDFBytes:     []byte("pdf"),
			ValidationOK: true,
		}, nil
	}

	srv := NewServer(cfg, runFn)
	req := makeRequest(map[string]any{
		"url":       "https://x.com/user/article/12345",
		"dark_mode": true,
	})

	_, err := srv.handleExportArticle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedOpts.DarkMode {
		t.Error("expected dark mode to be overridden to true")
	}
}

func TestExportArticle_ProgressCallback(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	var progressMessages []string
	srv := NewServer(cfg, func(_ context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		// Simulate the pipeline calling OnProgress at each step.
		if opts.OnProgress != nil {
			opts.OnProgress("Fetching article...")
			opts.OnProgress("Downloading images...")
			opts.OnProgress("Rendering PDF...")
			opts.OnProgress("Validating PDF...")
		}
		return &pipeline.Result{
			Title:        "Progress Test",
			PDFBytes:     []byte("pdf"),
			ValidationOK: true,
		}, nil
	})

	// Capture OnProgress calls by wrapping the server's runFn.
	origRunFn := srv.runFn
	srv.runFn = func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		orig := opts.OnProgress
		opts.OnProgress = func(msg string) {
			progressMessages = append(progressMessages, msg)
			if orig != nil {
				orig(msg)
			}
		}
		return origRunFn(ctx, opts)
	}

	req := makeRequest(map[string]any{
		"url": "https://x.com/user/article/12345",
	})

	res, err := srv.handleExportArticle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}

	// Verify pipeline progress messages are forwarded.
	want := []string{
		"Fetching article...",
		"Downloading images...",
		"Rendering PDF...",
		"Validating PDF...",
	}
	if len(progressMessages) != len(want) {
		t.Fatalf("got %d progress messages, want %d: %v", len(progressMessages), len(want), progressMessages)
	}
	for i, msg := range want {
		if progressMessages[i] != msg {
			t.Errorf("progress[%d] = %q, want %q", i, progressMessages[i], msg)
		}
	}
}

func TestExportArticle_PipelineError(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	srv := NewServer(cfg, mockRunFunc(nil, os.ErrPermission))

	req := makeRequest(map[string]any{
		"url": "https://x.com/user/article/12345",
	})

	res, err := srv.handleExportArticle(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Error("expected tool error for pipeline failure")
	}
	text := res.Content[0].(mcplib.TextContent).Text
	if !strings.Contains(text, "Export failed") {
		t.Errorf("expected 'Export failed', got: %s", text)
	}
}

func TestListExports_WithPDFs(t *testing.T) {
	dir := t.TempDir()

	// Create some test PDFs.
	for _, name := range []string{"article1.pdf", "article2.pdf", "not-a-pdf.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := newTestConfig(dir)
	srv := NewServer(cfg, nil)

	req := makeRequest(map[string]any{})
	res, err := srv.handleListExports(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}

	text := res.Content[0].(mcplib.TextContent).Text
	if !strings.Contains(text, "article1.pdf") {
		t.Errorf("expected article1.pdf in output, got: %s", text)
	}
	if !strings.Contains(text, "article2.pdf") {
		t.Errorf("expected article2.pdf in output, got: %s", text)
	}
	if strings.Contains(text, "not-a-pdf.txt") {
		t.Errorf("should not include non-PDF files, got: %s", text)
	}
}

func TestListExports_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	srv := NewServer(cfg, nil)

	req := makeRequest(map[string]any{})
	res, err := srv.handleListExports(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := res.Content[0].(mcplib.TextContent).Text
	if !strings.Contains(text, "No PDF files") {
		t.Errorf("expected 'No PDF files' message, got: %s", text)
	}
}

func TestListExports_NonexistentDir(t *testing.T) {
	cfg := newTestConfig("/nonexistent/path/that/does/not/exist")
	srv := NewServer(cfg, nil)

	req := makeRequest(map[string]any{})
	res, err := srv.handleListExports(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := res.Content[0].(mcplib.TextContent).Text
	if !strings.Contains(text, "does not exist") {
		t.Errorf("expected 'does not exist' message, got: %s", text)
	}
}

func TestListExports_WithLimit(t *testing.T) {
	dir := t.TempDir()
	for i := range 5 {
		name := filepath.Join(dir, strings.Replace("article_N.pdf", "N", string(rune('1'+i)), 1))
		if err := os.WriteFile(name, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := newTestConfig(dir)
	srv := NewServer(cfg, nil)

	req := makeRequest(map[string]any{
		"limit": float64(2),
	})
	res, err := srv.handleListExports(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := res.Content[0].(mcplib.TextContent).Text
	if !strings.Contains(text, "(2 files)") {
		t.Errorf("expected (2 files), got: %s", text)
	}
}

func TestResolveOutputPath(t *testing.T) {
	cfg := newTestConfig("/exports")
	srv := NewServer(cfg, nil)

	// Default: uses output dir + title.
	path := srv.resolveOutputPath("", "My Article")
	if path != "/exports/My Article.pdf" {
		t.Errorf("got %q, want /exports/My Article.pdf", path)
	}

	// Custom path without .pdf.
	path = srv.resolveOutputPath("/tmp/custom", "ignored")
	if path != "/tmp/custom.pdf" {
		t.Errorf("got %q, want /tmp/custom.pdf", path)
	}

	// Custom path with .pdf.
	path = srv.resolveOutputPath("/tmp/custom.pdf", "ignored")
	if path != "/tmp/custom.pdf" {
		t.Errorf("got %q, want /tmp/custom.pdf", path)
	}

	// Empty title fallback.
	path = srv.resolveOutputPath("", "")
	if path != "/exports/article.pdf" {
		t.Errorf("got %q, want /exports/article.pdf", path)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Hello World", "Hello World"},
		{"Hello/World", "Hello_World"},
		{"", "article"},
		{"<script>", "script"},
		{"a::b??c", "a_b_c"},
	}
	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
