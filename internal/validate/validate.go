package validate

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
	pdf "github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpumodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

// ValidatePDF validates a rendered PDF against the source article.
// It returns a Result with hard errors and soft warnings.
// A non-nil error return indicates an infrastructure failure (e.g., library crash),
// not a validation failure.
func ValidatePDF(pdfBytes []byte, article *model.Article) (*Result, error) {
	result := &Result{}
	conf := pdfcpumodel.NewDefaultConfiguration()

	// 1. Structural integrity.
	if err := api.Validate(bytes.NewReader(pdfBytes), conf); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("PDF structural integrity: %v", err))
		return result, nil
	}

	// 2. Page count.
	pageCount, err := api.PageCount(bytes.NewReader(pdfBytes), conf)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("reading page count: %v", err))
		return result, nil
	}
	result.PageCount = pageCount
	if pageCount == 0 {
		result.Errors = append(result.Errors, "PDF has 0 pages")
		return result, nil
	}

	// 3. Image count.
	imagePages, err := api.ExtractImagesRaw(bytes.NewReader(pdfBytes), nil, conf)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("extracting images: %v", err))
	} else {
		pdfImageCount := 0
		for _, pageImages := range imagePages {
			pdfImageCount += len(pageImages)
		}
		result.ImageCount = pdfImageCount
		expected := expectedImageCount(article)
		diff := pdfImageCount - expected
		if diff < 0 {
			diff = -diff
		}
		if diff > 2 {
			result.Errors = append(result.Errors,
				fmt.Sprintf("image count mismatch: PDF has %d, expected %d", pdfImageCount, expected))
		} else if diff > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("image count: PDF has %d, expected %d", pdfImageCount, expected))
		}
	}

	// 4-6. Text extraction (title, author, word count).
	pdfReader, err := pdf.NewReader(bytes.NewReader(pdfBytes), int64(len(pdfBytes)))
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("opening PDF for text extraction: %v", err))
		return result, nil
	}

	textReader, err := pdfReader.GetPlainText()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("extracting text: %v", err))
		return result, nil
	}
	textBytes, err := io.ReadAll(textReader)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("reading extracted text: %v", err))
		return result, nil
	}
	text := string(textBytes)
	pdfWordCount := countWords(text)
	result.WordCount = pdfWordCount
	expectedWords := sourceWordCount(article)

	// Text extraction from Typst-generated PDFs can be unreliable
	// (CID fonts, compressed streams). When extraction quality is poor,
	// downgrade text checks to warnings.
	poorExtraction := expectedWords > 0 && float64(pdfWordCount)/float64(expectedWords) < 0.5

	// 4. Title present.
	if article.Title != "" && !containsNormalized(text, article.Title) {
		msg := "title not found in PDF text"
		if poorExtraction {
			result.Warnings = append(result.Warnings, msg+" (text extraction incomplete)")
		} else {
			result.Errors = append(result.Errors, msg)
		}
	}

	// 5. Author present.
	if article.Author != "" && !containsNormalized(text, extractAuthorName(article.Author)) {
		msg := "author not found in PDF text"
		if poorExtraction {
			result.Warnings = append(result.Warnings, msg+" (text extraction incomplete)")
		} else {
			result.Errors = append(result.Errors, msg)
		}
	}

	// 6. Word count ±15%.
	if expectedWords > 0 {
		ratio := float64(pdfWordCount) / float64(expectedWords)
		if ratio < 0.85 || ratio > 1.15 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("word count: %d (expected ~%d, %.0f%% off)", pdfWordCount, expectedWords, (ratio-1)*100))
		}
	}

	return result, nil
}

// expectedImageCount returns the number of images expected in the PDF.
// This counts images from atomic blocks (actually rendered) plus the cover image.
func expectedImageCount(article *model.Article) int {
	n := article.RenderedImageCount()
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
