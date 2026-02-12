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
	if !strings.Contains(got, "[1] Hello world") {
		t.Error("prompt missing '[1] Hello world'")
	}
	if !strings.Contains(got, "[2] This is a test") {
		t.Error("prompt missing '[2] This is a test'")
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
			response:      "[1] Hallo Welt\n[2] Dies ist ein Test",
			expectedCount: 2,
			want:          []string{"Hallo Welt", "Dies ist ein Test"},
		},
		{
			name:          "multiline paragraph",
			response:      "[1] Dies ist ein langer\nAbsatz der umgebrochen wurde\n[2] Kurz",
			expectedCount: 2,
			want:          []string{"Dies ist ein langer Absatz der umgebrochen wurde", "Kurz"},
		},
		{
			name:          "blank lines between paragraphs",
			response:      "\n[1] Hallo\n\n[2] Welt\n\n",
			expectedCount: 2,
			want:          []string{"Hallo", "Welt"},
		},
		{
			name:          "wrong count returns error",
			response:      "[1] Only one",
			expectedCount: 2,
			wantErr:       true,
		},
		{
			name:          "single paragraph",
			response:      "[1] Einzelner Absatz",
			expectedCount: 1,
			want:          []string{"Einzelner Absatz"},
		},
		{
			name:          "paragraph with internal blank line",
			response:      "[1] Erster Teil\n\nZweiter Teil\n[2] Nächster",
			expectedCount: 2,
			want:          []string{"Erster Teil\nZweiter Teil", "Nächster"},
		},
		{
			name:          "eight paragraphs",
			response:      "[1] Eins\n[2] Zwei\n[3] Drei\n[4] Vier\n[5] Fünf\n[6] Sechs\n[7] Sieben\n[8] Acht",
			expectedCount: 8,
			want:          []string{"Eins", "Zwei", "Drei", "Vier", "Fünf", "Sechs", "Sieben", "Acht"},
		},
		{
			name:          "double digit numbers",
			response:      "[1] A\n[2] B\n[3] C\n[4] D\n[5] E\n[6] F\n[7] G\n[8] H\n[9] I\n[10] J",
			expectedCount: 10,
			want:          []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"},
		},
		{
			name:          "content starting with N. not confused as delimiter",
			response:      "[1] 6. Breadboarding\n[2] Nächster Absatz",
			expectedCount: 2,
			want:          []string{"6. Breadboarding", "Nächster Absatz"},
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
		{"[1] Hello", 1, "Hello", true},
		{"[12] World", 12, "World", true},
		{"[1]Hello", 1, "Hello", true},
		{"[8] Acht", 8, "Acht", true},
		{"Not numbered", 0, "", false},
		{"[] No number", 0, "", false},
		{"", 0, "", false},
		{"abc", 0, "", false},
		{"[0] Zero", 0, "Zero", true},
		{"1. Old format", 0, "", false},
		{"6. Breadboarding", 0, "", false},
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
