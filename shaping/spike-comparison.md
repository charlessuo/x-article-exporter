# Spike Comparison: WeasyPrint vs Typst

**Date:** 2026-02-14

## Summary

Both WeasyPrint and Typst solve all Chrome limitations. Both produce dark mode with full-bleed backgrounds, consistent margins, and page numbers. Both are GO.

## Side-by-Side

| Dimension                        | A: WeasyPrint (Docker)                    | B: Typst                                         |
|----------------------------------|-------------------------------------------|--------------------------------------------------|
| **Dark mode full-bleed**         | `@page { background-color }` — works      | `#set page(fill: black)` — works                 |
| **Consistent margins**           | `@page { margin: 0.75in }` — works        | `#set page(margin: 0.75in)` — works              |
| **Page numbers**                 | `@page { @bottom-center { content } }`    | `#set page(numbering: "1 / 1")`                  |
| **Rich content**                 | Reuse existing HTML+CSS                   | New Typst renderer needed                        |
| **Fonts**                        | Existing woff2 base64 (with warnings)     | TTF via `--font-path`                            |
| **Images**                       | Existing base64 data URIs                 | Write temp files, reference by path              |
| **Code syntax highlighting**     | None (same as Chrome)                     | Built-in, multi-language                         |
| **Render time (real article)**   | ~1.0s (16 pages)                          | ~0.36s (6 pages, subset)                         |
| **PDF size**                     | 2.29 MB (16 pages)                        | 568 KB (6 pages, subset)                         |
| **Dependency**                   | Docker (257 MB image)                     | Single binary (39 MB)                            |
| **Code change**                  | ~70 LOC (replace chromedp with exec)      | ~300 LOC (new Typst renderer)                    |
| **Reuses existing HTML**         | Yes — drop-in replacement                 | No — parallel renderer                           |
| **Typography quality**           | Good (browser-grade)                      | Excellent (proper typesetter)                    |

## Key Differences

### WeasyPrint: Minimal Change, Docker Dependency

**Pro:** Drop-in replacement. Our existing HTML+CSS pipeline stays intact. Change is ~70 LOC: swap chromedp exec for Docker exec, add `@page` CSS. The HTML template, styled text renderer, font embedding — all unchanged.

**Con:** Requires Docker running. Not great for `go install` distribution. The Docker image is 257 MB. Variable font weight warnings (fixable). No syntax highlighting in code blocks.

### Typst: Better Output, More Work

**Pro:** Superior typography (hyphenation, page breaks, spacing). Built-in syntax highlighting. Tiny dependency (39 MB binary, `brew install`). Faster rendering. Smaller PDFs.

**Con:** ~300 LOC of new code: a second renderer that converts Article blocks → Typst markup. The styled text boundary algorithm needs reimplementation in Typst syntax. Two rendering paths to maintain (HTML for sharing, Typst for PDF).

## Decision Framework

Choose **WeasyPrint** if:
- Minimizing code change is the priority
- Docker is already required (e.g., for V6 web API)
- Maintaining one rendering pipeline (HTML) is preferred

Choose **Typst** if:
- Output quality matters most
- Minimizing runtime dependencies matters (no Docker requirement)
- Syntax highlighting in code blocks is valuable
- The ~300 LOC investment pays off in better long-term maintainability

## Recommendation

**Typst is the stronger choice for this project.** Rationale:

1. **No Docker requirement** — the CLI tool should work with a simple `brew install typst` or even a bundled binary, not require Docker running.
2. **Better output quality** — proper typesetting with hyphenation, smart page breaks, and syntax highlighting produces noticeably better PDFs.
3. **Performance** — faster rendering, smaller binaries, lower memory.
4. **The work is bounded** — ~300 LOC for a Typst renderer that maps Article blocks → Typst markup. The mapping is mechanical: each block type and inline style has a direct Typst equivalent.
5. **HTML stays for sharing** — we keep `RenderHTML()` for `.html` output and web preview. Typst handles the PDF path.

WeasyPrint is the safe fallback — it's proven to work with zero HTML changes and could be swapped in quickly if Typst hits unexpected blockers.
