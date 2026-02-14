package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// fileConfig represents the YAML config file structure.
type fileConfig struct {
	AuthToken   string      `yaml:"auth_token"`
	CT0         string      `yaml:"ct0"`
	OllamaModel string     `yaml:"ollama_model"`
	DarkMode    bool        `yaml:"dark_mode"`
	OutputDir   string      `yaml:"output_dir"`
	Server      *serverYAML `yaml:"server"`
}

type serverYAML struct {
	Port                    int          `yaml:"port"`
	Host                    string       `yaml:"host"`
	JobTTLMinutes           int          `yaml:"job_ttl_minutes"`
	DefaultRateLimitPerHour int          `yaml:"default_rate_limit_per_hour"`
	MaxConcurrentJobs       int          `yaml:"max_concurrent_jobs"`
	APIKeys                 []apiKeyYAML `yaml:"api_keys"`
}

type apiKeyYAML struct {
	Key              string `yaml:"key"`
	Name             string `yaml:"name"`
	RateLimitPerHour int    `yaml:"rate_limit_per_hour"`
}

// configDir overrides the config directory in tests. Empty means use default.
var configDir string

func configFilePath() (string, error) {
	dir := configDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config", "x-article-exporter")
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// loadConfigFile reads the YAML config file.
// Returns (nil, nil) if the file does not exist.
func loadConfigFile() (*fileConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return &fc, nil
}
