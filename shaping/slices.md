# Shape A — Slices

## Slice Summary

| #   | Slice              | Mechanism                             | Demo                                                                 |
| --- | ------------------ | ------------------------------------- | -------------------------------------------------------------------- |
| V1  | Extract article    | A1 (content extraction)               | "Run command with URL + auth flags, see article summary in terminal" |
| V2  | Render PDF         | A3 (PDF rendering)                    | "Run command, get a well-formatted PDF"                              |
| V2b | Typst renderer     | A3 (chromedp → Typst switch)          | "Same PDF, now rendered via Typst — dark mode, better typography"    |
| V3  | Translation        | A2 (translation)                      | "Run with `--translate de`, get German PDF"                          |
| V4  | Quality validation | A4 (quality validation)               | "Run command, see validation pass/warnings before output"            |
| V5  | Config + query ID  | R5, A1 partial (query ID resolution)  | "Config file replaces flags, query ID auto-resolves"                 |
| V6  | Web API            | A6 (HTTP API server)                  | "POST URL to API, get PDF back"                                      |
| V7  | Thread export      | A5 (thread extraction + rendering)    | "Pass thread URL, get thread PDF"                                    |
| V8  | MCP server         | A8 (MCP stdio server for Claude Code) | "`claude mcp add`, ask Claude to export article → PDF on disk"       |
| V9  | README + LICENSE   | —                                     | "Visit repo, instantly understand what this is + how to use"         |
| V11 | GitHub Pages site  | Hugo + Hextra                         | "Browse docs at annismckenzie.github.io/x-article-exporter/"         |

---

## V1: Extract Article

**Demo:** `x-article-exporter https://x.com/i/article/123456 --auth-token abc --ct0 xyz` → prints article title, author, date, block count, image count to terminal.

**Notes:**

- Query ID is hardcoded (will rotate in 2-4 weeks — acceptable for first slice)
- `--auth-token` and `--ct0` are required flags (no config file yet)
- No PDF output — terminal summary only
- No image download (not needed for summary)

**New affordances:**

| #   | Place | Component | Affordance                                                      | Control | Wires Out | Returns To |
| --- | ----- | --------- | --------------------------------------------------------------- | ------- | --------- | ---------- |
| U1  | P1    | —         | CLI args: `<url> --auth-token <t> --ct0 <c>`                    | invoke  | → N1      | —          |
| U2  | P1    | —         | Progress log                                                    | render  | —         | —          |
| U4  | P1    | —         | Error messages (exit 1)                                         | render  | —         | —          |
| U5  | P1    | —         | Article summary (exit 0)                                        | render  | —         | —          |
| N1  | P2    | config    | `loadConfig(flags)` — parse CLI flags only, no config file      | call    | —         | → S1       |
| N2  | P2    | extract   | `extractArticleID(url)`                                         | call    | —         | → N5       |
| N5  | P2    | extract   | `fetchArticle(articleID, queryID, config)` — hardcoded query ID | call    | → N14     | → N6       |
| N6  | P2    | extract   | `parseArticle(response)`                                        | call    | —         | → S3       |
| N14 | P3    | —         | `GET TweetResultByRestId`                                       | call    | —         | → N5       |
| S1  | P2    | —         | `config` (from flags only)                                      | store   | —         | —          |
| S3  | P2    | —         | `article` (metadata + blocks)                                   | store   | —         | → U5       |

---

## V2: Render PDF

**Demo:** `x-article-exporter https://x.com/i/article/123456 --auth-token abc --ct0 xyz --output ./article.pdf` → writes a well-formatted, self-contained PDF to disk.

**Notes:**

- Images downloaded and base64-embedded (MEDIA entities resolved via `media_entities[]`)
- Typst source renderer generates `.typ` from article model, compiled via `typst compile`
- Block grouping: consecutive list items → `- `/`+ `, code blocks → ` ``` `, blockquotes → `#block(stroke: (left: ...))[]`
- Boundary-based styled text renderer for Typst markup (`*bold*`, `_italic_`, `` `code` ``, `#strike[]`, `#link()[]`)
- Newlines within styled segments handled by closing/reopening markup at line boundaries
- Embedded Open Sans static TTFs (Regular/Bold/Italic/BoldItalic) + Apple Symbols fallback
- Dark mode via `--dark` flag: `#set page(fill: rgb("#000"))` + `#set text(fill: rgb("#e7e9ea"))`
- Page numbers "1 / 1" centered at bottom, dark mode margins filled with page color
- Self-contained HTML always saved alongside PDF (diffable, shareable via Slack)
- Output file path via `--output` flag (default: `./{title}.pdf` + `.html`)

