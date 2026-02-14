// Spike B: Typst test — Dark mode
// Synthetic demo article for renderer feature verification

#set page(
  paper: "us-letter",
  margin: (top: 0.75in, bottom: 0.75in, left: 0.75in, right: 0.75in),
  fill: rgb("#000000"),
  numbering: "1 / 1",
  number-align: center,
)

#set text(
  font: "Open Sans",
  size: 12pt,
  fill: rgb("#e7e9ea"),
)

#set par(leading: 0.6em, spacing: 1.2em)

// Dark mode link styling
#show link: set text(fill: rgb("#1d9bf0"))

// Dark mode code block styling
#show raw.where(block: true): set block(
  fill: rgb("#1a1a1a"),
  inset: 12pt,
  radius: 4pt,
  width: 100%,
)
#show raw.where(block: true): set text(fill: rgb("#e7e9ea"))

// --- Header ---
#text(size: 24pt, weight: "bold")[Exploring the Art of PDF Rendering]

#v(4pt)
#text(size: 10pt, fill: rgb("#71767b"))[Demo Author (\@demo) · February 14, 2026]
#v(8pt)

#line(length: 100%, stroke: 0.5pt + rgb("#333"))
#v(8pt)

// --- Content paragraphs ---
This is a synthetic demo article created to verify that the PDF rendering pipeline handles all supported content types correctly. It is not a real article — it exists purely for testing and visual comparison purposes.

The renderer must faithfully convert Draft.js content blocks into Typst markup, preserving structure, formatting, and visual hierarchy. Below we exercise every content type the pipeline supports.

== *1.* *Headings and basic text*

The first thing any article renderer needs to handle is plain text organized into paragraphs with proper spacing. Each paragraph should have consistent leading and paragraph spacing, with clear visual separation between blocks.

Headings use the `==` syntax in Typst and appear as bold section dividers. They create visual structure that helps readers scan the article. A well-rendered article makes it easy to jump between sections without losing context.

Here is a second paragraph under the same heading. It demonstrates that consecutive paragraphs flow naturally with appropriate spacing. The text should wrap cleanly at the page margins without any overflow or clipping.

== *2.* *Inline formatting*

This section exercises all inline styles the renderer supports.

Here is *bold text* in the middle of a sentence. Here is _italic text_ in the middle of a sentence. Here is *_bold italic text_* combining both styles. Here is `inline code` rendered in a monospace font. Here is #strike[strikethrough text] with a line through it. And here is a #link("https://example.com")[hyperlink to example.com] that should be visually distinct.

All of these styles can be *combined in a single paragraph*. For example, you might write _a sentence_ that includes `code snippets`, *bold emphasis*, and even a #link("https://example.com/docs")[documentation link] — all flowing naturally together. The renderer must handle each style independently and in combination without breaking the line flow.

Multiple inline styles on the same word or phrase are common in technical writing. A *_bold italic phrase_* followed by `code` and then a #link("https://example.com")[link] should all render without visual artifacts or spacing issues.

== *3.* *Blockquotes*

Blockquotes are used to highlight quoted text, terminal prompts, or important callouts. They should have a distinct visual treatment — typically a left border and indented text.

#quote(block: true)[
  This is a simple blockquote. It should appear indented with a visible left border to distinguish it from regular body text. Blockquotes can span multiple lines and should maintain consistent formatting throughout.
]

Here is a paragraph between two blockquotes, demonstrating that normal text flow resumes correctly after a blockquote ends.

#quote(block: true)[
  ❯ This blockquote simulates a terminal prompt. The ❯ character is commonly used to indicate user input in command-line examples. The renderer should handle Unicode characters like this glyph without falling back to a missing-glyph placeholder.
]

Consecutive blockquotes with text between them should each have their own border and spacing, with clear visual separation.

#quote(block: true)[
  A third blockquote to verify that the renderer handles multiple sequential quoted blocks. This one is longer to test wrapping behavior within the blockquote container. When the text is long enough to wrap to a second line, the indentation and left border should continue consistently.
]

== *4.* *Code blocks and lists*

Code blocks preserve whitespace and use monospace fonts. They are essential for technical articles that include source code examples.

```go
package main

import (
    "context"
    "fmt"
    "net/http"
)

// FetchData retrieves content from the given URL.
func FetchData(ctx context.Context, url string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch: %w", err)
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}
```

Unordered lists group related items without implying order:

