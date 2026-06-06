package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/api"
	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/jobs"
	mcpsrv "github.com/annismckenzie/x-article-exporter/internal/mcp"
	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	// Check for --serve/--mcp before normal flag parsing (these modes have no positional URL arg).
	for _, arg := range os.Args[1:] {
		if arg == "--serve" {
			if err := runServer(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}
			return
		}
		if arg == "--mcp" {
			if err := runMCP(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				os.Exit(1)
			}
			return
		}
	}

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

	// Only report PDF validation when a PDF was actually produced.
	pdfFailed := result.PDFError != nil || len(result.PDFBytes) == 0
	if !pdfFailed {
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
	}

	basePath := cfg.OutputBasePath()
	if basePath == "" {
		basePath = sanitizeFilename(result.Title)
	}

	// Create the output directory if it does not exist.
	if dir := filepath.Dir(basePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory %s: %w", dir, err)
		}
	}

	// HTML is always written.
	htmlPath := basePath + ".html"
	if err := os.WriteFile(htmlPath, []byte(result.HTML), 0644); err != nil {
		return fmt.Errorf("writing HTML: %w", err)
	}
	log.Printf("HTML written to %s", htmlPath)

	// The PDF is written only when Typst succeeded.
	if pdfFailed {
		if result.PDFError != nil {
			fmt.Fprintf(os.Stderr, "warning: PDF generation failed, HTML written: %s\n", result.PDFError)
		} else {
			fmt.Fprintf(os.Stderr, "warning: PDF generation failed, HTML written\n")
		}
		return nil
	}

	pdfPath := basePath + ".pdf"
	if err := os.WriteFile(pdfPath, result.PDFBytes, 0644); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Printf("PDF written to %s\n", pdfPath)
	return nil
}

func runServer() error {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		return err
	}

	if cfg.Auth.AuthToken == "" || cfg.Auth.CT0 == "" {
		return errors.New("server mode requires auth_token and ct0 in config file")
	}
	if len(cfg.APIKeys) == 0 {
		return errors.New("server mode requires at least one API key in config file")
	}

	storage := jobs.NewStorage(time.Duration(cfg.JobTTLMinutes) * time.Minute)
	defer storage.Close()

	manager := jobs.NewManager(jobs.ManagerConfig{
		Storage:       storage,
		MaxConcurrent: cfg.MaxConcurrentJobs,
		Auth: jobs.AuthConfig{
			AuthToken:   cfg.Auth.AuthToken,
			CT0:         cfg.Auth.CT0,
			OllamaModel: cfg.Auth.OllamaModel,
		},
	})
	defer manager.Shutdown()

	srv := api.NewServer(cfg, manager)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server listening on %s", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case sig := <-sigCh:
		log.Printf("Received %s, shutting down...", sig)
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("Server stopped.")
	return nil
}

func runMCP() error {
	cfg, err := config.LoadMCPConfig()
	if err != nil {
		return err
	}

	if cfg.AuthToken == "" || cfg.CT0 == "" {
		return errors.New("MCP mode requires auth_token and ct0 in config file")
	}

	srv := mcpsrv.NewServer(cfg, nil)
	return srv.ServeStdio()
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