**New affordances:**

| #   | Place | Component | Affordance                                                               | Control | Wires Out | Returns To |
| --- | ----- | --------- | ------------------------------------------------------------------------ | ------- | --------- | ---------- |
| N7  | P2    | extract   | `downloadImages(blocks)` — fetch + base64-encode                         | call    | —         | updates S3 |
| N10 | P2    | render    | `renderHTML(article, blocks)` — Go template + CSS                        | call    | —         | → N11      |
| N11 | P2    | render    | `printToPDF(article, darkMode)` — generate Typst source, `typst compile` | call    | → N16     | → N13      |
| N13 | P6    | output    | `writePDF(pdfBytes, outputPath)`                                         | call    | writes P6 | → U5       |
| N16 | P5    | —         | `typst compile` — Typst binary renders .typ to PDF                       | call    | —         | → N11      |

---

## V2b: Typst Renderer Switch

**Demo:** `make preview` / `make preview-dark` → PDF output now rendered via Typst instead of chromedp. Dark mode works fully (margins, page numbers, backgrounds). Better typography (hyphenation, smart page breaks).

**Notes:**

- Chrome/chromedp had hard limitations: margin areas always white (killing dark mode), CSS body padding doesn't repeat across pages, page numbers in always-white margin area
- Dual spikes confirmed both WeasyPrint and Typst solve all Chrome limitations (see `shaping/spike-a-weasyprint.md`, `shaping/spike-b-typst.md`, `shaping/spike-comparison.md`)
- **Typst chosen:** no Docker dependency, superior typography, faster rendering, smaller PDFs, single static binary
- Typst source renderer (`internal/render/typst.go`): Article → `.typ` source → `typst compile` → PDF
- Styled text ported to Typst markup: `*bold*`, `_italic_`, `` `code` ``, `#strike[]`, `#link()[]`
- Inline markup can't span newlines — close/reopen at `\n` boundaries
- Images: base64 data URIs decoded to temp dir, referenced by file path in `.typ` source
- Embedded static Open Sans TTFs (Regular/Bold/Italic/BoldItalic) via `go:embed` — variable fonts produce warnings
- Apple Symbols font fallback for ❯ glyph
- Blockquotes use `#block(stroke: (left: 3pt + rgb(...)))` for Chrome-matching left-border look
- `--dark` flag: `#set page(fill: rgb("#000"))` + `#set text(fill: rgb("#e7e9ea"))`
- chromedp dependency removed (`go mod tidy`)
- Implementation plan: see `x-article-exporter-plan-4.md` (project root, adjacent to repo)

**New/changed affordances:**

| #   | Place | Component | Affordance                                                                                              | Control | Wires Out | Returns To |
| --- | ----- | --------- | ------------------------------------------------------------------------------------------------------- | ------- | --------- | ---------- |
| N11 | P2    | render    | `PrintToPDF(ctx, article, darkMode)` — **replaced**: Typst source + `typst compile` instead of chromedp | call    | → N16     | → N13      |
| N16 | P5    | —         | `typst compile` — Typst binary renders .typ to PDF                                                      | call    | —         | → N11      |

**Changed wiring:** N11 no longer calls chromedp. Instead generates Typst source, writes images to temp dir, invokes `typst compile`. Dark mode handled in Typst source (page fill, text color, link/code colors).

---

## V3: Translation

**Demo:** `x-article-exporter https://x.com/i/article/123456 --auth-token abc --ct0 xyz --translate de` → PDF in German with code blocks and images preserved unchanged.

**Notes:**

- `--translate <lang>` and `--ollama-model <model>` flags added
- Local Ollama with translategemma:12b (purpose-built translation model, 55 languages, no API key needed)
- Batch translation: 8 blocks per request using `[N]` delimiters to avoid collision with numbered content
- Plain text translation: InlineStyleRanges/EntityRanges cleared on translated blocks (render pipeline handles this gracefully)
- Translatable block types: unstyled, headers, list items, blockquotes. Code blocks and atomic (images) skipped.
- Title translated separately; trailing period stripped if original didn't have one
- ~6 min for a full 70-block article on M3/32GB

