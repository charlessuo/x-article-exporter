# X Article Exporter — Shaping

## Source

> I want to have the ability to create a PDF from an X article. Currently, printing an X article in the browser results in unreadable PDFs (wrong formatting etc.), not sure if that's intended. Anyways, nowadays you need an X account to read tweets and articles and I want to be able to share a PDF of an article I read with friends and colleagues. Further, it should be possible to translate the article before export because not all of my friends understand English well enough. Think hard about how to judge the quality of your solution without me having to constantly check the PDFs manually.

---

## Frame

### Problem

- Browser print of X articles produces unreadable PDFs (broken formatting, missing content)
- X requires an account to read content — can't just share a link with people who don't have one
- Some recipients don't read English well enough to consume articles in the original language

### Outcome

- Can produce a well-formatted, readable PDF from any X article
- Can share articles with people who don't have X accounts
- Can translate articles before export so non-English readers can consume them
- Can trust the output quality without manually inspecting every PDF

---

## Requirements (R)

| ID  | Requirement                                                                                   | Status    |
| --- | --------------------------------------------------------------------------------------------- | --------- |
| R0  | Produce a readable, well-formatted PDF from an X article URL                                  | Core goal |
| R1  | Extract full article content from X (text, images, author, date)                              | Must-have |
| R2  | Translate article text to a target language before PDF generation                             | Must-have |
| R3  | PDF is shareable with people who have no X account (self-contained)                           | Must-have |
| R4  | Automated quality validation — can judge PDF correctness without manual inspection            | Must-have |
| R5  | Handle X authentication via config file with CLI flag override (`auth_token` + `ct0` cookies) | Must-have |
| R6  | Translation is opt-in per export via explicit `--translate <lang>` CLI flag                   | Must-have |
| R7  | Export tweet threads as PDF with the same quality and shareability as articles                | Must-have |
| R8  | Thread structure is visually clear in the PDF (each tweet identifiable, chronological order)  | Must-have |
| R9  | HTTP API interface — POST a URL, receive a PDF                                                | Must-have |
| R10 | API authentication via API keys (not X cookies — those are server-side)                       | Must-have |
| R11 | Translation and thread export work through the API, not just CLI                              | Must-have |

---

## A: CLI tool — fetch, translate, render to PDF

**A1: Content extraction** — `GET x.com/i/api/graphql/{queryId}/TweetResultByRestId` with cookie auth (`auth_token` + `ct0`), hardcoded bearer token, feature flags, and `fieldToggles.withArticleRichContentState: true`. Articles are fetched as tweets — the article URL's snowflake ID is a tweet ID. Response is Draft.js `RawDraftContentState` in `content_state` field (blocks with types, inline styles, entity map as array of `{key, value}` pairs). Query IDs: dynamic extraction from webpack `api` chunk with 24h cache + manual `--query-id` fallback.

**A2: Translation** — Send all translatable blocks as a single XML-tagged document to DeepL API (`/v2/translate`) with `tag_handling: "xml"`. Use `ignore_tags` for code blocks and LaTeX. Thin Go HTTP client (~100 LOC). Free tier: 500K chars/month ≈ 16 articles. Pro: ~$0.75/article.

**A3: PDF rendering** — Generate Typst source from article model, compile to PDF via `typst compile`. Page setup (us-letter, margins, numbering), dark mode support (`--dark` flag / `dark_mode` config), embedded Open Sans fonts (static TTFs via `go:embed`), Apple Symbols fallback for special glyphs. Images decoded from base64 data URIs to temp files. ~500 LOC total (Typst renderer + styled text + PDF compilation). Requires `typst` binary installed.

**A4: Quality pipeline** — Two-tier validation. Per-export (<200ms): `pdfcpu.ValidateFile()`, page count, image count match, title/author present, word count ±15%. CI: golden metadata snapshots, must-contain phrases, empty page detection. Libraries: `pdfcpu` + `ledongthuc/pdf` (both pure Go).

### Spike results for A1

See [spike-a1-content-extraction.md](./spike-a1-content-extraction.md).

**Key findings:**

