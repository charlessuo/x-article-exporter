package config

import (
	"errors"
	"flag"
	"fmt"
)

// Config holds the CLI configuration for a single export run.
type Config struct {
	URL       string
	AuthToken string
	CT0       string
	QueryID   string
	Output    string
}

// ParseFlags parses CLI arguments into a Config.
// The first positional argument is the article URL.
func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("x-article-exporter", flag.ContinueOnError)

	cfg := &Config{}
	fs.StringVar(&cfg.AuthToken, "auth-token", "", "X auth_token cookie value (required)")
	fs.StringVar(&cfg.CT0, "ct0", "", "X ct0 cookie value (required)")
	fs.StringVar(&cfg.QueryID, "query-id", "", "GraphQL query ID override (optional)")
	fs.StringVar(&cfg.Output, "output", "", "output PDF path (default: ./{title}.pdf)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if fs.NArg() < 1 {
		return nil, errors.New("usage: x-article-exporter --auth-token <token> --ct0 <ct0> [flags] <url>")
	}
	cfg.URL = fs.Arg(0)

	var errs []error
	if cfg.AuthToken == "" {
		errs = append(errs, errors.New("--auth-token is required"))
	}
	if cfg.CT0 == "" {
		errs = append(errs, errors.New("--ct0 is required"))
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%w", errors.Join(errs...))
	}

	return cfg, nil
}
