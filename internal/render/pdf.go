package render

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// PrintToPDF renders an article to PDF bytes using Typst.
func PrintToPDF(ctx context.Context, article *model.Article, darkMode bool) ([]byte, error) {
	result, err := renderTypst(article, darkMode)
	if err != nil {
		return nil, fmt.Errorf("generating typst source: %w", err)
	}
	if result.TempDir != "" {
		defer os.RemoveAll(result.TempDir)
	}

	// Create a working directory for Typst compilation.
	workDir, err := os.MkdirTemp("", "typst-compile-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)

	// Write fonts.
	fontDir := filepath.Join(workDir, "fonts")
	if err := os.Mkdir(fontDir, 0755); err != nil {
		return nil, err
	}
	if err := writeFontsToDir(fontDir); err != nil {
		return nil, err
	}

	// Write Typst source.
	typFile := filepath.Join(workDir, "article.typ")
	if err := os.WriteFile(typFile, []byte(result.Source), 0644); err != nil {
		return nil, err
	}

	// Compile to PDF.
	pdfFile := filepath.Join(workDir, "article.pdf")
	cmd := exec.CommandContext(ctx, "typst", "compile",
		"--root", "/",
		"--font-path", fontDir,
		typFile, pdfFile,
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("typst compile: %w", err)
	}

	return os.ReadFile(pdfFile)
}