- X articles are 100% client-side rendered — no content in HTML
- Internal GraphQL API is the only path to article content
- `TweetResultByRestId` endpoint fetches article content via the parent tweet's snowflake ID
- Cookie-based auth (`auth_token` + `ct0`) is the only reliable auth method
- Article content is block-based rich text (headings, paragraphs, lists, code, quotes, LaTeX, media, embedded tweets)
- Query IDs rotate every 2-4 weeks — must be extracted from JS bundles
- No existing tool in any language handles article extraction
- GraphQL preferred over headless browser: faster, structured data, better for translation + validation

**A1 deep dive completed** — see [spike-a1-deep-dive.md](./spike-a1-deep-dive.md) for resolution of three unknowns:

- **Response format**: Operation is `TweetResultByRestId` — articles are fetched as tweets. Response path: `data.tweetResult.result.article.article_results.result`. Body content is Draft.js `RawDraftContentState` in `content_state` field, unlocked via `fieldToggles.withArticleRichContentState: true`. Entity map is an array of `{key, value}` pairs. Author info from tweet wrapper.
- **Query ID extraction**: IDs are in webpack `api` chunk. All major scrapers hardcode them. Regex extraction from JS bundle is feasible. Recommended: dynamic extraction with 24h cache + manual fallback.
- **Request format**: GET request with URL-encoded `variables`, `features`, `fieldToggles` params. Bearer token, headers, cookie format, and feature flags are fully documented.

**A1 status**: All unknowns resolved by V1 implementation. Content field is `content_state`, entity types include `MEDIA` (not `IMAGE`), `LINK`, `TWEMOJI`, `MARKDOWN`. Entity map is an array format. Author info from tweet wrapper. Query IDs rotate every 2-4 weeks (currently `d6YKjvQ920F-D4Y1PruO-A`).

### Spike results for A2

See [spike-a2-translation.md](./spike-a2-translation.md).

**Key findings:**

- DeepL is best-in-class for European languages and has native `ignore_tags` for code/LaTeX
- Free tier (500K chars/month) covers ~16 articles/month; Pro is ~$0.75/article
- No official Go SDK but REST API is trivial (~100 LOC custom client)
- Full-document translation (single request with XML tags) beats paragraph-by-paragraph
- **Round-trip translation is NOT a reliable quality signal** (EAMT 2020) — dropped from A4
- Instead: validate block structure preservation, code/LaTeX byte-identity, length ratio ±30%

**A2 is resolved** — mechanism is concrete and well-understood.

### Spike results for A3

See [spike-a3-pdf-rendering.md](./spike-a3-pdf-rendering.md).

**Key findings:**

- wkhtmltopdf is dead (archived 2024, deprecated QtWebKit)
- Go PDF libraries (gofpdf/fpdf/maroto) require ~1000+ LOC for rich article layout — too complex
- chromedp + HTML template was the initial choice (V2) but hit hard limitations: Chrome margin areas are always white (killing dark mode), CSS body padding doesn't repeat across pages, page numbers live in the always-white margin area
- WeasyPrint and Typst both solve all Chrome limitations — see [spike-a-weasyprint.md](./spike-a-weasyprint.md), [spike-b-typst.md](./spike-b-typst.md), [spike-comparison.md](./spike-comparison.md)
- **Typst wins:** no Docker dependency, superior typography (hyphenation, smart page breaks), faster rendering, smaller PDFs, single static binary

**A3 is resolved** — Typst renderer implemented and verified.

### Spike results for A4

See [spike-a4-quality-pipeline.md](./spike-a4-quality-pipeline.md).

**Key findings:**

- Two-tier validation: per-export (<200ms) + CI (golden fixtures)
- Per-export: `pdfcpu.ValidateFile()`, page count, image count match, title/author/date present, word count ±15%
- CI: golden metadata snapshots (JSON), must-contain phrases, empty page detection, translation fixtures
- Libraries: `pdfcpu` (Apache 2.0) + `ledongthuc/pdf` (BSD-3) — both pure Go, no CGo
- Visual regression is overkill — structural checks catch 95% of real problems
- Translation checks: block count match, code block byte-identity, length ratio ±30%

