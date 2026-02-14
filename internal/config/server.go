package config

import "fmt"

// ServerConfig holds configuration for the HTTP server mode.
type ServerConfig struct {
	Port                    int
	Host                    string
	JobTTLMinutes           int
	DefaultRateLimitPerHour int
	MaxConcurrentJobs       int
	APIKeys                 []APIKey
	Auth                    AuthConfig
}

// APIKey represents a configured API key with an optional per-key rate limit.
type APIKey struct {
	Key              string
	Name             string
	RateLimitPerHour int // 0 means use DefaultRateLimitPerHour
}

// AuthConfig holds the X API authentication fields needed by the pipeline.
type AuthConfig struct {
	AuthToken   string
	CT0         string
	OllamaModel string
	DarkMode    bool
}

// LoadServerConfig reads the config file and returns server configuration
// with defaults applied. Returns an error if the file exists but is malformed.
func LoadServerConfig() (*ServerConfig, error) {
	fc, err := loadConfigFile()
	if err != nil {
		return nil, err
	}

	sc := &ServerConfig{
		Port:                    8080,
		Host:                    "127.0.0.1",
		JobTTLMinutes:           60,
		DefaultRateLimitPerHour: 20,
		MaxConcurrentJobs:       4,
	}

	if fc == nil {
		return sc, nil
	}

	sc.Auth = AuthConfig{
		AuthToken:   fc.AuthToken,
		CT0:         fc.CT0,
		OllamaModel: fc.OllamaModel,
		DarkMode:    fc.DarkMode,
	}

	if fc.Server == nil {
		return sc, nil
	}

	srv := fc.Server
	if srv.Port != 0 {
		sc.Port = srv.Port
	}
	if srv.Host != "" {
		sc.Host = srv.Host
	}
	if srv.JobTTLMinutes != 0 {
		sc.JobTTLMinutes = srv.JobTTLMinutes
	}
	if srv.DefaultRateLimitPerHour != 0 {
		sc.DefaultRateLimitPerHour = srv.DefaultRateLimitPerHour
	}
	if srv.MaxConcurrentJobs != 0 {
		sc.MaxConcurrentJobs = srv.MaxConcurrentJobs
	}

	for _, k := range srv.APIKeys {
		if k.Key == "" {
			return nil, fmt.Errorf("API key entry missing 'key' field")
		}
		ak := APIKey{
			Key:              k.Key,
			Name:             k.Name,
			RateLimitPerHour: k.RateLimitPerHour,
		}
		sc.APIKeys = append(sc.APIKeys, ak)
	}

	return sc, nil
}
