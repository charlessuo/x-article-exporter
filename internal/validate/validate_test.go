package validate

import (
	"context"
	"testing"

	"github.com/annismckenzie/x-article-exporter/internal/model"
	"github.com/annismckenzie/x-article-exporter/internal/render"
)

func TestCountWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  multiple   spaces  ", 2},
		{"one\ttab\nnewline", 3},
	}
	for _, tt := range tests {
		if got := countWords(tt.input); got != tt.want {
			t.Errorf("countWords(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello  world", "hello world"},
		{"  leading and trailing  ", "leading and trailing"},
		{"tabs\tand\nnewlines", "tabs and newlines"},
	}
	for _, tt := range tests {
		if got := normalizeWhitespace(tt.input); got != tt.want {
			t.Errorf("normalizeWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestContainsNormalized(t *testing.T) {
	tests := []struct {
		name     string
		haystack string
		needle   string
		want     bool
	}{
		{"exact match", "Hello World", "Hello World", true},
		{"case insensitive", "Hello World", "hello world", true},
		{"extra whitespace in haystack", "Hello   World", "Hello World", true},
		{"extra whitespace in needle", "Hello World", "Hello   World", true},
		{"substring", "The Quick Brown Fox", "quick brown", true},
		{"not found", "Hello World", "Goodbye", false},
		{"empty needle", "Hello", "", true},
		{"empty haystack", "", "Hello", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsNormalized(tt.haystack, tt.needle); got != tt.want {
				t.Errorf("containsNormalized(%q, %q) = %v, want %v", tt.haystack, tt.needle, got, tt.want)
			}
		})
	}
}

func TestExtractAuthorName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"@testuser", "@testuser"},
		{"Test User (@testuser)", "@testuser"},
		{"No handle", "No handle"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := extractAuthorName(tt.input); got != tt.want {
			t.Errorf("extractAuthorName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpectedImageCount(t *testing.T) {
	tests := []struct {
		name    string
		article *model.Article
		want    int
	}{
		{
			name:    "no images",
			article: &model.Article{EntityMap: map[string]model.Entity{}},
			want:    0,
		},
		{
			name: "rendered images only",
			article: &model.Article{
				Blocks: []model.Block{
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 0}}},
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 1}}},
				},
				EntityMap: map[string]model.Entity{
					"0": {Type: "MEDIA"},
					"1": {Type: "MEDIA"},
				},
			},
			want: 2,
		},
		{
			name: "rendered images plus cover",
			article: &model.Article{
				CoverImageURL: "data:image/png;base64,abc",
				Blocks: []model.Block{
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 0}}},
				},
				EntityMap: map[string]model.Entity{
					"0": {Type: "MEDIA"},
				},
			},
			want: 2,
		},
		{
			name: "cover image only",
			article: &model.Article{
				CoverImageURL: "data:image/png;base64,abc",
				EntityMap:     map[string]model.Entity{},
			},
			want: 1,
		},
		{
			name: "non-image entities excluded",
			article: &model.Article{
				Blocks: []model.Block{
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 0}}},
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 1}}},
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 2}}},
				},
				EntityMap: map[string]model.Entity{
					"0": {Type: "MEDIA"},
					"1": {Type: "LINK"},
					"2": {Type: "TWEMOJI"},
				},
			},
			want: 1,
		},
		{
			name: "unreferenced entity map entries not counted",
			article: &model.Article{
				Blocks: []model.Block{
					{Type: "atomic", EntityRanges: []model.EntityRange{{Key: 0}}},
				},
				EntityMap: map[string]model.Entity{
					"0": {Type: "MEDIA"},
					"1": {Type: "MEDIA"}, // not referenced by any block
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expectedImageCount(tt.article); got != tt.want {
				t.Errorf("expectedImageCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSourceWordCount(t *testing.T) {
	tests := []struct {
		name    string
		article *model.Article
		want    int
	}{
		{
			name: "simple paragraphs",
			article: &model.Article{
				Title:  "My Title",
				Author: "@author",
				Blocks: []model.Block{
					{Type: "unstyled", Text: "Hello world"},
					{Type: "unstyled", Text: "Another paragraph here"},
				},
			},
			want: 2 + 1 + 2 + 3, // title + author + block1 + block2
		},
		{
			name: "code blocks excluded",
			article: &model.Article{
				Title: "Title",
				Blocks: []model.Block{
					{Type: "unstyled", Text: "Some text"},
					{Type: "code-block", Text: "func main() { fmt.Println() }"},
					{Type: "unstyled", Text: "More text"},
				},
			},
			want: 1 + 2 + 2, // title + block1 + block3 (code excluded)
		},
		{
			name: "atomic blocks excluded",
			article: &model.Article{
				Title: "Title",
				Blocks: []model.Block{
					{Type: "unstyled", Text: "Text here"},
					{Type: "atomic", Text: " "},
				},
			},
			want: 1 + 2, // title + block1 (atomic excluded)
		},
		{
			name: "headers and lists included",
			article: &model.Article{
				Blocks: []model.Block{
					{Type: "header-one", Text: "Big Header"},
					{Type: "unordered-list-item", Text: "List item one"},
					{Type: "blockquote", Text: "A quote"},
				},
			},
			want: 2 + 3 + 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sourceWordCount(tt.article); got != tt.want {
				t.Errorf("sourceWordCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResultOK(t *testing.T) {
	tests := []struct {
		name   string
		result Result
		want   bool
	}{
		{"no errors", Result{}, true},
		{"with warnings only", Result{Warnings: []string{"word count off"}}, true},
		{"with errors", Result{Errors: []string{"missing title"}}, false},
		{"errors and warnings", Result{Errors: []string{"bad"}, Warnings: []string{"meh"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.OK(); got != tt.want {
				t.Errorf("OK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResultString(t *testing.T) {
	r := &Result{PageCount: 4, ImageCount: 3, WordCount: 1847}
	got := r.String()
	if got != "PDF valid. 4 pages. 3 images. 1847 words. OK." {
		t.Errorf("String() = %q", got)
	}

	r.Warnings = append(r.Warnings, "word count: 1847 (expected ~1600, 15% off)")
	got = r.String()
	if got != "PDF valid. 4 pages. 3 images. 1847 words. Warning: word count: 1847 (expected ~1600, 15% off). OK." {
		t.Errorf("String() with warning = %q", got)
	}
}

func TestValidatePDF_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test (requires Chrome)")
	}

	html := `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h1>Test Article Title</h1>
<p>by @testauthor</p>
<p>This is the body of the test article with several words in it for counting.</p>
</body></html>`

	ctx := context.Background()
	pdfBytes, err := render.PrintToPDF(ctx, html)
	if err != nil {
		t.Fatalf("PrintToPDF: %v", err)
	}

	article := &model.Article{
		Title:  "Test Article Title",
		Author: "@testauthor",
		Blocks: []model.Block{
			{Type: "unstyled", Text: "This is the body of the test article with several words in it for counting."},
		},
		EntityMap: map[string]model.Entity{},
	}

	result, err := ValidatePDF(pdfBytes, article)
	if err != nil {
		t.Fatalf("ValidatePDF: %v", err)
	}

	if !result.OK() {
		t.Errorf("expected OK, got errors: %v", result.Errors)
	}
	if result.PageCount == 0 {
		t.Error("expected PageCount > 0")
	}
	t.Logf("result: %s", result)
}