**A4 is resolved** — mechanism is concrete and well-understood.

### A5: Thread extraction + rendering

See [spike-a5-thread-extraction.md](./spike-a5-thread-extraction.md).

**A5a:** Detect URL type — `/article/` → article pipeline, `/status/` → check if thread or article tweet.

**A5b:** Fetch thread — walk self-reply chain via conversation endpoint, collect all tweets by same author in chronological order. ⚠️

**A5c:** Parse thread tweets into unified block model — tweet text → unstyled blocks, tweet media → atomic blocks, tweet metadata → headers. ⚠️

**A5d:** Thread-specific HTML template — tweet cards with avatar, timestamp, text, media. Distinct from article prose layout. ⚠️

**A5 status:** ⚠️ Spike needed. A5b–d have flagged unknowns (thread API, tweet content model, PDF layout). Once tweets are parsed into blocks, the rest of the pipeline (translation, rendering, validation) works unchanged.

### A6: HTTP API server

**A6a:** HTTP server (`net/http`) with routes: `POST /export`, `GET /export/{id}`, `GET /export/{id}/pdf`.

**A6b:** API key auth middleware — keys stored in config YAML, checked via `Authorization: Bearer <key>` header.

**A6c:** Job manager — accepts export request, runs pipeline in goroutine, stores result. In-memory map with TTL (1h). Returns job ID immediately.

**A6d:** `POST /export` accepts `{"url": "...", "translate": "de"}`, creates job, returns `{"id": "...", "status": "processing"}`.

**A6e:** `GET /export/{id}` returns job status: `processing`, `complete`, `failed`. On complete, includes metadata (title, author, page count).

**A6f:** `GET /export/{id}/pdf` returns PDF bytes (`Content-Type: application/pdf`) when complete. 404 if not found, 202 if still processing.

**A6g:** Rate limiting per API key — token bucket or fixed window, configurable in config YAML.

**A6 is resolved** — implemented in V6. Pipeline extracted to reusable `internal/pipeline.Run()`. Go 1.22+ `ServeMux` patterns (`POST /export`, `GET /export/{id}`, `GET /export/{id}/pdf`). Bounded concurrency via semaphore channel, context propagation for graceful cancellation, token-bucket rate limiting per API key, Bearer token auth middleware. Graceful shutdown with `signal.Notify` + `http.Server.Shutdown`.

### A8: MCP server for Claude Code

**A8a:** MCP stdio server (`--mcp` flag) using `mcp-go` SDK — same binary, stdio JSON-RPC protocol. 4 tools: `export_article`, `get_article_info`, `list_exports`, `check_translation`.

**A8b:** Config-driven auth — reads `auth_token`, `ct0`, `output_dir`, `dark_mode`, `ollama_model` from config file. No API keys needed (Claude Code is the sole client via stdio).

**A8c:** `export_article` tool — validates URL, runs `pipeline.Run()`, writes PDF to configured output dir (or custom path), returns metadata. Supports per-call `translate`, `dark_mode`, `output` overrides.

**A8d:** `get_article_info` tool — fetches and parses article metadata without rendering a PDF. Fast preview before committing to export.

**A8e:** `list_exports` + `check_translation` tools — list PDFs in output dir, verify Ollama availability.

**A8 is resolved** — implemented in V8. Same binary with `--mcp` flag starts stdio MCP server. Reuses `pipeline.Run()` from V6 extraction. Config loaded via `LoadMCPConfig()` with `output_dir` tilde expansion. Registered via `claude mcp add`.

---

## Fit Check (R × A)

