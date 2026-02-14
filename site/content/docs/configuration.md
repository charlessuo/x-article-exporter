---
title: Configuration
weight: 5
---

All settings go in `~/.config/x-article-exporter/config.yaml`. CLI flags override config values.

## Full reference

```yaml
# Required: X authentication cookies (get from browser dev tools)
auth_token: "your-auth-token"
ct0: "your-ct0"

# Optional: Ollama model for translation (default: translategemma:12b)
ollama_model: "translategemma:12b"

# Optional: Render PDFs in dark mode (default: false)
dark_mode: false

# Optional: Default output directory for MCP server (default: ".")
# Supports ~ for home directory expansion
output_dir: "~/Documents/x-articles"

# Server mode configuration (used with --serve)
server:
  port: 8080
  host: "127.0.0.1"
  job_ttl_minutes: 60
  default_rate_limit_per_hour: 20
  max_concurrent_jobs: 4
  api_keys:
    - key: "your-api-key-here"
      name: "default"
      # rate_limit_per_hour: 20  # omit to use default_rate_limit_per_hour
```

## Authentication

| Field | Required | Description |
|-------|----------|-------------|
| `auth_token` | yes | X `auth_token` cookie from browser |
| `ct0` | yes | X `ct0` cookie from browser |

These are your X session cookies. See the [installation guide](installation) for how to obtain them.

{{< callout type="info" >}}
Auth cookies expire periodically. If you get authentication errors, re-copy fresh values from your browser.
{{< /callout >}}

## General options

| Field | Default | Description |
|-------|---------|-------------|
| `ollama_model` | `translategemma:12b` | Ollama model used for translation |
| `dark_mode` | `false` | Render PDFs with dark background |
| `output_dir` | `.` | Default PDF output directory (MCP server) |

## Server options

These settings are under the `server:` key and only apply when running with `--serve`.

| Field | Default | Description |
|-------|---------|-------------|
| `port` | `8080` | HTTP server port |
| `host` | `127.0.0.1` | Bind address |
| `job_ttl_minutes` | `60` | How long completed jobs are kept |
| `default_rate_limit_per_hour` | `20` | Default rate limit for API keys |
| `max_concurrent_jobs` | `4` | Maximum parallel export jobs |
| `api_keys` | *(none)* | List of API key objects |

### API key object

| Field | Required | Description |
|-------|----------|-------------|
| `key` | yes | The bearer token string |
| `name` | yes | Human-readable label |
| `rate_limit_per_hour` | no | Per-key override (omit to use default) |

## CLI flag precedence

CLI flags always override config file values:

```sh
# Config says dark_mode: false, but this export uses dark mode
x-article-exporter -dark https://x.com/user/article/123
```

| Config field | CLI flag |
|-------------|----------|
| `auth_token` | `-auth-token` |
| `ct0` | `-ct0` |
| `dark_mode` | `-dark` |
| `ollama_model` | `-ollama-model` |
| *(none)* | `-translate` |
| *(none)* | `-output` |
| *(none)* | `-query-id` |

## Minimal config

The smallest useful config file:

```yaml
auth_token: "your-auth-token"
ct0: "your-ct0"
```
