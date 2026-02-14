# Shape A — Breadboard

## Context

Go CLI tool with a linear pipeline: load config → extract article from X → (optionally) translate via Ollama → render to PDF via Typst → validate PDF → write to disk. The user interacts entirely through the terminal (CLI args in, progress/errors/output path out).

---

## Places

| #   | Place      | Description                                                                                              |
| --- | ---------- | -------------------------------------------------------------------------------------------------------- |
| P1  | CLI        | Terminal — user invokes command, sees progress, errors, and output path                                  |
| P2  | Pipeline   | Core processing: config → extraction → translation → rendering → validation → output                     |
| P3  | X API      | External: `x.com/i/api/graphql/` (article data) and `abs.twimg.com` (JS bundles for query ID extraction) |
| P4  | DeepL API  | External: `api-free.deepl.com/v2/translate`                                                              |
| P5  | Typst      | External: `typst compile` binary for PDF rendering                                                       |
| P6  | Filesystem | Config file (`~/.config/x-article-exporter/config.yaml`), query ID cache, output PDF                     |

---

## UI Affordances

| #   | Place | Affordance                                                                                                                                           | Control | Wires Out | Returns To |
| --- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | --------- | ---------- |
| U1  | P1    | CLI invocation: `x-article-exporter <url> [--translate <lang>] [--auth-token <t>] [--ct0 <c>] [--query-id <id>] [--deepl-key <k>] [--output <path>]` | invoke  | → N1      | —          |
| U2  | P1    | Progress log ("Fetching article...", "Translating to de...", "Rendering PDF...", "Validating...")                                                    | render  | —         | —          |
| U3  | P1    | Validation warnings (word count ±15%, translation length ±30%) — exit 0                                                                              | render  | —         | —          |
| U4  | P1    | Error messages (auth failure, article not found, validation hard fail) — exit 1                                                                      | render  | —         | —          |
| U5  | P1    | Success message + output file path — exit 0                                                                                                          | render  | —         | —          |

---

## Code Affordances

| #   | Place | Component | Affordance                                                                                                                                                                                                                                                         | Control | Wires Out         | Returns To                             |
| --- | ----- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------- | ----------------- | -------------------------------------- |
| N1  | P2    | config    | `loadConfig(flags, configPath)` — parse CLI flags, read config.yaml from P6, merge (flags override file values)                                                                                                                                                    | call    | reads P6          | → S1                                   |
| N2  | P2    | extract   | `extractArticleID(url)` — parse snowflake ID from `x.com/i/article/{id}` URL                                                                                                                                                                                       | call    | —                 | → N5                                   |
| N3  | P2    | extract   | `resolveQueryID(config)` — check S2 cache (24h TTL); if miss → N4; if `--query-id` flag → use directly                                                                                                                                                             | call    | → N4 (cache miss) | → N5                                   |
| N4  | P3    | extract   | `fetchQueryIDFromBundle()` — GET main.js → find api chunk URL → GET `api.{hash}.js` → regex extract query ID                                                                                                                                                       | call    | —                 | → S2, → N3                             |
| N5  | P2    | extract   | `fetchArticle(articleID, queryID, config)` — build GET with URL-encoded `variables`, `features` (23 flags), `fieldToggles` (`withArticleRichContentState: true`), bearer token, cookie auth                                                                        | call    | → N14             | → N6                                   |
| N6  | P2    | extract   | `parseArticle(response)` — parse JSON envelope (`data.tweetResult.result.article.article_results.result`) → article metadata (title, date, author from tweet wrapper) + Draft.js `content_state` (blocks array, inline styles, entity map as `{key, value}` pairs) | call    | —                 | → S3                                   |
| N7  | P2    | extract   | `downloadImages(blocks)` — fetch image URLs from entity map, base64-encode, embed inline in block model                                                                                                                                                            | call    | —                 | updates S3                             |
| N8  | P2    | translate | `translateBlocks(blocks, targetLang, apiKey)` — XML-wrap translatable blocks, set `ignore_tags` for code/LaTeX, single POST to DeepL, unwrap response, merge back                                                                                                  | call    | → N15             | updates S3                             |
| N9  | P2    | translate | `validateTranslation(original, translated)` — block count match, code block byte-identity (`bytes.Equal`), translation length ratio ±30%                                                                                                                           | call    | —                 | → U3 (soft), → U4 (hard)               |
| N10 | P2    | render    | `renderHTML(article, blocks)` — Go `html/template` renders blocks to HTML with inline CSS for print media (`@media print`, `break-inside: avoid`, page margins, Georgia/monospace font stack)                                                                      | call    | —                 | → N11                                  |
| N11 | P2    | render    | `printToPDF(article, darkMode)` — generate Typst source from article model, decode base64 images to temp dir, write embedded fonts, `typst compile` to PDF                                                                                                         | call    | → N16             | → N12                                  |
| N12 | P2    | validate  | `validatePDF(pdfBytes, article, blocks)` — `pdfcpu.ValidateFile()` (~5ms), `PageCountFile()` > 0, `ExtractImagesRaw()` count matches expected, `ledongthuc/pdf.GetPlainText()` for title/author present + word count ±15%                                          | call    | —                 | → U3 (soft), → U4 (hard), → N13 (pass) |
| N13 | P6    | output    | `writePDF(pdfBytes, outputPath)` — `os.WriteFile(path, pdfBytes, 0644)`                                                                                                                                                                                            | call    | writes to P6      | → U5                                   |
| N14 | P3    | —         | `GET /graphql/{queryId}/TweetResultByRestId` — `authorization: Bearer {token}`, `x-csrf-token: {ct0}`, `cookie: auth_token={auth_token}; ct0={ct0}`                                                                                                                | call    | —                 | → N5                                   |
| N15 | P4    | —         | `POST /v2/translate` — `text`, `target_lang`, `tag_handling: xml`, `ignore_tags: code,latex`                                                                                                                                                                       | call    | —                 | → N8                                   |
| N16 | P5    | —         | `typst compile` — Typst CLI binary renders .typ source to PDF                                                                                                                                                                                                      | call    | —                 | → N11                                  |

