---
title: Installation
weight: 1
---

## Prerequisites

- [Go 1.25+](https://go.dev/dl/) — required for building from source or `go install`
- [Typst](https://typst.app) — PDF rendering engine

### Installing Typst

{{< tabs items="macOS,Linux,Windows" >}}
  {{< tab >}}
  ```sh
  brew install typst
  ```
  {{< /tab >}}
  {{< tab >}}
  ```sh
  # Arch Linux
  pacman -S typst

  # Or download from GitHub releases
  curl -fsSL https://github.com/typst/typst/releases/latest/download/typst-x86_64-unknown-linux-musl.tar.xz | tar xJ
  ```
  {{< /tab >}}
  {{< tab >}}
  ```sh
  winget install --id Typst.Typst
  ```
  {{< /tab >}}
{{< /tabs >}}

### Optional: Translation support

To translate articles before export, you need:

- [Ollama](https://ollama.com) — local LLM runtime
- `translategemma:12b` model — pull with `ollama pull translategemma:12b`

## Install with go install

```sh
go install github.com/annismckenzie/x-article-exporter@latest
```

This places the binary in your `$GOPATH/bin` (or `$HOME/go/bin` by default).

## Build from source

```sh
git clone https://github.com/annismckenzie/x-article-exporter.git
cd x-article-exporter
go build -o x-article-exporter .
```

## Authentication setup

X Article Exporter requires browser auth cookies to access the X API.

{{< steps >}}

### Open X in your browser

Go to [x.com](https://x.com) and make sure you're logged in.

### Open DevTools

Press F12 (or Cmd+Option+I on macOS) to open browser developer tools.

### Copy cookies

Navigate to **Application** > **Cookies** > `https://x.com` and copy the values of:

- `auth_token`
- `ct0`

### Create config file

Save to `~/.config/x-article-exporter/config.yaml`:

```yaml
auth_token: "your-auth-token"
ct0: "your-ct0"
```

{{< /steps >}}

## Verify installation

```sh
x-article-exporter https://x.com/user/article/1234567890
```

This creates a PDF in the current directory named after the article title.
