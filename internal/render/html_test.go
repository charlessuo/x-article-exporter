package render

import (
	"strings"
	"testing"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

func TestGroupBlocks(t *testing.T) {
	tests := []struct {
		name   string
		blocks []model.Block
		want   []renderGroup
	}{
		{
			name:   "empty blocks",
			blocks: nil,
			want:   nil,
		},
		{
			name: "single paragraph",
			blocks: []model.Block{
				{Type: "unstyled", Text: "Hello"},
			},
			want: []renderGroup{
				{Blocks: []model.Block{{Type: "unstyled", Text: "Hello"}}},
			},
		},
		{
			name: "consecutive unordered list items grouped",
			blocks: []model.Block{
				{Type: "unordered-list-item", Text: "Item 1"},
				{Type: "unordered-list-item", Text: "Item 2"},
				{Type: "unordered-list-item", Text: "Item 3"},
			},
			want: []renderGroup{
				{
					Tag: "ul",
					Blocks: []model.Block{
						{Type: "unordered-list-item", Text: "Item 1"},
						{Type: "unordered-list-item", Text: "Item 2"},
						{Type: "unordered-list-item", Text: "Item 3"},
					},
				},
			},
		},
		{
			name: "consecutive ordered list items grouped",
			blocks: []model.Block{
				{Type: "ordered-list-item", Text: "Step 1"},
				{Type: "ordered-list-item", Text: "Step 2"},
			},
			want: []renderGroup{
				{
					Tag: "ol",
					Blocks: []model.Block{
						{Type: "ordered-list-item", Text: "Step 1"},
						{Type: "ordered-list-item", Text: "Step 2"},
					},
				},
			},
		},
		{
			name: "consecutive code blocks grouped",
			blocks: []model.Block{
				{Type: "code-block", Text: "func main() {"},
				{Type: "code-block", Text: "  fmt.Println(\"hello\")"},
				{Type: "code-block", Text: "}"},
			},
			want: []renderGroup{
				{
					Tag: "pre",
					Blocks: []model.Block{
						{Type: "code-block", Text: "func main() {"},
						{Type: "code-block", Text: "  fmt.Println(\"hello\")"},
						{Type: "code-block", Text: "}"},
					},
				},
			},
		},
		{
			name: "mixed blocks with list in the middle",
			blocks: []model.Block{
				{Type: "unstyled", Text: "Intro"},
				{Type: "unordered-list-item", Text: "A"},
				{Type: "unordered-list-item", Text: "B"},
				{Type: "unstyled", Text: "Outro"},
			},
			want: []renderGroup{
				{Blocks: []model.Block{{Type: "unstyled", Text: "Intro"}}},
				{
					Tag: "ul",
					Blocks: []model.Block{
						{Type: "unordered-list-item", Text: "A"},
						{Type: "unordered-list-item", Text: "B"},
					},
				},
				{Blocks: []model.Block{{Type: "unstyled", Text: "Outro"}}},
			},
		},
		{
			name: "different list types not merged",
			blocks: []model.Block{
				{Type: "unordered-list-item", Text: "Bullet"},
				{Type: "ordered-list-item", Text: "Number"},
			},
			want: []renderGroup{
				{Tag: "ul", Blocks: []model.Block{{Type: "unordered-list-item", Text: "Bullet"}}},
				{Tag: "ol", Blocks: []model.Block{{Type: "ordered-list-item", Text: "Number"}}},
			},
		},
		{
			name: "consecutive blockquotes grouped",
			blocks: []model.Block{
				{Type: "blockquote", Text: "First quote"},
				{Type: "blockquote", Text: "Second quote"},
			},
			want: []renderGroup{
				{
					Tag: "blockquote",
					Blocks: []model.Block{
						{Type: "blockquote", Text: "First quote"},
						{Type: "blockquote", Text: "Second quote"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupBlocks(tt.blocks)
			if len(got) != len(tt.want) {
				t.Fatalf("groupBlocks() returned %d groups, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Tag != tt.want[i].Tag {
					t.Errorf("group[%d].Tag = %q, want %q", i, got[i].Tag, tt.want[i].Tag)
				}
				if len(got[i].Blocks) != len(tt.want[i].Blocks) {
					t.Errorf("group[%d] has %d blocks, want %d", i, len(got[i].Blocks), len(tt.want[i].Blocks))
				}
			}
		})
	}
}

func TestRenderHTML(t *testing.T) {
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

	got := RenderHTML(article)

	// Check document structure.
	checks := []struct {
		name     string
		contains string
	}{
		{"doctype", "<!DOCTYPE html>"},
		{"title tag", "<title>Test Article</title>"},
		{"article title", `<h1 class="article-title">Test Article</h1>`},
		{"author", `<span class="author">@testauthor</span>`},
		{"date", `<span class="date">June 15, 2024</span>`},
		{"h2", "<h2>Introduction</h2>"},
		{"paragraph", "<p>This is a paragraph.</p>"},
		{"bold text", "<strong>bold</strong>"},
		{"unordered list open", "<ul>"},
		{"list item", "<li>Item one</li>"},
		{"unordered list close", "</ul>"},
		{"pre/code block", "<pre><code>"},
		{"code content joined", "fmt.Println(&#34;hello&#34;)\nfmt.Println(&#34;world&#34;)"},
		{"blockquote", "<blockquote>"},
		{"blockquote content", "<p>A wise quote.</p>"},
		{"css styles", "font-family: Georgia"},
	}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(got, c.contains) {
				t.Errorf("RenderHTML() missing %q\n\ngot:\n%s", c.contains, got)
			}
		})
	}
}

func TestRenderHTMLAtomic(t *testing.T) {
	article := &model.Article{
		Title:       "Image Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "unstyled", Text: "Before image."},
			{
				Type: "atomic",
				Text: " ",
				EntityRanges: []model.EntityRange{
					{Offset: 0, Length: 1, Key: 0},
				},
			},
			{Type: "unstyled", Text: "After image."},
		},
		EntityMap: map[string]model.Entity{
			"0": {Type: "MEDIA", Data: map[string]any{"src": "data:image/jpeg;base64,abc123"}},
		},
	}

	got := RenderHTML(article)

	if !strings.Contains(got, `<figure>`) {
		t.Error("missing <figure> tag")
	}
	if !strings.Contains(got, `<img src="data:image/jpeg;base64,abc123"`) {
		t.Error("missing image with data URL")
	}
}

