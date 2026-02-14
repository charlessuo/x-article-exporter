package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RunFunc is the function signature for the pipeline. Defaults to pipeline.Run.
type RunFunc func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error)

// Server holds the MCP server and its configuration.
type Server struct {
	cfg   *config.MCPConfig
	runFn RunFunc
	srv   *mcpserver.MCPServer
}

// NewServer creates a configured MCP server with all tools registered.
func NewServer(cfg *config.MCPConfig, runFn RunFunc) *Server {
	if runFn == nil {
		runFn = pipeline.Run
	}

	s := &Server{
		cfg:   cfg,
		runFn: runFn,
		srv:   mcpserver.NewMCPServer("x-article-exporter", "1.0.0", mcpserver.WithToolCapabilities(false)),
	}

	s.registerTools()
	return s
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return mcpserver.ServeStdio(s.srv)
}

func (s *Server) registerTools() {
	exportTool := mcplib.NewTool("export_article",
		mcplib.WithDescription("Export an X (Twitter) article as a PDF. Fetches the article, optionally translates it, renders to PDF, and saves to disk. Returns the file path and article metadata."),
		mcplib.WithString("url", mcplib.Required(), mcplib.Description("Full X/Twitter article URL (e.g. https://x.com/user/article/123)")),
		mcplib.WithString("translate", mcplib.Description("Target language code for translation (e.g. 'de', 'fr', 'ja'). Requires Ollama running locally.")),
		mcplib.WithBoolean("dark_mode", mcplib.Description("Render with dark background. Overrides config file setting.")),
		mcplib.WithString("output", mcplib.Description("Custom output file path for the PDF. If not set, uses the configured output directory with the article title as filename.")),
	)
	s.srv.AddTool(exportTool, s.handleExportArticle)

	infoTool := mcplib.NewTool("get_article_info",
		mcplib.WithDescription("Fetch metadata about an X article without rendering a PDF. Returns title, author, date, block count, image count, and word count."),
		mcplib.WithString("url", mcplib.Required(), mcplib.Description("Full X/Twitter article URL")),
	)
	s.srv.AddTool(infoTool, s.handleGetArticleInfo)

	listTool := mcplib.NewTool("list_exports",
		mcplib.WithDescription("List PDF files in the configured output directory."),
		mcplib.WithNumber("limit", mcplib.Description("Maximum number of files to return. Defaults to 20.")),
	)
	s.srv.AddTool(listTool, s.handleListExports)

	translationTool := mcplib.NewTool("check_translation",
		mcplib.WithDescription("Check if the Ollama translation service is available and the configured model is ready."),
	)
	s.srv.AddTool(translationTool, s.handleCheckTranslation)
}

// resolveOutputPath determines the full path for a PDF file.
func (s *Server) resolveOutputPath(customOutput, title string) string {
	if customOutput != "" {
		if !strings.HasSuffix(customOutput, ".pdf") {
			customOutput += ".pdf"
		}
		return customOutput
	}

	filename := sanitizeFilename(title) + ".pdf"
	return filepath.Join(s.cfg.OutputDir, filename)
}

var unsafeFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func sanitizeFilename(title string) string {
	name := strings.TrimSpace(title)
	if name == "" {
		return "article"
	}
	name = unsafeFilenameChars.ReplaceAllString(name, "_")
	name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "article"
	}
	return name
}

// formatFileSize returns a human-readable file size.
func formatFileSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatTime returns a human-readable timestamp.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// ensureDir creates the directory for a file path if it doesn't exist.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}