**Nice-to-have ideas:**

- **Image text translation**: Images with text (screenshots, diagrams with labels) are not translated. Investigate OCR + overlay or image regeneration approaches. Spike needed to assess feasibility and quality.

**New affordances:**

| #   | Place | Component | Affordance                                                                 | Control | Wires Out | Returns To |
| --- | ----- | --------- | -------------------------------------------------------------------------- | ------- | --------- | ---------- |
| N8  | P2    | translate | `TranslateArticle(article, targetLang, client)` — batch translate in-place | call    | → N15     | updates S3 |
| N15 | local | —         | `POST /api/chat` — Ollama (translategemma:12b)                             | call    | —         | → N8       |

---

## V4: Quality Validation

**Demo:** `x-article-exporter ...` → after PDF render, validation output: "PDF valid. 4 pages. 3 images (expected 3). Title found. Author found. Word count: 1847 (expected ~1800). OK."

**Notes:**

- Runs automatically after every PDF render (<200ms overhead)
- `pdfcpu`: structural integrity, page count, image count
- `ledongthuc/pdf`: text extraction for title/author present, word count ±15%
- U3 extended with PDF validation warnings (word count off, image count mismatch)
- Hard failures (missing title, PDF corrupt) → U4 (exit 1)
- Soft warnings (word count slightly off) → U3 (exit 0)

**Nice-to-have ideas (from V2 E2E feedback):**

- **Per-article settings JSON**: A small JSON file alongside the article that overrides rendering settings (e.g., image max-width, page break locations, font size tweaks) for per-article taste adjustments. Each article has unique "sharp edges" that only manual tweaks can fix.
- **Golden file testing**: Use the HTML output (pre-PDF) as golden files for regression/integration tests. The HTML is clean and deterministic, making it ideal for diffing.

**New affordances:**

| #   | Place | Component | Affordance                                                         | Control | Wires Out | Returns To        |
| --- | ----- | --------- | ------------------------------------------------------------------ | ------- | --------- | ----------------- |
| N12 | P2    | validate  | `validatePDF(pdfBytes, article, blocks)` — pdfcpu + ledongthuc/pdf | call    | —         | → U3, → U4, → N13 |

**Changed wiring:** N11 now wires to N12 instead of directly to N13. N12 gates output: pass → N13, hard fail → U4.

---

## V5: Config File + Query ID Resolution

**Demo:** Create `~/.config/x-article-exporter/config.yaml` with `auth_token` and `ct0`. Run `x-article-exporter https://x.com/i/article/123456` without auth flags — works. Query ID auto-resolves from X's JS bundles, cached for 24h.

**Notes:**

- YAML config file at `~/.config/x-article-exporter/config.yaml` (gopkg.in/yaml.v3)
- Fields: `auth_token`, `ct0`, `ollama_model` — only persistent settings, per-invocation args stay CLI-only
- CLI flags override config file values (detected via `flag.FlagSet.Visit`)
- When auth missing and no config file: error includes tip with `mkdir -p` + `cat >` example
- Query ID resolution chain: cache (24h TTL) → JS bundle extraction → `--query-id` flag → hardcoded fallback
- Bundle extraction: fetch `x.com` HTML → find `main.*.js` URL → regex for `queryId:"...",operationName:"TweetResultByRestId"`
- Cache at `~/.cache/x-article-exporter/query-id.json` (JSON, machine-managed)
- Resolver never fatally errors — always falls through to a fallback

**New affordances:**

| #   | Place | Component | Affordance                                                              | Control | Wires Out   | Returns To |
| --- | ----- | --------- | ----------------------------------------------------------------------- | ------- | ----------- | ---------- |
| N1  | P2    | config    | `ParseFlags()` — **extended**: reads config.yaml + merges with flags    | call    | reads P6    | → S1       |
| N3  | P2    | extract   | `ResolveQueryID(ctx, flagOverride)` — cache → bundle → flag → hardcoded | call    | → N4 (miss) | → N5       |
| N4  | P3    | extract   | `fetchQueryIDFromBundle()` — GET x.com → find main.\*.js → regex        | call    | —           | → S2, → N3 |
| S2  | P6    | —         | `queryIDCache` — 24h TTL, `~/.cache/x-article-exporter/query-id.json`   | store   | —           | → N3       |

