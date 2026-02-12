package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	content := `auth_token: "my-token"
ct0: "my-ct0"
ollama_model: "custom:7b"
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc == nil {
		t.Fatal("expected config, got nil")
	}
	if fc.AuthToken != "my-token" {
		t.Errorf("AuthToken = %q, want %q", fc.AuthToken, "my-token")
	}
	if fc.CT0 != "my-ct0" {
		t.Errorf("CT0 = %q, want %q", fc.CT0, "my-ct0")
	}
	if fc.OllamaModel != "custom:7b" {
		t.Errorf("OllamaModel = %q, want %q", fc.OllamaModel, "custom:7b")
	}
}

func TestLoadConfigFile_FileNotExists(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	fc, err := loadConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc != nil {
		t.Fatalf("expected nil, got %+v", fc)
	}
}

func TestLoadConfigFile_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	content := `auth_token: [invalid yaml
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfigFile()
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLoadConfigFile_PartialFields(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	content := `auth_token: "only-token"
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fc, err := loadConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.AuthToken != "only-token" {
		t.Errorf("AuthToken = %q, want %q", fc.AuthToken, "only-token")
	}
	if fc.CT0 != "" {
		t.Errorf("CT0 = %q, want empty", fc.CT0)
	}
	if fc.OllamaModel != "" {
		t.Errorf("OllamaModel = %q, want empty", fc.OllamaModel)
	}
}

// writeConfigFile is a test helper that writes a YAML config to the given dir.
func writeConfigFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFlags_ConfigFileMerge(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "file-token"
ct0: "file-ct0"
`)

	cfg, err := ParseFlags([]string{"https://x.com/i/article/123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthToken != "file-token" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "file-token")
	}
	if cfg.CT0 != "file-ct0" {
		t.Errorf("CT0 = %q, want %q", cfg.CT0, "file-ct0")
	}
}

func TestParseFlags_CLIOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "file-token"
ct0: "file-ct0"
ollama_model: "file-model"
`)

	cfg, err := ParseFlags([]string{
		"--auth-token", "cli-token",
		"--ct0", "cli-ct0",
		"https://x.com/i/article/123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthToken != "cli-token" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "cli-token")
	}
	if cfg.CT0 != "cli-ct0" {
		t.Errorf("CT0 = %q, want %q", cfg.CT0, "cli-ct0")
	}
	// OllamaModel not set on CLI, should come from file.
	if cfg.OllamaModel != "file-model" {
		t.Errorf("OllamaModel = %q, want %q", cfg.OllamaModel, "file-model")
	}
}

func TestParseFlags_MissingAuthShowsHelpMessage(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })
	// No config file written — directory is empty.

	_, err := ParseFlags([]string{"https://x.com/i/article/123"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--auth-token is required") {
		t.Errorf("error missing auth-token message: %s", msg)
	}
	if !strings.Contains(msg, "config file") {
		t.Errorf("error missing config file tip: %s", msg)
	}
}
