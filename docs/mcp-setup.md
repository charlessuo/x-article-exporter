# MCP Server Setup

The x-article-exporter includes an MCP (Model Context Protocol) server that lets Claude Code export X articles as PDFs directly from your conversation.

## Prerequisites

1. **Go 1.25+** installed
2. **Typst** installed (`brew install typst` or see [typst.app](https://typst.app))
3. **Auth credentials** configured in `~/.config/x-article-exporter/config.yaml`
4. **Ollama** (optional, for translation) running locally with `translategemma:12b`

## Quick Start

### 1. Build the binary

```bash
cd /path/to/x-article-exporter
go build -o x-article-exporter .
```

### 2. Configure credentials

Create `~/.config/x-article-exporter/config.yaml`:

```yaml
auth_token: "your-auth-token"
ct0: "your-ct0-cookie"
output_dir: "~/Documents/x-articles"
```

**Getting auth credentials:** Open X in your browser, open DevTools (F12), go to Application > Cookies > `https://x.com`, and copy the values of `auth_token` and `ct0`.

The `output_dir` setting controls where PDFs are saved by default. Use `~` for home directory expansion. Defaults to `.` (current directory) if not set.

### 3. Add to Claude Code

```bash
claude mcp add x-article-exporter /path/to/x-article-exporter -- --mcp
```

This registers the MCP server with Claude Code. The `--mcp` flag tells the binary to start in MCP mode (stdio JSON-RPC) instead of CLI mode.

To verify it's registered:

```bash
claude mcp list
```

## Available Tools

### export_article

Export an X article as a PDF file.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `url` | string | yes | Full X/Twitter article URL |
| `translate` | string | no | Target language code (e.g. `de`, `fr`, `ja`) |
| `dark_mode` | boolean | no | Render with dark background |
| `output` | string | no | Custom output file path |

**Example prompt:** "Export this article as a PDF: https://x.com/user/article/123"

### get_article_info

Fetch article metadata without rendering a PDF. Useful for previewing what an article contains before exporting.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `url` | string | yes | Full X/Twitter article URL |

**Example prompt:** "What's in this article? https://x.com/user/article/123"

### list_exports

List PDF files in the configured output directory.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `limit` | number | no | Max files to return (default: 20) |

**Example prompt:** "Show me my recent article exports"

### check_translation

Check if the Ollama translation service is available and the configured model is ready.

**Example prompt:** "Can I translate articles right now?"

## Configuration Reference

All settings go in `~/.config/x-article-exporter/config.yaml`:

```yaml
# Required: X authentication
auth_token: "your-auth-token"
ct0: "your-ct0-cookie"

# Optional: PDF output
output_dir: "~/Documents/x-articles"  # default: "."
dark_mode: false                       # default: false

# Optional: Translation
ollama_model: "translategemma:12b"     # default: "translategemma:12b"
```

## Troubleshooting

**"MCP mode requires auth_token and ct0 in config file"**
Create `~/.config/x-article-exporter/config.yaml` with your X auth credentials.

**"article has no content blocks"**
The article URL may not point to an X article, or your auth credentials may have expired. Re-copy `auth_token` and `ct0` from your browser.

**"cannot connect to Ollama"**
Start Ollama with `ollama serve` and ensure the model is pulled: `ollama pull translategemma:12b`.

**"query ID" errors**
The X GraphQL query ID rotates periodically. The tool attempts to resolve it automatically, but if that fails, you may need to update it manually from browser DevTools.
