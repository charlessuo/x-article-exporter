package config

import (
	"testing"
)

func TestLoadServerConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	sc, err := LoadServerConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Port != 8080 {
		t.Errorf("Port = %d, want 8080", sc.Port)
	}
	if sc.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", sc.Host)
	}
	if sc.JobTTLMinutes != 60 {
		t.Errorf("JobTTLMinutes = %d, want 60", sc.JobTTLMinutes)
	}
	if sc.DefaultRateLimitPerHour != 20 {
		t.Errorf("DefaultRateLimitPerHour = %d, want 20", sc.DefaultRateLimitPerHour)
	}
	if sc.MaxConcurrentJobs != 4 {
		t.Errorf("MaxConcurrentJobs = %d, want 4", sc.MaxConcurrentJobs)
	}
	if len(sc.APIKeys) != 0 {
		t.Errorf("APIKeys = %v, want empty", sc.APIKeys)
	}
}

func TestLoadServerConfig_NoServerSection(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "tok"
ct0: "ct"
`)

	sc, err := LoadServerConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Auth.AuthToken != "tok" {
		t.Errorf("Auth.AuthToken = %q, want tok", sc.Auth.AuthToken)
	}
	if sc.Auth.CT0 != "ct" {
		t.Errorf("Auth.CT0 = %q, want ct", sc.Auth.CT0)
	}
	// Defaults still apply.
	if sc.Port != 8080 {
		t.Errorf("Port = %d, want 8080", sc.Port)
	}
}

func TestLoadServerConfig_FullConfig(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "tok"
ct0: "ct"
ollama_model: "gemma:7b"
dark_mode: true
server:
  port: 9090
  host: "0.0.0.0"
  job_ttl_minutes: 30
  default_rate_limit_per_hour: 100
  max_concurrent_jobs: 8
  api_keys:
    - key: "key-1"
      name: "test"
      rate_limit_per_hour: 50
    - key: "key-2"
      name: "admin"
`)

	sc, err := LoadServerConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.Port != 9090 {
		t.Errorf("Port = %d, want 9090", sc.Port)
	}
	if sc.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", sc.Host)
	}
	if sc.JobTTLMinutes != 30 {
		t.Errorf("JobTTLMinutes = %d, want 30", sc.JobTTLMinutes)
	}
	if sc.DefaultRateLimitPerHour != 100 {
		t.Errorf("DefaultRateLimitPerHour = %d, want 100", sc.DefaultRateLimitPerHour)
	}
	if sc.MaxConcurrentJobs != 8 {
		t.Errorf("MaxConcurrentJobs = %d, want 8", sc.MaxConcurrentJobs)
	}
	if sc.Auth.AuthToken != "tok" {
		t.Errorf("Auth.AuthToken = %q, want tok", sc.Auth.AuthToken)
	}
	if sc.Auth.DarkMode != true {
		t.Error("Auth.DarkMode = false, want true")
	}
	if sc.Auth.OllamaModel != "gemma:7b" {
		t.Errorf("Auth.OllamaModel = %q, want gemma:7b", sc.Auth.OllamaModel)
	}

	if len(sc.APIKeys) != 2 {
		t.Fatalf("APIKeys len = %d, want 2", len(sc.APIKeys))
	}
	if sc.APIKeys[0].Key != "key-1" || sc.APIKeys[0].Name != "test" || sc.APIKeys[0].RateLimitPerHour != 50 {
		t.Errorf("APIKeys[0] = %+v", sc.APIKeys[0])
	}
	if sc.APIKeys[1].Key != "key-2" || sc.APIKeys[1].RateLimitPerHour != 0 {
		t.Errorf("APIKeys[1] = %+v, want key-2 with rate 0 (inherit default)", sc.APIKeys[1])
	}
}

func TestLoadServerConfig_PartialServerSection(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `auth_token: "tok"
ct0: "ct"
server:
  port: 3000
`)

	sc, err := LoadServerConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Port != 3000 {
		t.Errorf("Port = %d, want 3000", sc.Port)
	}
	// Other fields should have defaults.
	if sc.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", sc.Host)
	}
	if sc.MaxConcurrentJobs != 4 {
		t.Errorf("MaxConcurrentJobs = %d, want 4", sc.MaxConcurrentJobs)
	}
}

func TestLoadServerConfig_APIKeyMissingKey(t *testing.T) {
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })

	writeConfigFile(t, dir, `server:
  api_keys:
    - name: "no-key"
`)

	_, err := LoadServerConfig()
	if err == nil {
		t.Fatal("expected error for API key missing 'key' field")
	}
}
