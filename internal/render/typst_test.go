package render

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

func TestRenderTypstBasicArticle(t *testing.T) {
	article := &model.Article{
		Title:       "Test Article",
		Author:      "@testauthor",
		PublishedAt: time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "header-two", Text: "Introduction"},
			{Type: "unstyled", Text: "This is a paragraph."},
			{Type: "unstyled", Text: "Another paragraph with bold.", InlineStyleRanges: []model.InlineStyleRange{
				{Offset: 23, Length: 4, Style: "BOLD"},
			}},
			{Type: "unordered-list-item", Text: "Item one"},
			{Type: "unordered-list-item", Text: "Item two"},
			{Type: "code-block", Text: "fmt.Println(\"hello\")"},
			{Type: "code-block", Text: "fmt.Println(\"world\")"},
			{Type: "blockquote", Text: "A wise quote."},
		},
		EntityMap: map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.TempDir != "" {
		defer os.RemoveAll(result.TempDir)
	}
	got := result.Source

	checks := []struct {
		name     string
		contains string
	}{
		{"page setup", `#set page(`},
		{"us-letter", `paper: "us-letter"`},
		{"page numbering", `numbering: "1 / 1"`},
		{"font", `font: ("Open Sans", "Apple Symbols")`},
		{"title", `#text(size: 24pt, weight: "bold")[Test Article]`},
		{"author and date", `\@testauthor · June 15, 2024`},
		{"h2", `== Introduction`},
		{"paragraph", "This is a paragraph."},
		{"bold", "*bold*"},
		{"unordered list", "- Item one"},
		{"code block", "```\nfmt.Println(\"hello\")\nfmt.Println(\"world\")\n```"},
		{"blockquote border", `stroke: (left: 3pt + rgb("#ccc"))`},
		{"blockquote content", "A wise quote."},
		{"horizontal rule", `#line(length: 100%, stroke: 0.5pt + rgb("#ddd"))`},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(got, c.contains) {
				t.Errorf("renderTypst() missing %q\n\ngot:\n%s", c.contains, got)
			}
		})
	}
}

func TestRenderTypstDarkMode(t *testing.T) {
	article := &model.Article{
		Title:       "Dark Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks:      []model.Block{{Type: "unstyled", Text: "Content."}},
		EntityMap:   map[string]model.Entity{},
	}

	result, err := renderTypst(article, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.TempDir != "" {
		defer os.RemoveAll(result.TempDir)
	}
	got := result.Source

	checks := []struct {
		name     string
		contains string
	}{
		{"dark page fill", `fill: rgb("#000000")`},
		{"dark text fill", `fill: rgb("#e7e9ea")`},
		{"dark link color", `#show link: set text(fill: rgb("#1d9bf0"))`},
		{"dark code bg", `fill: rgb("#1a1a1a")`},
		{"dark line color", `stroke: 0.5pt + rgb("#333")`},
		{"dark meta color", `fill: rgb("#8b98a5")`},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(got, c.contains) {
				t.Errorf("renderTypst(dark) missing %q\n\ngot:\n%s", c.contains, got)
			}
		})
	}
}

func TestRenderTypstEmptyUnstyled(t *testing.T) {
	article := &model.Article{
		Title:       "Empty Block",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "unstyled", Text: "Before."},
			{Type: "unstyled", Text: ""},
			{Type: "unstyled", Text: "After."},
		},
		EntityMap: map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	got := result.Source

	if !strings.Contains(got, "#v(0.5em)") {
		t.Error("empty unstyled block should render as #v(0.5em)")
	}
}

func TestRenderTypstHeaders(t *testing.T) {
	article := &model.Article{
		Title:       "Headers",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "header-one", Text: "H1"},
			{Type: "header-two", Text: "H2"},
			{Type: "header-three", Text: "H3"},
			{Type: "header-four", Text: "H4"},
			{Type: "header-five", Text: "H5"},
			{Type: "header-six", Text: "H6"},
		},
		EntityMap: map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	got := result.Source

	checks := map[string]string{
		"h1": "= H1",
		"h2": "== H2",
		"h3": "=== H3",
		"h4": "==== H4",
		"h5": "===== H5",
		"h6": "====== H6",
	}
	for name, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("%s: missing %q", name, want)
		}
	}
}