**Changed wiring:** N2 (extractArticleID) now wires to N3 (resolveQueryID) instead of directly to N5. N3 wires to N5 with the resolved query ID.

---

## V6: Web API

**Demo:** `make serve` starts the server. Then: `curl -X POST -H "Authorization: Bearer <key>" -d '{"url":"https://x.com/i/article/123"}' http://localhost:8080/export` → 202 with job ID. Poll `GET /export/{id}` for status. Download `GET /export/{id}/pdf` when complete. Ctrl-C → graceful shutdown.

**Notes:**

- Pipeline extracted to reusable `internal/pipeline.Run()` — both CLI and server call the same code path
- `--serve` flag pre-scanned before `config.ParseFlags` (server mode has no positional URL argument)
- Go 1.22+ `ServeMux` patterns: `"POST /export"`, `"GET /export/{id}"`, `"GET /export/{id}/pdf"` (no manual string parsing)
- Bounded concurrency via `chan struct{}` semaphore (configurable `max_concurrent_jobs`, default 4)
- Context propagation: `context.WithCancel` from manager to pipeline goroutines for cancellation
- `Storage.Get()` returns copies to prevent data races between readers and writers
- Graceful shutdown: `signal.Notify(SIGINT, SIGTERM)` + `http.Server.Shutdown` + `sync.WaitGroup` for in-flight jobs
- TTL cleanup uses `UpdatedAt` (not `CreatedAt`), so long-running jobs survive their full TTL after completion
- Bearer token auth middleware validates `Authorization: Bearer <key>` against config
- Token-bucket rate limiting per API key on POST /export only (the expensive endpoint)
- `http.MaxBytesReader` limits request body to 1MB
- HTTP server timeouts: read 30s, write 60s, idle 120s
- URL validated via `extract.ExtractArticleID` before accepting job (400 on bad URL)
- Config: `server:` YAML section with `api_keys`, `port`, `host`, `default_rate_limit_per_hour`, `max_concurrent_jobs`, `job_ttl_minutes`

**New affordances:**

| #   | Place | Component | Affordance                                                                     | Control | Wires Out       | Returns To |
| --- | ----- | --------- | ------------------------------------------------------------------------------ | ------- | --------------- | ---------- |
| U6  | P7    | —         | HTTP API: POST /export, GET /export/{id}, GET /export/{id}/pdf                 | invoke  | → N17           | —          |
| N17 | P7    | api       | `Server.Handler()` — mux with auth + rate limit middleware                     | call    | → N18, N19, N20 | —          |
| N18 | P7    | api       | `handleExport()` — validate URL, submit to manager, return 202                 | call    | → N21           | → U6       |
| N19 | P7    | api       | `handleExportStatus()` — return job metadata as JSON                           | call    | reads S4        | → U6       |
| N20 | P7    | api       | `handleExportPDF()` — return PDF bytes with Content-Disposition                | call    | reads S4        | → U6       |
| N21 | P2    | jobs      | `Manager.Submit()` — bounded goroutine, calls pipeline.Run()                   | call    | → N22           | → S4       |
| N22 | P2    | pipeline  | `pipeline.Run()` — reusable pipeline (extract → translate → render → validate) | call    | → existing      | → N21      |
| S4  | P2    | jobs      | `Storage` — in-memory job map with TTL cleanup via `UpdatedAt`                 | store   | —               | → N19, N20 |

**Changed wiring:** CLI mode: `main.run()` calls `pipeline.Run()` then writes files. Server mode: `Manager.Submit()` calls `pipeline.Run()` in a goroutine, stores result in S4, HTTP handlers serve from S4.

---

## V8: MCP Server

**Demo:** `claude mcp add x-article-exporter /path/to/binary -- --mcp` → registered. Then in Claude Code: "Export this article as PDF: https://x.com/..." → PDF saved to configured output dir.

**Notes:**

- Same binary, `--mcp` flag starts stdio JSON-RPC server (no HTTP)
- 4 tools: `export_article`, `get_article_info`, `list_exports`, `check_translation`
- Reuses `pipeline.Run()` from V6 extraction — same pipeline for CLI, HTTP API, and MCP
- Config via `LoadMCPConfig()`: auth creds + `output_dir` (with `~` expansion) + `dark_mode` + `ollama_model`
- Per-call overrides: `translate`, `dark_mode`, `output` parameters on `export_article`
- No API keys needed — Claude Code is the sole client via stdio pipe

