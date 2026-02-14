# Spike B: Typst

**Date:** 2026-02-14
**Status:** Complete
**Verdict:** GO — Typst produces excellent output with full dark mode support

## Method

1. Installed Typst via `brew install typst` (v0.14.2, 39 MB)
2. Downloaded Open Sans variable TTF fonts from Google Fonts repo
3. Hand-wrote `test-article-light.typ` and `test-article-dark.typ` mirroring our article structure:
   - Title, author, date header
   - Cover image + 3 inline images (extracted from test-article.html base64 data)
   - Paragraphs with inline styles (bold, italic, bold+italic, inline code, strikethrough, links)
   - Code block with Go syntax
   - Blockquotes
   - Ordered and unordered lists
   - ~6 pages of content
4. Compiled with `typst compile --root / --font-path /tmp/spike-preview/fonts`
5. Generated PNGs via ImageMagick, visually inspected

## Answers

### B-Q1: Can we produce a good-looking PDF?

**YES.** The output is excellent. Typst produces professional-quality typography with:

- Proper hyphenation and justification
- Clean page breaks
- Elegant spacing between elements
- Beautiful code block rendering with syntax highlighting

### B-Q2: Does `#set page(fill: black)` produce full-bleed dark mode on ALL pages?

**YES.** Every single page has a solid black background edge-to-edge, including the margin area. This is Typst's native behavior — no hacks needed.

### B-Q3: Do page numbers work?

**YES.** `#set page(numbering: "1 / 1")` produces "1 / 6", "2 / 6", etc. at the bottom of every page. In dark mode, the page number color automatically follows the text fill color (white on black). Fully customizable via `#set page(number-align: center)`.

### B-Q4: Can Typst load images from file paths?

**YES.** `#image("/path/to/file.jpg", width: 60%)` works when compiled with `--root /`. For our use case, we'd write base64-decoded images to a temp directory and reference them by path. Typst supports JPEG, PNG, GIF, and SVG.

### B-Q5: Does `--font-path` work?

**YES.** `--font-path /path/to/fonts` loads fonts from a local directory. Open Sans variable TTF fonts load successfully (with a warning that variable fonts "may render incorrectly" — but the output looks fine). Static TTF fonts would eliminate the warning. Typst also sees all system fonts, so system-installed fonts work without `--font-path`.

### B-Q6: How does styled text look?

| Style         | Typst Markup         | Rendering                               |
| ------------- | -------------------- | --------------------------------------- |
| Bold          | `*bold*`             | Excellent                               |
| Italic        | `_italic_`           | Excellent                               |
| Bold+Italic   | `*_bold italic_*`    | Excellent                               |
| Inline code   | `` `code` ``         | Good — monospace with subtle background |
| Strikethrough | `#strike[text]`      | Works                                   |
| Link          | `#link("url")[text]` | Blue, clickable                         |

All inline styles render correctly. The link color can be customized with `#show link: set text(fill: rgb("#1d9bf0"))`.

### B-Q7: How do code blocks render?

**Excellent.** Typst has built-in syntax highlighting for Go (and many other languages):

````typst
```go
func main() { ... }
```
````

- Keywords colored (blue/red)
- Strings, types, and identifiers properly colored
- Monospace font (configurable)
- Background fill and border radius customizable via `#show raw.where(block: true)`
- In dark mode: dark background with light syntax colors

### B-Q8: Output quality vs Chrome?

| Aspect            | Chrome                   | Typst                                               |
| ----------------- | ------------------------ | --------------------------------------------------- |
| Typography        | Good (browser rendering) | Excellent (proper typesetter)                       |
| Hyphenation       | None                     | Automatic                                           |
| Page breaks       | Browser-decided          | Typst-decided (smarter avoidance of orphans/widows) |
| Code highlighting | None (plain monospace)   | Built-in syntax highlighting                        |
| Images            | Inline base64            | File path references                                |
| Fonts             | woff2 embedded           | TTF/OTF via --font-path                             |

Typst produces arguably **better** typography than Chrome — it's a proper typesetting engine, not a browser print function.

### B-Q9: Performance?

| Metric                | Chrome (chromedp)        | Typst         |
| --------------------- | ------------------------ | ------------- |
| Render time (6 pages) | ~1-2s                    | **0.36s**     |
| Binary size           | ~500 MB (Chromium)       | **39 MB**     |
| Cold start            | ~1-2s                    | **<0.1s**     |
| Memory usage          | ~200+ MB                 | ~20 MB        |
| Dependencies          | Chrome + chromedp Go lib | Single binary |

Typst is ~5-8x faster and uses ~10x less memory.

## Code Change Estimate

New file `internal/render/typst.go` (~250-350 LOC):

- `RenderTypst(article *Article, darkMode bool) string` — converts Article blocks to Typst source
- Block type mapping: unstyled→paragraph, headers→`=`/`==`/`===`, lists→`-`/`+`, blockquotes→`#quote`, code→backtick blocks, atomic→`#image`
- Inline style mapping: BOLD→`*`, ITALIC→`_`, CODE→backtick, STRIKETHROUGH→`#strike[]`, LINK→`#link("")[]`
- Image handling: decode base64 → write to temp dir → `#image("path")`

New file or function for compilation (~30 LOC):

- `CompileTypst(typstSource string) ([]byte, error)` — exec `typst compile` with stdin/stdout

Updates to `html.go` (~10 LOC):

- Keep HTML rendering for `.html` output (sharing/preview)
- No changes to existing HTML pipeline

Remove `chromedp` from `go.mod`.

## Risks

1. **New renderer** — ~300 LOC of new code for block/style→Typst conversion. More work than WeasyPrint.
2. **Variable font warning** — harmless but could be noisy. Fixable by bundling static TTFs.
3. **Typst binary dependency** — users need `typst` installed (`brew install typst`). Not bundled in Go binary.
4. **Entity mapping complexity** — our styled text boundary algorithm in `styled_text.go` would need a Typst equivalent.

## Preview Files

- Light mode: `/tmp/spike-preview/typst/test-light.pdf`
- Dark mode: `/tmp/spike-preview/typst/test-dark.pdf`
- PNGs: `/tmp/spike-preview/typst/light-page-*.png`, `dark-page-*.png`
- Source: `/tmp/spike-preview/test-article-light.typ`, `test-article-dark.typ`
