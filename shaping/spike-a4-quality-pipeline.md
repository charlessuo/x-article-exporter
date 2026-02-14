# A4 Spike: Automated Quality Validation Pipeline

## Context

Shape A needs automated quality validation so the user doesn't have to manually inspect every exported PDF. The pipeline must validate extraction completeness, translation correctness (when used), and PDF rendering quality.

## Goal

Design a validation pipeline with specific Go libraries, thresholds, and a strategy for what runs per-export vs in CI.

## Questions & Answers

**A4-Q1: What Go libraries can extract text from PDFs?**

**`pdfcpu`** (Apache 2.0): `ValidateFile()`, `PageCountFile()`, `ExtractImagesRaw()`, `PDFInfo()`. **`ledongthuc/pdf`** (BSD-3): `GetPlainText()`, `GetStyledTexts()` with font/position info. Both pure Go, no CGo. `unipdf` is most accurate but commercial (AGPL/$).

**A4-Q2: Can we count images in a generated PDF?**

**Yes.** `pdfcpu.ExtractImagesRaw()` returns `[]map[int]model.Image` — image objects per page. Count them and compare against expected count from article extraction.

**A4-Q3: How accurate is PDF text extraction for self-generated PDFs?**

**Moderate for Typst output (~29% word extraction).** Typst uses CID font encoding that `ledongthuc/pdf` can't fully extract. Title and author are typically found, but word count is significantly underreported. The validation pipeline handles this via a `poorExtraction` threshold (<50%) that downgrades text checks from errors to warnings. Visual output is correct — this is purely a text extraction limitation.

**A4-Q4: Is visual regression testing worth it?**

**No, not initially.** High complexity (CGo dependency for `go-fitz` or external `pdftoppm`), moderate false positive rate (font rendering differs across OS). Structural checks catch 95% of real problems. Defer to later if layout stability becomes a concern.

**A4-Q5: How to handle golden file testing for PDFs?**

**Store metadata snapshots (JSON), not PDFs.** Golden JSON files contain expected page count range, image count, word count range, title, must-contain phrases. For deterministic PDFs (if using fpdf): `SetCreationDate()` + `SetCatalogSort()`.

**A4-Q6: What thresholds avoid false positives?**

Word count: ±15% (headers/footers/page numbers add text). Translation length: ±30%. Image count: exact match. Page count: within expected range.

## Recommended Pipeline: Two Tiers

### Tier 1: Per-Export Validation (runs every time, <200ms overhead)

| Check                                        | Library                         | Cost  | Signal                                                                                  |
| -------------------------------------------- | ------------------------------- | ----- | --------------------------------------------------------------------------------------- |
| PDF structural integrity                     | `pdfcpu.ValidateFile()`         | ~5ms  | Catches corruption                                                                      |
| Page count > 0                               | `pdfcpu.PageCountFile()`        | ~2ms  | Catches blank output                                                                    |
| Image count matches input                    | `pdfcpu.ExtractImagesRaw()`     | ~20ms | Catches broken images (note: image entities use type `MEDIA`, not `IMAGE` — count both) |
| Title present in PDF text                    | `ledongthuc/pdf.GetPlainText()` | ~50ms | Catches extraction failure                                                              |
| Author present in PDF text                   | (same extraction)               | ~0ms  | Catches extraction failure                                                              |
| Word count within ±15% of input              | (same extraction)               | ~0ms  | Catches truncation/duplication                                                          |
| Translation block count matches source       | In-memory comparison            | ~1ms  | Catches translation corruption                                                          |
| Code blocks byte-identical after translation | `bytes.Equal()`                 | ~1ms  | Catches `ignore_tags` failure                                                           |
| Translation length within ±30%               | `len()` comparison              | ~1ms  | Catches empty/gibberish translation                                                     |

**Total overhead: <200ms** — invisible to the user.

**Behavior:**

- Hard failures (missing title, missing images, PDF invalid) → print error, exit code 1
- Soft warnings (word count slightly off) → print warning, exit code 0

### Tier 2: CI Test Suite (runs in `go test`)

Additional checks against known test fixtures:

| Check                         | What it catches                                    |
| ----------------------------- | -------------------------------------------------- |
| Golden metadata comparison    | Regressions in page count, image count, word count |
| Per-page empty page detection | Rendering bugs leaving blank pages                 |
| Must-contain phrase list      | Content that should definitely appear in the PDF   |
| Translation-specific fixtures | Code blocks with LaTeX, mixed content articles     |
| Multiple article types        | Image-heavy, code-heavy, short, long articles      |

**Test fixture structure:**

```
testdata/
  fixtures/
    tech-article/
      input.json              # GraphQL response (sanitized)
      expected.json           # Expected validation metrics
    image-heavy-article/
      input.json
      expected.json
    code-article/
      input.json
      expected.json
```

**`expected.json` format:**

```json
{
  "page_count_min": 3,
  "page_count_max": 5,
  "image_count": 4,
  "word_count_min": 1500,
  "word_count_max": 2000,
  "title": "Understanding X's Architecture",
  "author": "@engineer",
  "must_contain": ["GraphQL", "REST API", "authentication"]
}
```

## Dependencies

```
github.com/pdfcpu/pdfcpu   # Apache 2.0, pure Go, PDF structural inspection
github.com/ledongthuc/pdf   # BSD-3, pure Go, text extraction
```

Both are pure Go — no CGo, no external dependencies, work on all platforms.

## Explicitly Deferred

- **Visual regression** via `go-fitz` — high complexity for marginal benefit
- **`unipdf`** for better text extraction — only if `ledongthuc/pdf` proves too lossy
- **PDF byte-level golden comparison** — fragile across Go/library versions

## Acceptance

Spike is complete. We can describe:

- The two-tier validation architecture (per-export + CI)
- Specific Go libraries and their API functions (`pdfcpu`, `ledongthuc/pdf`)
- Concrete thresholds for each check (±15% word count, ±30% translation length, exact image count)
- What runs per-export (<200ms overhead) vs CI-only (golden fixtures)
- Why visual regression is deferred (high complexity, marginal benefit over structural checks)