**Files:**

- `internal/mcp/server.go` — MCP server init, tool registration, helpers
- `internal/mcp/tools.go` — 4 tool handlers
- `internal/mcp/tools_test.go` — 12 tests with mock pipeline
- `internal/config/mcp.go` — `MCPConfig`, `LoadMCPConfig()`, `expandHome()`
- `internal/config/mcp_test.go` — 4 config tests
- `docs/mcp-setup.md` — setup guide with `claude mcp add` instructions

---

## V9: README + LICENSE

**Demo:** Visit the GitHub repo → immediately understand what the tool does, how to install it, and how to use all three modes (CLI, API, MCP). Side-by-side light/dark screenshots show output quality.

**Notes:**

- MIT LICENSE with copyright holder `annismckenzie`
- Hero screenshots copied from `shaping/spike-typst-output/` to `docs/images/` (originals kept as spike artifacts)
- README structure: badges → screenshots → features → prerequisites → installation → quick start → config → usage (CLI, API, MCP) → how it works → contributing → license
- Config example shown inline (essential for getting started); API and MCP docs linked, not duplicated
- Auth cookie setup instructions included (biggest onboarding hurdle)
- Shields.io badges for Go version and license

**Files:**

- `LICENSE` — MIT license
- `README.md` — full project documentation
- `docs/images/example-light.png` — light mode hero screenshot
- `docs/images/example-dark.png` — dark mode hero screenshot

---

## Sliced Breadboard

```mermaid
flowchart TB
    subgraph V1["V1: EXTRACT ARTICLE"]
        U1["U1: CLI args"]
        U2["U2: Progress"]
        U4["U4: Errors"]
        U5["U5: Success"]
        N1["N1: loadConfig()"]
        S1["S1: config"]
        N2["N2: extractArticleID()"]
        N5["N5: fetchArticle()"]
        N6["N6: parseArticle()"]
        S3["S3: article"]
    end

    subgraph V2["V2: RENDER PDF"]
        N7["N7: downloadImages()"]
        N10["N10: renderHTML()"]
        N11["N11: printToPDF()"]
        N13["N13: writePDF()"]
    end

    subgraph V3["V3: TRANSLATION"]
        N8["N8: TranslateArticle()"]
    end

    subgraph V4["V4: QUALITY VALIDATION"]
        N12["N12: validatePDF()"]
    end

    subgraph V5["V5: CONFIG + QUERY ID"]
        N3["N3: resolveQueryID()"]
        N4["N4: fetchQueryIDFromBundle()"]
        S2["S2: queryIDCache"]
    end

    subgraph V6["V6: WEB API"]
        U6["U6: HTTP API"]
        N17["N17: Server.Handler()"]
        N18["N18: handleExport()"]
        N19["N19: handleExportStatus()"]
        N20["N20: handleExportPDF()"]
        N21["N21: Manager.Submit()"]
        N22["N22: pipeline.Run()"]
        S4["S4: jobStorage"]
    end

    %% External systems
    N14["N14: GET TweetResultByRestId"]
    N15["N15: POST /api/chat (Ollama)"]
    N16["N16: typst compile"]

    %% Force slice ordering
    V1 ~~~ V2
    V2 ~~~ V3
    V3 ~~~ V4
    V4 ~~~ V5
    V5 ~~~ V6

    %% V1 flow
    U1 --> N1
    N1 --> S1
    N1 --> N2
    N2 --> N5
    S1 -.-> N5
    N5 --> N14
    N14 -.-> N5
    N5 --> N6
    N6 --> S3

    %% V2 flow
    N6 --> N7
    N7 --> S3
    S3 -.-> N10
    N7 --> N10
    N10 --> N11
    N11 --> N16
    N16 -.-> N11
    N11 --> N13
    N13 --> U5

    %% V3 flow (conditional)
    N6 -->|if --translate| N8
    S3 -.-> N8
    N8 --> N15
    N15 -.-> N8
    N8 --> S3
    N8 --> N7

    %% V4 flow
    N11 --> N12
    S3 -.-> N12
    N12 -.-> U3
    N12 -.->|hard fail| U4
    N12 -->|pass| N13

    %% V5 flow
    N2 --> N3
    S2 -.-> N3
    N3 -->|cache miss| N4
    N4 -.-> S2
    N3 --> N5

    %% Error/progress flows
    N5 -.->|auth fail| U4
    N5 -.-> U2
    N8 -.-> U2
    N11 -.-> U2

    %% V6 flow
    U6 --> N17
    N17 --> N18
    N17 --> N19
    N17 --> N20
    N18 --> N21
    N21 --> N22
    N22 --> N2
    N21 --> S4
    S4 -.-> N19
    S4 -.-> N20

    %% Slice boundary styling
    style V1 fill:#e8f5e9,stroke:#4caf50,stroke-width:2px
    style V2 fill:#e3f2fd,stroke:#2196f3,stroke-width:2px
    style V3 fill:#fff3e0,stroke:#ff9800,stroke-width:2px
    style V4 fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px
    style V5 fill:#fff8e1,stroke:#ffc107,stroke-width:2px
    style V6 fill:#fce4ec,stroke:#e91e63,stroke-width:2px

    classDef ui fill:#ffb6c1,stroke:#d87093,color:#000
    classDef nonui fill:#d3d3d3,stroke:#808080,color:#000
    classDef store fill:#e6e6fa,stroke:#9370db,color:#000
    classDef external fill:#b3e5fc,stroke:#0288d1,color:#000

    class U1,U2,U4,U5,U6 ui
    class N1,N2,N3,N4,N5,N6,N7,N8,N10,N11,N12,N13,N17,N18,N19,N20,N21,N22 nonui
    class N14,N15,N16 external
    class S1,S2,S3,S4 store
```