- First item with some descriptive text
- Second item with *bold* and _italic_ formatting
- Third item with `inline code` in it
- Fourth item to verify consistent bullet alignment

Ordered lists indicate sequential steps:

+ Step one: configure the environment and install dependencies
+ Step two: run the build pipeline with verbose output
+ Step three: verify the output matches expectations
+ Step four: deploy to the target environment

Lists can also contain longer text that wraps to multiple lines. The continuation lines should align with the text start, not the bullet or number.

== *5.* *Page breaks and longer content*

This final section provides enough text to push the article past a single page, verifying that page breaks, headers, footers, and page numbering all work correctly across multiple pages.

When rendering long-form content, the engine must handle orphan and widow control — ensuring that single lines don't get stranded at the top or bottom of a page. While Typst handles this automatically, it is worth verifying with real content that spans several pages.

The PDF validation pipeline checks several properties of the rendered output. First, it verifies structural integrity using pdfcpu — confirming that the PDF conforms to the specification and can be parsed by standard readers. Second, it performs text extraction to verify that the rendered content is searchable and selectable, not just a rasterized image.

Text extraction from Typst-generated PDFs is sometimes incomplete. The validation pipeline accounts for this with a poor-extraction threshold, accepting results where a meaningful fraction of the expected words are found even if the total count is lower than the original. This pragmatic approach avoids false negatives while still catching catastrophic rendering failures where no text is extractable at all.

=== Quality validation details

The validation pipeline runs six checks in total:

+ *Structural integrity* — pdfcpu validates the PDF structure against the specification, checking cross-reference tables, object streams, and page tree consistency.
+ *Page count* — The rendered PDF must contain at least one page. A zero-page PDF indicates a rendering failure.
+ *Text presence* — At least some text must be extractable from the PDF. A completely empty extraction suggests the content was rasterized rather than rendered as searchable text.
+ *Title match* — The article title should appear in the extracted text, confirming that the header rendering is working.
+ *Content sampling* — Random words from the source content are checked against the extracted text to verify body content rendering.
+ *File size bounds* — The PDF file size must fall within reasonable bounds. Too small suggests missing content; too large suggests embedded resources were not optimized.

=== Rendering pipeline overview

The rendering pipeline follows a straightforward sequence. Content is fetched from the source API as a JSON payload containing Draft.js content blocks. Each block has a type (unstyled, header-two, blockquote, code-block, ordered-list-item, unordered-list-item) and an array of inline style ranges marking bold, italic, and other formatting.

The parser walks each block, resolving entity references against the entity map — which is structured as an array of key-value pairs rather than a direct map. Entity types include MEDIA for images, LINK for hyperlinks, TWEMOJI for emoji graphics, and MARKDOWN for inline code.

After parsing, the article model passes to the Typst source renderer, which generates a complete `.typ` document. The renderer handles several subtleties: inline markup characters cannot span newlines in Typst, so styles must be closed and reopened at line boundaries. Static font files are required because variable fonts produce warnings. Images embedded as base64 data URIs are decoded to a temporary directory and referenced by file path.

Finally, `typst compile` converts the `.typ` source to PDF, and the validation pipeline checks the result. The entire process — fetch, parse, render, validate — takes roughly one to three seconds for a typical article, making it fast enough for interactive use.

=== Dark mode rendering

The renderer supports a dark mode that inverts the color scheme for comfortable reading in low-light environments. Dark mode uses a pure black background (`#000000`) with light text (`#e7e9ea`), matching the native dark theme. Link colors switch to the platform blue (`#1d9bf0`) for visibility against the dark background.

Typst fills the entire page including margins with the page fill color, so dark mode produces a fully black page with no white borders. Page numbers at the bottom remain visible by inheriting the light text color. Code blocks use a slightly lighter background (`#1a1a1a`) to create subtle visual separation from the page fill.

This approach differs from a "dim" theme that might use a dark navy or gray background. The pure black choice was intentional — it matches the platform's native appearance and provides maximum contrast for text readability.

=== Closing notes

This synthetic article has exercised headings, paragraphs, bold, italic, bold-italic, inline code, strikethrough, hyperlinks, blockquotes with Unicode glyphs, Go code blocks, unordered lists, ordered lists, sub-headings, and multi-page content with page numbering. It serves as a comprehensive test fixture for the PDF rendering pipeline.

Every content type that the renderer supports is represented here. If this article renders correctly — with proper formatting, page breaks, and visual hierarchy — then the pipeline is working as intended.
