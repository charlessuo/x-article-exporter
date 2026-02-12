# A3 Spike: PDF Rendering Approach

## Context

Shape A renders structured article blocks (headings, paragraphs, lists, code blocks, images, embedded tweets) into a well-formatted, self-contained PDF. Content may be in any European language (translated via DeepL). Needs clean typography, embedded images, code block styling, page numbers.

## Goal

Evaluate PDF rendering approaches for a Go CLI tool producing professional-quality article PDFs.

## Questions & Answers

| #         | Question                                         | Answer                                                                                                                                                                                                                                                                                                                                   |
| --------- | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **A3-Q1** | Is wkhtmltopdf viable?                           | **No.** Officially archived/deprecated as of 2024. Based on deprecated QtWebKit (removed 2016). Removed from Homebrew. Not an option.                                                                                                                                                                                                    |
| **A3-Q2** | What does chromedp + HTML template involve?      | Render blocks to HTML with CSS, use `chromedp` (Go Chrome DevTools Protocol) to load HTML via `page.SetDocumentContent()` and call `page.PrintToPDF()`. Requires Chrome/Chromium installed. Full CSS print media support including page breaks, headers/footers via `WithDisplayHeaderFooter`/`WithHeaderTemplate`/`WithFooterTemplate`. |
| **A3-Q3** | Does chromedp require Chrome installed?          | **Yes.** Needs Chrome or Chromium on the system. On macOS most users have it. Docker option: `chromedp/headless-shell` image (~100MB). `chromedp` itself is pure Go, no CGo.                                                                                                                                                             |
| **A3-Q4** | What Go PDF libraries exist?                     | `jung-kurt/gofpdf` (archived 2021), `go-pdf/fpdf` (archived March 2025, Codeberg fork active), `maroto` v2 (active, higher-level API built on fpdf), `unipdf` (commercial AGPL/$), `pdfcpu` (primarily manipulation, not document generation).                                                                                           |
| **A3-Q5** | Can Go PDF libraries handle rich article layout? | Poorly. They excel at structured reports/invoices (tables, grids) but lack native support for flowing rich text with code blocks, block quotes, nested lists, and inline images. Building article layout from low-level primitives would require significant code (~1000+ LOC for layout logic).                                         |
| **A3-Q6** | What about pandoc + typst?                       | Pandoc supports `--pdf-engine=typst`. Typst is a modern Rust-based typesetter (single binary, ~tens of MB vs multi-GB LaTeX). Excellent quality. But requires two external tools (pandoc + typst) and an intermediate Markdown representation that may lose formatting fidelity for embedded tweets and complex blocks.                  |
| **A3-Q7** | What about weasyprint?                           | Python tool with C dependencies (cairo, pango, gdk-pixbuf). Go port (`go-weasyprint`) is alpha/WIP. Heavy dependency chain. Not suitable.                                                                                                                                                                                                |

## Comparison

| Criterion                   | chromedp + HTML                  | Go PDF lib (maroto/fpdf)   | pandoc + typst                | weasyprint                |
| --------------------------- | -------------------------------- | -------------------------- | ----------------------------- | ------------------------- |
| **Output quality**          | Excellent (Chrome rendering)     | Good for simple layouts    | Excellent                     | Excellent                 |
| **Rich text / code blocks** | Full CSS control                 | Manual layout (~1000+ LOC) | Good (Markdown limits)        | Full CSS control          |
| **Images**                  | Base64 in HTML or file:// URLs   | Programmatic embedding     | Markdown image refs           | CSS + HTML                |
| **Page breaks**             | CSS `break-inside: avoid`        | Manual calculation         | Automatic (typst)             | CSS `break-inside: avoid` |
| **Headers/footers**         | `WithDisplayHeaderFooter`        | Manual per-page            | Template-based                | CSS `@page`               |
| **External deps**           | Chrome/Chromium                  | None (pure Go)             | pandoc + typst                | Python + C libs           |
| **Unicode/i18n**            | Excellent (system fonts)         | TTF embedding required     | Excellent                     | Excellent                 |
| **Code complexity**         | Low (~200 LOC template + render) | High (~1000+ LOC layout)   | Medium (~300 LOC)             | Medium + dep mgmt         |
| **Cross-platform**          | macOS + Linux (Chrome needed)    | macOS + Linux (pure Go)    | macOS + Linux (install tools) | macOS + Linux (Python)    |

