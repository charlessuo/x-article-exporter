package config

import (
	"os"
	"path/filepath"
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
