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

---

## A: CLI tool — fetch, translate, render to PDF

**A1: Content extraction** — `GET x.com/i/api/graphql/{queryId}/TweetResultByRestId` with cookie auth (`auth_token` + `ct0`), hardcoded bearer token, feature flags, and `fieldToggles.withArticleRichContentState: true`. Articles are fetched as tweets — the article URL's snowflake ID is a tweet ID. Response is Draft.js `RawDraftContentState` in `content_state` field (blocks with types, inline styles, entity map as array of `{key, value}` pairs). Query IDs: dynamic extraction from webpack `api` chunk with 24h cache + manual `--query-id` fallback.

**A2: Translation** — Send all translatable blocks as a single XML-tagged document to DeepL API (`/v2/translate`) with `tag_handling: "xml"`. Use `ignore_tags` for code blocks and LaTeX. Thin Go HTTP client (~100 LOC). Free tier: 500K chars/month ≈ 16 articles. Pro: ~$0.75/article.

**A3: PDF rendering** — Render blocks to HTML via Go template + CSS, use `chromedp` (headless Chrome) to `PrintToPDF`. Full CSS print media control (page breaks, headers/footers with page numbers). Images base64-encoded inline. ~250 LOC total (template + CSS + render). Requires Chrome/Chromium installed.

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
- pandoc + typst is good quality but two external dependencies
- weasyprint has heavy Python/C dependency chain, Go port is alpha
- **chromedp + HTML template wins:** HTML/CSS is the natural representation for article content, Chrome renders it perfectly, CSS handles all styling/page breaks/headers, ~250 LOC total
- Chrome dependency is acceptable for a personal CLI tool (almost certainly installed on macOS)

**A3 is resolved** — mechanism is concrete and well-understood.

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

---

## Fit Check (R × A)

| Req | Requirement                                                                                   | Status    | A   |
| --- | --------------------------------------------------------------------------------------------- | --------- | --- |
| R0  | Produce a readable, well-formatted PDF from an X article URL                                  | Core goal | ✅  |
| R1  | Extract full article content from X (text, images, author, date)                              | Must-have | ✅  |
| R2  | Translate article text to a target language before PDF generation                             | Must-have | ✅  |
| R3  | PDF is shareable with people who have no X account (self-contained)                           | Must-have | ✅  |
| R4  | Automated quality validation — can judge PDF correctness without manual inspection            | Must-have | ✅  |
| R5  | Handle X authentication via config file with CLI flag override (`auth_token` + `ct0` cookies) | Must-have | ✅  |
| R6  | Translation is opt-in per export via explicit `--translate <lang>` CLI flag                   | Must-have | ✅  |

**Notes:**

- R0 ✅: A1 (GraphQL extraction → Draft.js blocks) + A3 (chromedp HTML→PDF) form the complete pipeline
- R1 ✅: `TweetResultByRestId` with `withArticleRichContentState: true` returns full content as Draft.js blocks in `content_state` field (title, body, images, entities)
- R2 ✅: DeepL API with XML tag handling, `ignore_tags` for code/LaTeX, full-document translation
- R3 ✅: chromedp `PrintToPDF` produces self-contained PDF with base64-embedded images
- R4 ✅: Two-tier validation pipeline with `pdfcpu` + `ledongthuc/pdf`, <200ms per-export overhead
- R5 ✅: Config file (`~/.config/x-article-exporter/config.yaml`) for `auth_token` + `ct0`, CLI flags as override. Bearer token is hardcoded (same for all users).
- R6 ✅: `--translate <lang>` flag maps directly to DeepL's `target_lang` parameter

**Remaining risk (low):** Query IDs rotate every 2-4 weeks and must be updated. All request/response details are verified by V1 implementation.

---

## Breadboard

See [breadboard.md](./breadboard.md).

**Summary:** 5 UI affordances (CLI args, progress, warnings, errors, success), 16 code affordances (13 internal pipeline stages + 3 external API boundaries), 3 data stores (config, query ID cache, article model). Linear pipeline: load config → extract article → [translate] → render PDF → validate → write.

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
