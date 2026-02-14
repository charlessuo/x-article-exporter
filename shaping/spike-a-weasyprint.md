# Spike A: WeasyPrint (Dockerized)

**Date:** 2026-02-14
**Status:** Complete
**Verdict:** GO — WeasyPrint solves all Chrome limitations

## Method

1. Built Docker image: `python:3.12-slim` + system libs (Pango, Cairo, gdk-pixbuf) + `weasyprint==63.1`
2. Fed our real `test-article.html` (3.1 MB, 21 base64 images, base64 woff2 fonts) directly into WeasyPrint
3. Created dark-mode variant with `@page { background-color: #000 }` + dark CSS overrides
4. Generated PNGs via ImageMagick for visual inspection
5. Compared output side-by-side with Chrome-generated PDF

## Answers

### A-Q1: Does our existing HTML render correctly in WeasyPrint?

**YES.** Our HTML with base64-encoded woff2 fonts, base64-encoded images, and all CSS renders correctly. Two warnings:

- `font-weight: 100 900` ignored (variable font weight range syntax) — text still renders, just loses weight granularity
- `print-color-adjust: exact` unknown property — this is a Chrome-specific hint, harmless to ignore

All 21 images render. All styled text (bold, italic, code, links, strikethrough) renders. Blockquotes, lists, code blocks, headers all work.

### A-Q2: Does `@page { background-color }` fill the entire page?

**YES.** `@page { background-color: #000000; }` fills the **entire page** including the margin area. This is the key finding — it solves the Chrome limitation that killed dark mode.

### A-Q3: Do CSS Paged Media page numbers work?

**YES.** The following CSS works:

```css
@page {
  @bottom-center {
    content: counter(page) " / " counter(pages);
    font-size: 9px;
    color: #8b98a5;
  }
}
```

Page numbers appear at bottom-center on every page: "1 / 16", "2 / 16", etc. Color can be customized per-mode (gray for light, muted for dark).

### A-Q4: Is content margin/padding consistent across ALL pages?

**YES.** `@page { margin: 0.75in; }` applies uniformly to every page. No more first-page-only padding issue. This replaces both the `body { padding }` CSS hack and the Chrome `WithMargin*()` API calls.

### A-Q5: Output quality vs Chrome?

**Good, with minor differences:**

| Aspect | Chrome | WeasyPrint |
|--------|--------|------------|
| Text rendering | Excellent | Good — slightly different line breaks |
| Font weight | Full variable range | Only regular weight (variable font warning) |
| Images | Identical | Identical (base64 data URIs work) |
| Code blocks | Gray background | Gray background |
| Blockquotes | Left border + gray text | Left border + gray text |
| Page count | 18 pages | 16 pages (tighter layout) |
| Page breaks | Chrome-decided | WeasyPrint-decided (different split points) |

The font-weight warning is fixable: either provide static font files alongside woff2, or use `@font-face` with explicit `font-weight: 400` / `font-weight: 700` instead of `100 900` ranges.

### A-Q6: Performance?

| Metric | Chrome (chromedp) | WeasyPrint (Docker) |
|--------|-------------------|---------------------|
| Render time | ~2-3s | ~1s |
| Cold start | ~1-2s (browser launch) | ~0.5s (container start) |
| PDF size | 2.47 MB | 2.29 MB |
| Dependency | Chrome/Chromium (~500 MB) | Docker image (257 MB) |

WeasyPrint is faster and produces slightly smaller PDFs.

## Code Change Estimate

~50 LOC in `pdf.go`:
- Replace `chromedp` import with `os/exec`
- Replace `PrintToPDF()` body with `docker run` exec
- Pipe HTML via stdin or mount as volume
- Read PDF bytes from stdout or mounted volume

~20 LOC in `html.go`:
- Add `@page` CSS block with background-color, margins, page numbers
- Fix `font-weight` in `@font-face` (explicit 400/700 instead of range)
- Remove `@media print` body padding (handled by `@page margin`)
- Add dark-mode `@page` variant

Remove `chromedp` from `go.mod` (and its transitive deps).

## Risks

1. **Docker dependency** — requires Docker/OrbStack to be running. Not ideal for a CLI tool users install via `go install`.
2. **Variable font warnings** — fixable but needs font-face CSS adjustment.
3. **Layout differences** — page breaks happen at different points than Chrome. Acceptable but different.

## Preview Files

- Light mode: `/tmp/spike-preview/weasyprint/test-light.pdf`
- Dark mode: `/tmp/spike-preview/weasyprint/test-dark.pdf`
- PNGs: `/tmp/spike-preview/weasyprint/light-page-*.png`, `dark-page-*.png`