---

## Slices Grid

| Slice                      | Status      | Highlights                                                                                                                                             | Demo                                               |
| :------------------------- | :---------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------- |
| **V1: Extract Article**    | ✅ Complete | Parse CLI args, extract snowflake ID, fetch via TweetResultByRestId, parse Draft.js content_state blocks                                               | Run command, see article summary in terminal       |
| **V2: Render PDF**         | ✅ Complete | Download + base64-encode images, chromedp HTML-to-PDF, embedded OpenSans fonts, HTML + PDF output                                                      | Run command, get HTML + PDF                        |
| **V2b: Typst Renderer**    | ✅ Complete | Replace chromedp with Typst, dark mode (black bg + light text), static font TTFs, Apple Symbols fallback, blockquote left-border styling               | Same PDF, now via Typst — dark mode works fully    |
| **V3: Translation**        | ✅ Complete | --translate and --ollama-model flags, local Ollama with translategemma:12b (55 langs), batch 8 blocks with [N] delimiters, code/images skipped         | Run with --translate de, get German PDF            |
| **V4: Quality Validation** | ✅ Complete | pdfcpu structural integrity + page/image count, ledongthuc/pdf text extraction, title/author present, word count ±15%, soft warnings vs hard failures  | Run command, see validation pass/warnings          |
| **V5: Config + Query ID**  | ✅ Complete | YAML config file (~/.config/…), CLI flags override via flag.Visit(), query ID: cache (24h) → main.\*.js bundle → flag → hardcoded, helpful auth error  | Config file replaces flags, query ID auto-resolves |
| **V6: Web API**            | ✅ Complete | Pipeline extracted to reusable package, Go 1.22+ ServeMux, bounded concurrency, Bearer auth, token-bucket rate limiting, graceful shutdown             | POST URL to API, get PDF back                      |
| **V7: Thread Export**      | ⏳ Pending  | Detect thread vs article URL, walk self-reply chain, parse tweets into block model, thread-specific HTML template with tweet cards. Spike needed (A5). | Pass thread URL, get thread PDF                    |
| **V8: MCP Server**         | ✅ Complete | `--mcp` stdio server via mcp-go, 4 tools (export/info/list/check), config-driven auth + output_dir, per-call overrides, `claude mcp add` setup         | Ask Claude to export article → PDF on disk         |
| **V9: README + LICENSE**   | ✅ Complete | MIT LICENSE, README with badges/screenshots/features/install/config/usage (CLI + API + MCP), hero images in `docs/images/`                             | Visit repo, instantly understand + use             |
| **V11: GitHub Pages**      | ✅ Complete | Hugo + Hextra docs site in `site/`, 7 content pages, Mermaid diagrams, GitHub Actions deploy, `make site-dev`/`site-build`                             | Browse docs at annismckenzie.github.io             |