| Req | Requirement                                                                                   | Status    | A (current, V1–V6, V8) | +A5 (threads) |
| --- | --------------------------------------------------------------------------------------------- | --------- | :--------------------: | :-----------: |
| R0  | Produce a readable, well-formatted PDF from an X article URL                                  | Core goal |         ✅         |      ✅       |
| R1  | Extract full article content from X (text, images, author, date)                              | Must-have |         ✅         |      ✅       |
| R2  | Translate article text to a target language before PDF generation                             | Must-have |         ✅         |      ✅       |
| R3  | PDF is shareable with people who have no X account (self-contained)                           | Must-have |         ✅         |      ✅       |
| R4  | Automated quality validation — can judge PDF correctness without manual inspection            | Must-have |         ✅         |      ✅       |
| R5  | Handle X authentication via config file with CLI flag override (`auth_token` + `ct0` cookies) | Must-have |         ✅         |      ✅       |
| R6  | Translation is opt-in per export via explicit `--translate <lang>` CLI flag                   | Must-have |         ✅         |      ✅       |
| R7  | Export tweet threads as PDF with the same quality and shareability as articles                | Must-have |         ❌         |      ❌       |
| R8  | Thread structure is visually clear in the PDF (each tweet identifiable, chronological order)  | Must-have |         ❌         |      ❌       |
| R9  | HTTP API interface — POST a URL, receive a PDF                                                | Must-have |         ✅         |      ✅       |
| R10 | API authentication via API keys (not X cookies — those are server-side)                       | Must-have |         ✅         |      ✅       |
| R11 | Translation and thread export work through the API, not just CLI                              | Must-have |         ❌         |      ❌       |

**Notes:**

- R0–R6: Satisfied by V1–V5. See notes below for mechanism details.
- R7, R8 fail: A5b–d are flagged unknowns (⚠️). Can't claim ✅ until spike resolves them.
- R9, R10: Satisfied by V6 (HTTP API server with API key auth).
- R11 fails: translation works through the API (`"translate": "de"` in POST body), but thread export (A5) isn't implemented yet. Passes once A5 is done.
- V8 (MCP server) doesn't add new requirements — it's a new interface for the same pipeline, like V6. All R0–R6 tools work through MCP.
- R0 ✅: A1 (GraphQL extraction → Draft.js blocks) + A3 (Typst article→PDF) form the complete pipeline
- R1 ✅: `TweetResultByRestId` with `withArticleRichContentState: true` returns full content as Draft.js blocks in `content_state` field (title, body, images, entities)
- R2 ✅: Local Ollama with translategemma:12b, batch [N] delimiters, plain text translation
- R3 ✅: Typst `compile` produces self-contained PDF with embedded images and fonts
- R4 ✅: Two-tier validation pipeline with `pdfcpu` + `ledongthuc/pdf`, <200ms per-export overhead
- R5 ✅: Config file (`~/.config/x-article-exporter/config.yaml`) for `auth_token` + `ct0`, CLI flags as override. Bearer token is hardcoded (same for all users).
- R6 ✅: `--translate <lang>` flag maps directly to Ollama's target language parameter
- R9 ✅: `POST /export` accepts URL + options, returns job ID; `GET /export/{id}/pdf` returns completed PDF
- R10 ✅: Bearer token API key auth in middleware, keys configured in YAML; X cookies are server-side only

---

## Breadboard

See [breadboard.md](./breadboard.md).

**Summary:** 6 UI affordances (CLI args, progress, warnings, errors, success, HTTP API), 22 code affordances (13 pipeline stages + 6 API/job management + 3 external boundaries), 4 data stores (config, query ID cache, article model, job storage). CLI mode: load config → extract → [translate] → render → validate → write. Server mode: HTTP request → job manager → pipeline.Run() → serve PDF.

---

## Slices

See [slices.md](./slices.md).

| #   | Slice              | Mechanism      | Demo                                           |
| --- | ------------------ | -------------- | ---------------------------------------------- |
| V1  | Extract article    | A1             | "Run command, see article summary in terminal" |
| V2  | Render PDF         | A3             | "Run command, get a well-formatted PDF"        |
| V3  | Translation        | A2             | "Run with --translate de, get German PDF"      |
| V4  | Quality validation | A4             | "Run command, see validation pass/warnings"    |
| V5  | Config + query ID  | R5, A1 partial | "Config file works, query ID auto-resolves"    |
| V6  | Web API            | A6             | "POST URL to API, get PDF back"                |
| V7  | Thread export      | A5             | "Pass thread URL, get thread PDF"              |
| V8  | MCP server         | A8             | "`claude mcp add`, export articles from Claude"|
| V9  | README + LICENSE   | —              | "Visit repo, instantly understand + use"       |