---

## Data Stores

| #   | Place | Store          | Description                                                                                                                                                                                        |
| --- | ----- | -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| S1  | P2    | `config`       | Merged configuration: `auth_token`, `ct0`, `output_dir`, `translate_lang`, `deepl_key`, `query_id` (CLI flags override config file)                                                                |
| S2  | P6    | `queryIDCache` | Cached query ID + extraction timestamp, 24h TTL. File: `~/.cache/x-article-exporter/query-id.json`                                                                                                 |
| S3  | P2    | `article`      | In-memory article model: title, author, date, blocks (`[]Block` with text, type, inline styles, entities), images (base64-encoded). Mutated through pipeline: parse → download images → translate. |

---

## Diagram

```mermaid
flowchart TB
    subgraph P1["P1: CLI"]
        U1["U1: CLI args<br/>(url, --translate, --auth-token,<br/>--ct0, --query-id, --output)"]
        U2["U2: Progress log"]
        U3["U3: Warnings (exit 0)"]
        U4["U4: Errors (exit 1)"]
        U5["U5: Success + path (exit 0)"]
    end

    subgraph P2["P2: Pipeline"]
        N1["N1: loadConfig()"]
        S1["S1: config"]

        subgraph A1["A1: Content Extraction"]
            N2["N2: extractArticleID()"]
            N3["N3: resolveQueryID()"]
            N4["N4: fetchQueryIDFromBundle()"]
            N5["N5: fetchArticle()"]
            N6["N6: parseArticle()"]
            N7["N7: downloadImages()"]
        end

        S3["S3: article"]

        subgraph A2["A2: Translation (if --translate)"]
            N8["N8: translateBlocks()"]
            N9["N9: validateTranslation()"]
        end

        subgraph A3["A3: Rendering"]
            N10["N10: renderHTML()"]
            N11["N11: printToPDF()"]
        end

        subgraph A4["A4: Validation"]
            N12["N12: validatePDF()"]
        end

        N13["N13: writePDF()"]
    end

    S2["S2: queryIDCache"]

    subgraph P3["P3: X API"]
        N14["N14: GET TweetResultByRestId"]
    end

    subgraph P4["P4: DeepL API"]
        N15["N15: POST /v2/translate"]
    end

    subgraph P5["P5: Typst"]
        N16["N16: typst compile"]
    end

    %% Entry
    U1 --> N1
    N1 --> S1

    %% Config → Extraction
    N1 --> N2
    N2 --> N3
    S2 -.-> N3
    N3 -->|cache miss| N4
    N4 -.-> S2
    N3 --> N5
    S1 -.-> N5
    N5 --> N14
    N14 -.-> N5
    N5 --> N6
    N6 --> S3
    N6 --> N7
    N7 --> S3

    %% Translation (conditional)
    N7 -->|if --translate| N8
    S3 -.-> N8
    N8 --> N15
    N15 -.-> N8
    N8 --> S3
    N8 --> N9
    N9 --> N10

    %% No-translate path
    N7 -->|no --translate| N10

    %% Rendering
    S3 -.-> N10
    N10 --> N11
    N11 --> N16
    N16 -.-> N11

    %% Validation
    N11 --> N12
    S3 -.-> N12

    %% Output
    N12 -->|pass| N13
    N13 --> U5

    %% Warnings/errors
    N9 -.-> U3
    N12 -.-> U3
    N5 -.->|auth fail| U4
    N12 -.->|hard fail| U4

    %% Progress
    N5 -.-> U2
    N8 -.-> U2
    N11 -.-> U2
    N12 -.-> U2

    %% Styling
    classDef ui fill:#ffb6c1,stroke:#d87093,color:#000
    classDef nonui fill:#d3d3d3,stroke:#808080,color:#000
    classDef store fill:#e6e6fa,stroke:#9370db,color:#000
    classDef external fill:#b3e5fc,stroke:#0288d1,color:#000

    class U1,U2,U3,U4,U5 ui
    class N1,N2,N3,N4,N5,N6,N7,N8,N9,N10,N11,N12,N13 nonui
    class N14,N15,N16 external
    class S1,S2,S3 store
```

**Legend:**

- **Pink nodes (U)** = UI affordances (things the user sees/provides)
- **Grey nodes (N)** = Code affordances (functions, handlers)
- **Lavender nodes (S)** = Data stores (persistent or in-memory state)
- **Blue nodes (N14-N16)** = External system boundaries (API calls)
- **Solid lines (→)** = Wires Out (control flow: calls, triggers)
- **Dashed lines (-.->)** = Returns To (data flow: return values, store reads)
