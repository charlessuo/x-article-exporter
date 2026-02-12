package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/extract"
	"github.com/annismckenzie/x-article-exporter/internal/model"
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

	printSummary(article)
	return nil
}

func printSummary(a *model.Article) {
	fmt.Printf("Article: %s\n", a.Title)

	if a.Author != "" {
		fmt.Printf("Author:  %s\n", a.Author)
	}

	if !a.PublishedAt.IsZero() {
		fmt.Printf("Date:    %s\n", a.PublishedAt.Format("2006-01-02"))
	}

	if a.CoverImageURL != "" {
		fmt.Printf("Cover:   %s\n", a.CoverImageURL)
	}

	fmt.Println()

	// Block counts
	blockCounts := a.BlockCounts()
	fmt.Printf("Blocks: %d\n", len(a.Blocks))
	if len(blockCounts) > 0 {
		printCounts(blockCounts)
	}

	fmt.Println()

	// Entity counts
	entityCounts := a.EntityCounts()
	fmt.Printf("Entities: %d\n", len(a.EntityMap))
	if len(entityCounts) > 0 {
		printCounts(entityCounts)
	}

	fmt.Println()
	fmt.Printf("Images: %d (from entity map)\n", a.ImageCount())
}

func printCounts(counts map[string]int) {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Find max key length for alignment
	maxLen := 0
	for _, k := range keys {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	for _, k := range keys {
		padding := strings.Repeat(" ", maxLen-len(k))
		fmt.Printf("  %s:%s %d\n", k, padding, counts[k])
	}
}
