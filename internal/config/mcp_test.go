package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	cfg, err := LoadMCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OutputDir != "." {
		t.Errorf("OutputDir = %q, want \".\"", cfg.OutputDir)
	}
}

func TestLoadMCPConfig_WithOutputDir(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "tok"
ct0: "ct"
output_dir: "/tmp/exports"
`)

	cfg, err := LoadMCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AuthToken != "tok" {
		t.Errorf("AuthToken = %q, want tok", cfg.AuthToken)
	}
	if cfg.CT0 != "ct" {
		t.Errorf("CT0 = %q, want ct", cfg.CT0)
	}
	if cfg.OutputDir != "/tmp/exports" {
		t.Errorf("OutputDir = %q, want /tmp/exports", cfg.OutputDir)
	}
}

func TestLoadMCPConfig_TildeExpansion(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `output_dir: "~/Documents/x-articles"
`)

	cfg, err := LoadMCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "Documents/x-articles")
	if cfg.OutputDir != want {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, want)
	}
}

func TestLoadMCPConfig_AllFields(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "tok"
ct0: "ct"
ollama_model: "gemma:7b"
dark_mode: true
output_dir: "/exports"
`)

	cfg, err := LoadMCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.OllamaModel != "gemma:7b" {
		t.Errorf("OllamaModel = %q, want gemma:7b", cfg.OllamaModel)
	}
	if !cfg.DarkMode {
		t.Error("DarkMode = false, want true")
	}
}