## Recommendation

**chromedp + HTML template** is the clear winner for this use case because:

1. **HTML/CSS is the natural representation for article content.** Blocks map directly to HTML elements (`<h1>`, `<p>`, `<pre><code>`, `<blockquote>`, `<ul>`, `<img>`). CSS handles all styling.
2. **Chrome renders HTML perfectly.** No layout code needed — the browser engine handles typography, line wrapping, image sizing, code blocks with syntax highlighting.
3. **Full CSS print media control.** `break-inside: avoid` for code blocks/images, `@page` margins, header/footer templates with page numbers.
4. **Lowest code complexity.** A Go HTML template (~100 LOC) + CSS stylesheet (~100 LOC) + chromedp render function (~50 LOC) = ~250 LOC total. Compare to ~1000+ LOC for manual PDF layout.
5. **Excellent Unicode/i18n support.** Chrome uses system fonts — umlauts, accents, and CJK characters render correctly out of the box.
6. **Chrome is already installed** on most macOS systems. For CI/Docker: `chromedp/headless-shell` image.

**The Chrome dependency is acceptable** because:
- This is a personal CLI tool, not a library
- macOS (your platform) almost certainly has Chrome
- The dependency is well-documented and easy to install
- Docker alternative exists for headless environments

### Implementation sketch

```go
// 1. Render blocks to HTML via Go template
html := renderToHTML(article, tmpl)

// 2. Use chromedp to print to PDF
ctx, cancel := chromedp.NewContext(context.Background())
defer cancel()

var pdfBuf []byte
chromedp.Run(ctx,
    chromedp.Navigate("about:blank"),
    chromedp.ActionFunc(func(ctx context.Context) error {
        frameTree, _ := page.GetFrameTree().Do(ctx)
        page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
        return nil
    }),
    chromedp.ActionFunc(func(ctx context.Context) error {
        buf, _, _ := page.PrintToPDF().
            WithPrintBackground(true).
            WithDisplayHeaderFooter(true).
            WithFooterTemplate(`<div style="font-size:9px;text-align:center;width:100%"><span class="pageNumber"></span> / <span class="totalPages"></span></div>`).
            WithMarginTop(0.5).
            WithMarginBottom(0.75).
            WithMarginLeft(0.5).
            WithMarginRight(0.5).
            Do(ctx)
        pdfBuf = buf
        return nil
    }),
)
os.WriteFile(outputPath, pdfBuf, 0644)
```

### CSS strategy

```css
@media print {
    body { font-family: Georgia, serif; font-size: 12pt; line-height: 1.6; }
    h1 { font-size: 24pt; margin-bottom: 8pt; }
    pre { background: #f5f5f5; padding: 12pt; font-family: 'Courier New', monospace;
          break-inside: avoid; }
    blockquote { border-left: 3pt solid #ccc; padding-left: 12pt; color: #555; }
    img { max-width: 100%; break-inside: avoid; }
    .header { border-bottom: 1pt solid #ddd; padding-bottom: 8pt; margin-bottom: 16pt; }
}
```

## Acceptance

Spike is complete. We can describe:
- Which rendering approach to use (chromedp + HTML template) and why
- How to implement it (Go HTML template → chromedp `SetDocumentContent` → `PrintToPDF`)
- CSS strategy for print-quality article layout
- Why the Chrome dependency is acceptable for this use case
- The code complexity is ~250 LOC (template + CSS + render function)
