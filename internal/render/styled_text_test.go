package render

import (
	"testing"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

func TestRenderStyledText(t *testing.T) {
	tests := []struct {
		name      string
		block     model.Block
		entityMap map[string]model.Entity
		want      string
	}{
		{
			name:      "plain text no styles",
			block:     model.Block{Text: "Hello world"},
			entityMap: nil,
			want:      "Hello world",
		},
		{
			name: "single bold range",
			block: model.Block{
				Text: "Hello world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 5, Style: "BOLD"},
				},
			},
			entityMap: nil,
			want:      "Hello <strong>world</strong>",
		},
		{
			name: "overlapping bold and italic",
			block: model.Block{
				Text: "Hello bold and italic world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 17, Style: "BOLD"},   // "bold and italic w"
					{Offset: 15, Length: 12, Style: "ITALIC"}, // "italic world"
				},
			},
			entityMap: nil,
			want:      "Hello <strong>bold and </strong><strong><em>italic w</em></strong><em>orld</em>",
		},
		{
			name: "link entity",
			block: model.Block{
				Text: "Click here for more",
				EntityRanges: []model.EntityRange{
					{Offset: 6, Length: 4, Key: 0},
				},
			},
			entityMap: map[string]model.Entity{
				"0": {Type: "LINK", Data: map[string]any{"url": "https://example.com"}},
			},
			want: `Click <a href="https://example.com">here</a> for more`,
		},
		{
			name: "bold text inside a link",
			block: model.Block{
				Text: "See this link now",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 4, Length: 9, Style: "BOLD"},
				},
				EntityRanges: []model.EntityRange{
					{Offset: 4, Length: 9, Key: 0},
				},
			},
			entityMap: map[string]model.Entity{
				"0": {Type: "LINK", Data: map[string]any{"url": "https://example.com"}},
			},
			want: `See <a href="https://example.com"><strong>this link</strong></a> now`,
		},
		{
			name: "inline code style",
			block: model.Block{
				Text: "Use the fmt.Println function",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 8, Length: 11, Style: "CODE"},
				},
			},
			entityMap: nil,
			want:      "Use the <code>fmt.Println</code> function",
		},
		{
			name: "strikethrough",
			block: model.Block{
				Text: "This is removed text here",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 8, Length: 7, Style: "STRIKETHROUGH"},
				},
			},
			entityMap: nil,
			want:      "This is <del>removed</del> text here",
		},
		{
			name:      "html special characters escaped",
			block:     model.Block{Text: "Use <script> & \"quotes\""},
			entityMap: nil,
			want:      "Use &lt;script&gt; &amp; &#34;quotes&#34;",
		},
		{
			name: "unicode text with styles (rune offsets)",
			block: model.Block{
				Text: "Ägypten ist schön",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 0, Length: 7, Style: "BOLD"}, // "Ägypten" is 7 runes
				},
			},
			entityMap: nil,
			want:      "<strong>Ägypten</strong> ist schön",
		},
		{
			name:      "empty text",
			block:     model.Block{Text: ""},
			entityMap: nil,
			want:      "",
		},
		{
			name: "adjacent non-overlapping ranges",
			block: model.Block{
				Text: "bold then italic",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 0, Length: 4, Style: "BOLD"},
					{Offset: 10, Length: 6, Style: "ITALIC"},
				},
			},
			entityMap: nil,
			want:      "<strong>bold</strong> then <em>italic</em>",
		},
		{
			name: "non-link entity passes through as plain text",
			block: model.Block{
				Text: "Hello emoji",
				EntityRanges: []model.EntityRange{
					{Offset: 6, Length: 5, Key: 0},
				},
			},
			entityMap: map[string]model.Entity{
				"0": {Type: "TWEMOJI", Data: map[string]any{}},
			},
			want: "Hello emoji",
		},
		{
			name: "link with special characters in URL",
			block: model.Block{
				Text: "Click here",
				EntityRanges: []model.EntityRange{
					{Offset: 0, Length: 10, Key: 0},
				},
			},
			entityMap: map[string]model.Entity{
				"0": {Type: "LINK", Data: map[string]any{"url": "https://example.com/path?q=a&b=c"}},
			},
			want: `<a href="https://example.com/path?q=a&amp;b=c">Click here</a>`,
		},
		{
			name: "title-case style names from API (Bold, Italic)",
			block: model.Block{
				Text: "Hello bold italic world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 4, Style: "Bold"},
					{Offset: 11, Length: 6, Style: "Italic"},
				},
			},
			entityMap: nil,
			want:      "Hello <strong>bold</strong> <em>italic</em> world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderStyledText(tt.block, tt.entityMap)
			if got != tt.want {
				t.Errorf("renderStyledText() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}
