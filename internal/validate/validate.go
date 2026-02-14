package validate

import (
	"fmt"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// Result holds the outcome of PDF validation.
type Result struct {
	PageCount  int
	ImageCount int
	WordCount  int
	Errors     []string // Hard failures — caller should exit 1.
	Warnings   []string // Soft warnings — caller should log and continue.
}

// OK returns true if there are no hard failures.
func (r *Result) OK() bool {
	return len(r.Errors) == 0
}

// String returns a human-readable summary of validation results.
func (r *Result) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "PDF valid. %d pages. %d images. %d words.", r.PageCount, r.ImageCount, r.WordCount)
	if len(r.Warnings) > 0 {
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, " Warning: %s.", w)
		}
	}
	if r.OK() {
		b.WriteString(" OK.")
	}
	return b.String()
}

// expectedImageCount returns the number of images expected in the PDF.
// This includes entity map images (IMAGE/MEDIA) plus the cover image if present.
func expectedImageCount(article *model.Article) int {
	n := article.ImageCount()
	if article.CoverImageURL != "" {
		n++
	}
	return n
}

// sourceWordCount computes the baseline word count from article blocks.
// It excludes code blocks and atomic blocks (images).
func sourceWordCount(article *model.Article) int {
	var total int
	for _, block := range article.Blocks {
		switch block.Type {
		case "code-block", "atomic":
			continue
		}
		total += countWords(block.Text)
	}
	total += countWords(article.Title)
	total += countWords(article.Author)
	return total
}

// countWords counts whitespace-delimited words in a string.
func countWords(s string) int {
	return len(strings.Fields(s))
}

// containsNormalized checks if haystack contains needle, case-insensitive,
// after collapsing whitespace. PDF text extraction can introduce whitespace
// differences.
func containsNormalized(haystack, needle string) bool {
	h := normalizeWhitespace(strings.ToLower(haystack))
	n := normalizeWhitespace(strings.ToLower(needle))
	return strings.Contains(h, n)
}

// normalizeWhitespace collapses all runs of whitespace to a single space.
func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// extractAuthorName extracts the @screenname portion from the author string.
// Author format: "Name (@screenname)" or "@screenname".
func extractAuthorName(author string) string {
	if i := strings.Index(author, "@"); i >= 0 {
		rest := author[i:]
		rest = strings.TrimRight(rest, ")")
		return rest
	}
	return author
}
