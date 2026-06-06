package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// Config holds the CLI configuration for a single export run.
type Config struct {
	URL         string
	AuthToken   string
	CT0         string
	QueryID     string
	Output      string
	OutputDir   string
	FileName    string
	TranslateTo string
	OllamaModel string
	DarkMode    bool
}

// OutputBasePath returns the output base path (without extension) for the run.
// The tool writes "<base>.html" and "<base>.pdf".
//
// Precedence:
//  1. --output-dir + --file-name (when both are set) -> join(dir, name)
//  2. --output (legacy) -> the value with a trailing ".pdf" stripped
//  3. "" -> the caller falls back to a sanitized article title
func (c *Config) OutputBasePath() string {
	if c.OutputDir != "" && c.FileName != "" {
		return filepath.Join(c.OutputDir, c.FileName)
	}
	if c.Output != "" {
		return strings.TrimSuffix(c.Output, ".pdf")
	}
	return ""
}

// ParseFlags parses CLI arguments into a Config.
// Values are loaded from the config file first, then CLI flags override.
// The first positional argument is the article URL.
func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("x-article-exporter", flag.ContinueOnError)

	cfg := &Config{}
	fs.StringVar(&cfg.AuthToken, "auth-token", "", "X auth_token cookie value")
	fs.StringVar(&cfg.CT0, "ct0", "", "X ct0 cookie value")
	fs.StringVar(&cfg.QueryID, "query-id", "", "GraphQL query ID override (optional)")
	fs.StringVar(&cfg.Output, "output", "", "output PDF path (default: ./{title}.pdf)")
	fs.StringVar(&cfg.OutputDir, "output-dir", "", "directory to write outputs into (used with --file-name)")
	fs.StringVar(&cfg.FileName, "file-name", "", "base output filename without extension (used with --output-dir)")
	fs.StringVar(&cfg.TranslateTo, "translate", "", "translate article to target language (e.g., de, fr)")
	fs.StringVar(&cfg.OllamaModel, "ollama-model", "", "Ollama model for translation (default: translategemma:12b)")
	fs.BoolVar(&cfg.DarkMode, "dark", false, "render PDF in dark mode")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if fs.NArg() < 1 {
		return nil, errors.New("usage: x-article-exporter [flags] <url>")
	}
	cfg.URL = fs.Arg(0)

	// Track which flags were explicitly set on the command line.
	flagsSet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

	// Load config file and merge values for flags not explicitly set.
	fileCfg, err := loadConfigFile()
	if err != nil {
		return nil, err
	}
	configFileFound := fileCfg != nil
	if fileCfg != nil {
		if !flagsSet["auth-token"] && fileCfg.AuthToken != "" {
			cfg.AuthToken = fileCfg.AuthToken
		}
		if !flagsSet["ct0"] && fileCfg.CT0 != "" {
			cfg.CT0 = fileCfg.CT0
		}
		if !flagsSet["ollama-model"] && fileCfg.OllamaModel != "" {
			cfg.OllamaModel = fileCfg.OllamaModel
		}
		if !flagsSet["dark"] && fileCfg.DarkMode {
			cfg.DarkMode = true
		}
		log.Println("Loaded config file.")
	}

	// Validate required fields (must come from either flags or config file).
	var errs []error
	if cfg.AuthToken == "" {
		errs = append(errs, errors.New("--auth-token is required"))
	}
	if cfg.CT0 == "" {
		errs = append(errs, errors.New("--ct0 is required"))
	}
	if len(errs) > 0 {
		msg := fmt.Sprintf("%s", errors.Join(errs...))
		if !configFileFound {
			path, _ := configFilePath()
			if path != "" {
				msg += fmt.Sprintf("\n\nTip: create a config file to avoid passing auth flags every time:\n\n"+
					"    mkdir -p %s\n"+
					"    cat > %s << 'EOF'\n"+
					"    auth_token: \"your-auth-token\"\n"+
					"    ct0: \"your-ct0\"\n"+
					"    EOF",
					filepath.Dir(path), path)
			}
		}
		return nil, errors.New(msg)
	}

	return cfg, nil
}
