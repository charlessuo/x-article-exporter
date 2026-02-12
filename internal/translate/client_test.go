package translate

import (
	"encoding/json"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("", "")
	if c.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %q, want default", c.BaseURL)
	}
	if c.Model != "translategemma:12b" {
		t.Errorf("Model = %q, want default", c.Model)
	}
}

func TestNewClientCustom(t *testing.T) {
	c := NewClient("http://myhost:1234", "llama3:8b")
	if c.BaseURL != "http://myhost:1234" {
		t.Errorf("BaseURL = %q, want custom", c.BaseURL)
	}
	if c.Model != "llama3:8b" {
		t.Errorf("Model = %q, want custom", c.Model)
	}
}

func TestChatResponseUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantContent string
		wantError  string
	}{
		{
			name:        "successful response",
			input:       `{"message":{"role":"assistant","content":"1. Hallo Welt"},"error":""}`,
			wantContent: "1. Hallo Welt",
		},
		{
			name:      "error response",
			input:     `{"message":{"role":"","content":""},"error":"model not found"}`,
			wantError: "model not found",
		},
		{
			name:        "empty content",
			input:       `{"message":{"role":"assistant","content":""}}`,
			wantContent: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp chatResponse
			if err := json.Unmarshal([]byte(tt.input), &resp); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if tt.wantError != "" && resp.Error != tt.wantError {
				t.Errorf("Error = %q, want %q", resp.Error, tt.wantError)
			}
			if tt.wantContent != "" && resp.Message.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", resp.Message.Content, tt.wantContent)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is longer than ten", 10, "this is lo..."},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