func TestRenderTypstAtomicImage(t *testing.T) {
	// Use a minimal valid base64 PNG (1x1 pixel).
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	dataURI := "data:image/png;base64," + pngData

	article := &model.Article{
		Title:       "Image Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "unstyled", Text: "Before."},
			{
				Type: "atomic",
				Text: " ",
				EntityRanges: []model.EntityRange{
					{Offset: 0, Length: 1, Key: 0},
				},
			},
			{Type: "unstyled", Text: "After."},
		},
		EntityMap: map[string]model.Entity{
			"0": {Type: "MEDIA", Data: map[string]any{"src": dataURI}},
		},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.TempDir != "" {
		defer os.RemoveAll(result.TempDir)
	}
	got := result.Source

	if !strings.Contains(got, "#figure(") {
		t.Error("missing #figure(")
	}
	if !strings.Contains(got, "image(") {
		t.Error("missing image(")
	}
	if !strings.Contains(got, "width: 60%") {
		t.Error("content images should be 60% width")
	}
}

func TestRenderTypstCoverImage(t *testing.T) {
	pngData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	dataURI := "data:image/png;base64," + pngData

	article := &model.Article{
		Title:         "Cover Test",
		PublishedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CoverImageURL: dataURI,
		Blocks:        []model.Block{{Type: "unstyled", Text: "Content."}},
		EntityMap:     map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.TempDir != "" {
		defer os.RemoveAll(result.TempDir)
	}
	got := result.Source

	if !strings.Contains(got, "width: 100%") {
		t.Error("cover image should be 100% width")
	}
}

func TestRenderTypstBlockquoteGrouping(t *testing.T) {
	article := &model.Article{
		Title:       "Blockquote Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "blockquote", Text: "First quote"},
			{Type: "blockquote", Text: "Second quote"},
		},
		EntityMap: map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	got := result.Source

	// Should have a single #block() with both quote lines.
	if count := strings.Count(got, "#block("); count != 1 {
		t.Errorf("expected 1 #block(, got %d", count)
	}
	if !strings.Contains(got, "First quote") {
		t.Error("missing first quote")
	}
	if !strings.Contains(got, "Second quote") {
		t.Error("missing second quote")
	}
}

func TestRenderTypstEscapesTitle(t *testing.T) {
	article := &model.Article{
		Title:       "Title with #set and @ref",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks:      []model.Block{{Type: "unstyled", Text: "Content."}},
		EntityMap:   map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	got := result.Source

	if strings.Contains(got, "[Title with #set") {
		t.Error("title should escape # character")
	}
	if !strings.Contains(got, "\\#set") {
		t.Error("title should contain escaped \\#set")
	}
}

func TestRenderTypstOrderedList(t *testing.T) {
	article := &model.Article{
		Title:       "OL Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "ordered-list-item", Text: "Step one"},
			{Type: "ordered-list-item", Text: "Step two"},
		},
		EntityMap: map[string]model.Entity{},
	}

	result, err := renderTypst(article, false)
	if err != nil {
		t.Fatal(err)
	}
	got := result.Source

	if !strings.Contains(got, "+ Step one\n+ Step two") {
		t.Errorf("ordered list should use + prefix, got:\n%s", got)
	}
}

func TestDecodeDataURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		ext  string
	}{
		{"jpeg", "data:image/jpeg;base64,/9j/4A==", ".jpg"},
		{"png", "data:image/png;base64,iVBOR", ".png"},
		{"gif", "data:image/gif;base64,R0lG", ".gif"},
		{"webp", "data:image/webp;base64,UklG", ".webp"},
		{"unknown", "data:application/octet-stream;base64,AAAA", ".bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dataURIExtension(tt.uri)
			if got != tt.ext {
				t.Errorf("dataURIExtension() = %q, want %q", got, tt.ext)
			}
		})
	}
}
