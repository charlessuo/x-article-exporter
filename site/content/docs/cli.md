---
title: CLI Usage
weight: 2
---

## Synopsis

```
x-article-exporter [flags] <article-url>
```

Flags must come **before** the positional URL argument (Go `flag` package requirement).

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-auth-token` | string | config file | X `auth_token` cookie |
| `-ct0` | string | config file | X `ct0` cookie |
| `-translate` | string | *(none)* | Translate to target language code (e.g. `de`, `fr`, `ja`) |
| `-dark` | bool | `false` | Render PDF in dark mode |
| `-output` | string | `./{title}.pdf` | Output PDF file path |
| `-query-id` | string | *(auto)* | GraphQL query ID override |
| `-ollama-model` | string | `translategemma:12b` | Ollama model for translation |

CLI flags override values from the [config file](configuration).

## Examples

### Basic export

```sh
x-article-exporter https://x.com/elonmusk/article/1234567890
```

Creates `{article-title}.pdf` in the current directory.

### Translate to German

```sh
x-article-exporter -translate de https://x.com/elonmusk/article/1234567890
```

Requires [Ollama](https://ollama.com) running locally with the `translategemma:12b` model pulled.

### Dark mode

```sh
x-article-exporter -dark https://x.com/elonmusk/article/1234567890
```

Renders with a black background matching X's dark theme.

### Custom output path

```sh
x-article-exporter -output ~/Documents/article.pdf https://x.com/elonmusk/article/1234567890
```

### Combine flags

```sh
x-article-exporter -dark -translate fr -output ~/exports/article.pdf https://x.com/user/article/123
```

### Override auth cookies

```sh
x-article-exporter -auth-token "abc123" -ct0 "xyz789" https://x.com/user/article/123
```

Useful for one-off exports without modifying your config file.

## Supported languages

Translation uses Ollama with `translategemma:12b`, which supports 55+ languages. Common language codes:

| Code | Language | Code | Language |
|------|----------|------|----------|
| `de` | German | `ja` | Japanese |
| `fr` | French | `ko` | Korean |
| `es` | Spanish | `zh` | Chinese |
| `it` | Italian | `ar` | Arabic |
| `pt` | Portuguese | `hi` | Hindi |
| `nl` | Dutch | `ru` | Russian |

See the [translategemma documentation](https://ollama.com/library/translategemma) for the full list.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (invalid URL, auth failure, rendering error, etc.) |
