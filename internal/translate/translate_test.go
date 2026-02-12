package translate

import (
	"strings"
	"testing"
)

func TestBuildSinglePrompt(t *testing.T) {
	got := buildSinglePrompt("Hello world", "German")
	if !strings.Contains(got, "German") {
		t.Error("prompt missing target language")
	}
	if !strings.Contains(got, "Hello world") {
		t.Error("prompt missing source text")
	}
}

func TestBuildBatchPrompt(t *testing.T) {
	got := buildBatchPrompt([]string{"Hello world", "This is a test"}, "German")
	if !strings.Contains(got, "German") {
		t.Error("prompt missing target language")
	}
	if !strings.Contains(got, "1. Hello world") {
		t.Error("prompt missing '1. Hello world'")
	}
	if !strings.Contains(got, "2. This is a test") {
		t.Error("prompt missing '2. This is a test'")
	}
}

func TestParseBatchResponse(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectedCount int
		want          []string
		wantErr       bool
	}{
		{
			name:          "simple numbered response",
			response:      "1. Hallo Welt\n2. Dies ist ein Test",
			expectedCount: 2,
			want:          []string{"Hallo Welt", "Dies ist ein Test"},
		},
		{
			name:          "multiline paragraph",
			response:      "1. Dies ist ein langer\nAbsatz der umgebrochen wurde\n2. Kurz",
			expectedCount: 2,
			want:          []string{"Dies ist ein langer Absatz der umgebrochen wurde", "Kurz"},
		},
		{
			name:          "blank lines between paragraphs",
			response:      "\n1. Hallo\n\n2. Welt\n\n",
			expectedCount: 2,
			want:          []string{"Hallo", "Welt"},
		},
		{
			name:          "wrong count returns error",
			response:      "1. Only one",
			expectedCount: 2,
			wantErr:       true,
		},
		{
			name:          "single paragraph",
			response:      "1. Einzelner Absatz",
			expectedCount: 1,
			want:          []string{"Einzelner Absatz"},
		},
		{
			name:          "paragraph with internal blank line",
			response:      "1. Erster Teil\n\nZweiter Teil\n2. Nächster",
			expectedCount: 2,
			want:          []string{"Erster Teil\nZweiter Teil", "Nächster"},
		},
		{
			name:          "eight paragraphs",
			response:      "1. Eins\n2. Zwei\n3. Drei\n4. Vier\n5. Fünf\n6. Sechs\n7. Sieben\n8. Acht",
			expectedCount: 8,
			want:          []string{"Eins", "Zwei", "Drei", "Vier", "Fünf", "Sechs", "Sieben", "Acht"},
		},
		{
			name:          "double digit numbers",
			response:      "1. A\n2. B\n3. C\n4. D\n5. E\n6. F\n7. G\n8. H\n9. I\n10. J",
			expectedCount: 10,
			want:          []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBatchResponse(tt.response, tt.expectedCount)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseNumberedLine(t *testing.T) {
	tests := []struct {
		line      string
		wantNum   int
		wantRest  string
		wantMatch bool
	}{
		{"1. Hello", 1, "Hello", true},
		{"12. World", 12, "World", true},
		{"1.Hello", 1, "Hello", true},
		{"8. Acht", 8, "Acht", true},
		{"Not numbered", 0, "", false},
		{". No number", 0, "", false},
		{"", 0, "", false},
		{"abc", 0, "", false},
		{"0. Zero", 0, "Zero", true},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			num, rest, ok := parseNumberedLine(tt.line)
			if ok != tt.wantMatch || num != tt.wantNum || rest != tt.wantRest {
				t.Errorf("parseNumberedLine(%q) = (%d, %q, %v), want (%d, %q, %v)",
					tt.line, num, rest, ok, tt.wantNum, tt.wantRest, tt.wantMatch)
			}
		})
	}
}

func TestTranslatableBlockTypes(t *testing.T) {
	shouldTranslate := []string{
		"unstyled", "header-one", "header-two", "header-three",
		"header-four", "header-five", "header-six",
		"unordered-list-item", "ordered-list-item", "blockquote",
	}
	for _, typ := range shouldTranslate {
		if !translatableBlockTypes[typ] {
			t.Errorf("block type %q should be translatable", typ)
		}
	}

	shouldSkip := []string{"code-block", "atomic"}
	for _, typ := range shouldSkip {
		if translatableBlockTypes[typ] {
			t.Errorf("block type %q should NOT be translatable", typ)
		}
	}
}