func TestRenderHTMLCoverImage(t *testing.T) {
	article := &model.Article{
		Title:         "Cover Test",
		PublishedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		CoverImageURL: "data:image/png;base64,coverdata",
		Blocks:        []model.Block{{Type: "unstyled", Text: "Content."}},
		EntityMap:     map[string]model.Entity{},
	}

	got := RenderHTML(article)

	if !strings.Contains(got, `class="cover-image"`) {
		t.Error("missing cover-image class")
	}
	if !strings.Contains(got, `src="data:image/png;base64,coverdata"`) {
		t.Error("missing cover image src")
	}
}

func TestRenderHTMLEmptyUnstyled(t *testing.T) {
	article := &model.Article{
		Title:       "Empty Block Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "unstyled", Text: "Before."},
			{Type: "unstyled", Text: ""},
			{Type: "unstyled", Text: "After."},
		},
		EntityMap: map[string]model.Entity{},
	}

	got := RenderHTML(article)

	if !strings.Contains(got, "<br>") {
		t.Error("empty unstyled block should render as <br>")
	}
}

func TestRenderHTMLNewlinesInBlock(t *testing.T) {
	article := &model.Article{
		Title:       "Newline Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "unstyled", Text: "First paragraph.\n\nSecond paragraph."},
		},
		EntityMap: map[string]model.Entity{},
	}

	got := RenderHTML(article)

	if !strings.Contains(got, "First paragraph.<br>") {
		t.Error("newlines within block should be converted to <br>")
	}
}

func TestRenderHTMLBlockquoteGrouping(t *testing.T) {
	article := &model.Article{
		Title:       "Blockquote Test",
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks: []model.Block{
			{Type: "blockquote", Text: "First quote"},
			{Type: "blockquote", Text: "Second quote"},
		},
		EntityMap: map[string]model.Entity{},
	}

	got := RenderHTML(article)

	// Should have a single <blockquote> with two <p> elements.
	if count := strings.Count(got, "<blockquote>"); count != 1 {
		t.Errorf("expected 1 <blockquote>, got %d", count)
	}
	if !strings.Contains(got, "<p>First quote</p>") {
		t.Error("missing first quote paragraph")
	}
	if !strings.Contains(got, "<p>Second quote</p>") {
		t.Error("missing second quote paragraph")
	}
}

func TestRenderHTMLEscapesTitle(t *testing.T) {
	article := &model.Article{
		Title:       `Title with <script> & "quotes"`,
		PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Blocks:      []model.Block{{Type: "unstyled", Text: "Content."}},
		EntityMap:   map[string]model.Entity{},
	}

	got := RenderHTML(article)

	if strings.Contains(got, "<script>") {
		t.Error("title should be HTML-escaped, found raw <script>")
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Error("title should contain escaped &lt;script&gt;")
	}
}
