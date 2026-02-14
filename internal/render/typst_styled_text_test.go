package render

import (
	"testing"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

func TestTypstEscape(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain text", "plain text"},
		{"*bold*", "\\*bold\\*"},
		{"_italic_", "\\_italic\\_"},
		{"`code`", "\\`code\\`"},
		{"#heading", "\\#heading"},
		{"@mention", "\\@mention"},
		{"$math$", "\\$math\\$"},
		{"<tag>", "\\<tag\\>"},
		{"[link]", "\\[link\\]"},
		{"back\\slash", "back\\\\slash"},
		{"no escapes here", "no escapes here"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := typstEscape(tt.in)
			if got != tt.want {
				t.Errorf("typstEscape(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRenderTypstStyledText(t *testing.T) {
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
			want:      "Hello *world*",
		},
		{
			name: "single italic range",
			block: model.Block{
				Text: "Hello world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 5, Style: "ITALIC"},
				},
			},
			entityMap: nil,
			want:      "Hello _world_",
		},
		{
			name: "bold and italic on same range",
			block: model.Block{
				Text: "Hello world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 5, Style: "BOLD"},
					{Offset: 6, Length: 5, Style: "ITALIC"},
				},
			},
			entityMap: nil,
			want:      "Hello *_world_*",
		},
		{
			name: "overlapping bold and italic",
			block: model.Block{
				Text: "Hello bold and italic world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 17, Style: "BOLD"},
					{Offset: 15, Length: 12, Style: "ITALIC"},
				},
			},
			entityMap: nil,
			want:      "Hello *bold and **_italic w_*_orld_",
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
			want: `Click #link("https://example.com")[here] for more`,
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
			want: `See #link("https://example.com")[*this link*] now`,
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
			want:      "Use the `fmt.Println` function",
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
			want:      "This is #strike[removed] text here",
		},
		{
			name:      "typst special characters escaped",
			block:     model.Block{Text: "Use #set and @ref with $math"},
			entityMap: nil,
			want:      "Use \\#set and \\@ref with \\$math",
		},
		{
			name: "unicode text with styles (rune offsets)",
			block: model.Block{
				Text: "Ägypten ist schön",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 0, Length: 7, Style: "BOLD"},
				},
			},
			entityMap: nil,
			want:      "*Ägypten* ist schön",
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
			want:      "*bold* then _italic_",
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
			name: "title-case style names from API",
			block: model.Block{
				Text: "Hello bold italic world",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 6, Length: 4, Style: "Bold"},
					{Offset: 11, Length: 6, Style: "Italic"},
				},
			},
			entityMap: nil,
			want:      "Hello *bold* _italic_ world",
		},
		{
			name: "underline style",
			block: model.Block{
				Text: "This is underlined text",
				InlineStyleRanges: []model.InlineStyleRange{
					{Offset: 8, Length: 10, Style: "UNDERLINE"},
				},
			},
			entityMap: nil,
			want:      "This is #underline[underlined] text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTypstStyledText(tt.block, tt.entityMap)
			if got != tt.want {
				t.Errorf("renderTypstStyledText() =\n  %q\nwant:\n  %q", got, tt.want)
			}
		})
	}
}
