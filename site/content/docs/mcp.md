---
title: MCP Server
weight: 4
---

The MCP (Model Context Protocol) server lets [Claude Code](https://docs.anthropic.com/en/docs/claude-code) export X articles as PDFs directly from your conversation.

## Prerequisites

- X Article Exporter [installed](installation)
- Auth credentials [configured](configuration)
- Ollama (optional, for translation) running locally with `translategemma:12b`

## Setup

{{< steps >}}

### Build the binary

```sh
go build -o x-article-exporter .
```

### Add to Claude Code

```sh
claude mcp add x-article-exporter /path/to/x-article-exporter -- --mcp
```

The `--mcp` flag starts the binary in MCP mode (stdio JSON-RPC) instead of CLI mode.

### Verify

```sh
claude mcp list
```

{{< /steps >}}

## Available tools

### export_article

Export an X article as a PDF file.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | Full X/Twitter article URL |
| `translate` | string | no | Target language code (e.g. `de`, `fr`, `ja`) |
| `dark_mode` | boolean | no | Render with dark background |
| `output` | string | no | Custom output file path |

**Example prompt:** "Export this article as a PDF: https://x.com/user/article/123"

### get_article_info

Fetch article metadata without rendering a PDF. Useful for previewing what an article contains before exporting.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | Full X/Twitter article URL |

**Example prompt:** "What's in this article? https://x.com/user/article/123"

### list_exports

List PDF files in the configured output directory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `limit` | number | no | Max files to return (default: 20) |

**Example prompt:** "Show me my recent article exports"

### check_translation

Check if the Ollama translation service is available and the configured model is ready.

**Example prompt:** "Can I translate articles right now?"

## MCP configuration

The MCP server reads from the same config file as the CLI (`~/.config/x-article-exporter/config.yaml`):

```yaml
auth_token: "your-auth-token"
ct0: "your-ct0-cookie"
output_dir: "~/Documents/x-articles"  # default: "."
dark_mode: false                       # default: false
ollama_model: "translategemma:12b"     # default: "translategemma:12b"
```

The `output_dir` setting controls where PDFs are saved. Use `~` for home directory expansion.

## Troubleshooting

{{< callout type="warning" >}}
**"MCP mode requires auth_token and ct0 in config file"**
Create `~/.config/x-article-exporter/config.yaml` with your X auth credentials.
{{< /callout >}}

{{< callout type="warning" >}}
**"article has no content blocks"**
The article URL may not point to an X article, or your auth credentials may have expired. Re-copy `auth_token` and `ct0` from your browser.
{{< /callout >}}

{{< callout type="warning" >}}
**"cannot connect to Ollama"**
Start Ollama with `ollama serve` and ensure the model is pulled: `ollama pull translategemma:12b`.
{{< /callout >}}

{{< callout type="warning" >}}
**"query ID" errors**
The X GraphQL query ID rotates periodically. The tool resolves it automatically, but if that fails, update it manually from browser DevTools.
{{< /callout >}}
