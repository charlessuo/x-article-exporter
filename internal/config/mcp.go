package config

import (
	"os"
	"path/filepath"
	"strings"
)

// MCPConfig holds configuration for the MCP server mode.
type MCPConfig struct {
	AuthToken   string
	CT0         string
	OllamaModel string
	DarkMode    bool
	OutputDir   string // default save location for exported PDFs
}

// LoadMCPConfig reads the config file and returns MCP configuration.
// OutputDir defaults to "." if not set.
func LoadMCPConfig() (*MCPConfig, error) {
	fc, err := loadConfigFile()
	if err != nil {
		return nil, err
	}

	cfg := &MCPConfig{OutputDir: "."}

	if fc == nil {
		return cfg, nil
	}

	cfg.AuthToken = fc.AuthToken
	cfg.CT0 = fc.CT0
	cfg.OllamaModel = fc.OllamaModel
	cfg.DarkMode = fc.DarkMode
	if fc.OutputDir != "" {
		cfg.OutputDir = expandHome(fc.OutputDir)
	}

	return cfg, nil
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
