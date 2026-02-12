package render

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"strconv"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// renderGroup is a group of consecutive blocks that share a wrapper element.
// Consecutive list items or code blocks are grouped together; all others are
// single-block groups.
type renderGroup struct {
	Tag    string        // "ul", "ol", "pre", or "" for no wrapper
	Blocks []model.Block // one or more blocks in the group
}

// RenderHTML converts an Article into a complete HTML document string suitable
// for printing to PDF via chromedp.
func RenderHTML(article *model.Article) string {
	groups := groupBlocks(article.Blocks)

	data := templateData{
		Title:         article.Title,
		Author:        article.Author,
		PublishedAt:   article.PublishedAt.Format("January 2, 2006"),
		CoverImageURL: template.URL(article.CoverImageURL), //nolint:gosec // URLs are user-provided or base64 data URLs we generated
		Groups:        groups,
		EntityMap:     article.EntityMap,
		FontFaceCSS:   template.CSS(fontFaceCSS()), //nolint:gosec // Generated from embedded font files
	}

	var buf bytes.Buffer
	if err := htmlTmpl.Execute(&buf, data); err != nil {
		// Template errors indicate a programming bug, not runtime input issues.
		panic(fmt.Sprintf("render: template execution failed: %v", err))
	}
	return buf.String()
}

type templateData struct {
	Title         string
	Author        string
	PublishedAt   string
	CoverImageURL template.URL // Trusted: URLs are either user-provided or base64 data URLs we generated.
	Groups        []renderGroup
	EntityMap     map[string]model.Entity
	FontFaceCSS   template.CSS // Trusted: generated from embedded font files.
}

// groupBlocks pre-processes the flat block list into render groups.
func groupBlocks(blocks []model.Block) []renderGroup {
	var groups []renderGroup
	i := 0
	for i < len(blocks) {
		switch blocks[i].Type {
		case "unordered-list-item":
			j := i
			for j < len(blocks) && blocks[j].Type == "unordered-list-item" {
				j++
			}
			groups = append(groups, renderGroup{Tag: "ul", Blocks: blocks[i:j]})
			i = j
		case "ordered-list-item":
			j := i
			for j < len(blocks) && blocks[j].Type == "ordered-list-item" {
				j++
			}
			groups = append(groups, renderGroup{Tag: "ol", Blocks: blocks[i:j]})
			i = j
		case "code-block":
			j := i
			for j < len(blocks) && blocks[j].Type == "code-block" {
				j++
			}
			groups = append(groups, renderGroup{Tag: "pre", Blocks: blocks[i:j]})
			i = j
		case "blockquote":
			j := i
			for j < len(blocks) && blocks[j].Type == "blockquote" {
				j++
			}
			groups = append(groups, renderGroup{Tag: "blockquote", Blocks: blocks[i:j]})
			i = j
		default:
			groups = append(groups, renderGroup{Blocks: blocks[i : i+1]})
			i++
		}
	}
	return groups
}

// renderBlock converts a single block to its HTML representation.
// Used as a template function.
func renderBlock(block model.Block, entityMap map[string]model.Entity) template.HTML {
	text := renderStyledText(block, entityMap)

	// Convert newlines within a single block's text to <br> tags.
	// Draft.js blocks can contain \n\n for paragraph breaks within one block.
	text = strings.ReplaceAll(text, "\n", "<br>\n")

	switch block.Type {
	case "unstyled":
		if text == "" {
			return "<br>"
		}
		return template.HTML("<p>" + text + "</p>")
	case "header-one":
		return template.HTML("<h1>" + text + "</h1>")
	case "header-two":
		return template.HTML("<h2>" + text + "</h2>")
	case "header-three":
		return template.HTML("<h3>" + text + "</h3>")
	case "header-four":
		return template.HTML("<h4>" + text + "</h4>")
	case "header-five":
		return template.HTML("<h5>" + text + "</h5>")
	case "header-six":
		return template.HTML("<h6>" + text + "</h6>")
	case "blockquote":
		// Blockquotes are wrapped by the template; return just the content.
		return template.HTML(text)
	case "atomic":
		return renderAtomic(block, entityMap)
	case "unordered-list-item", "ordered-list-item":
		// List items are wrapped in <li> by the template; return just the content.
		return template.HTML(text)
	case "code-block":
		// Code blocks are joined in the template; this shouldn't be called directly.
		return template.HTML(html.EscapeString(block.Text))
	default:
		// Unknown block type: render as paragraph.
		return template.HTML("<p>" + text + "</p>")
	}
}

// renderAtomic renders an atomic block (image, media).
func renderAtomic(block model.Block, entityMap map[string]model.Entity) template.HTML {
	// Atomic blocks reference a single entity via EntityRanges.
	if len(block.EntityRanges) == 0 {
		return ""
	}
	key := strconv.Itoa(block.EntityRanges[0].Key)
	entity, ok := entityMap[key]
	if !ok {
		return ""
	}

	switch entity.Type {
	case "IMAGE", "MEDIA":
		src, _ := entity.Data["src"].(string)
		if src == "" {
			return ""
		}
		alt, _ := entity.Data["alt"].(string)
		var buf strings.Builder
		buf.WriteString("<figure>")
		fmt.Fprintf(&buf, `<img src="%s" alt="%s">`, html.EscapeString(src), html.EscapeString(alt))
		buf.WriteString("</figure>")
		return template.HTML(buf.String())
	default:
		return ""
	}
}

var funcMap = template.FuncMap{
	"renderBlock": renderBlock,
	"joinCodeLines": func(blocks []model.Block) template.HTML {
		var lines []string
		for _, b := range blocks {
			lines = append(lines, html.EscapeString(b.Text))
		}
		return template.HTML(strings.Join(lines, "\n"))
	},
}

var htmlTmpl = template.Must(template.New("article").Funcs(funcMap).Parse(articleTemplate))

const articleTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
{{.FontFaceCSS}}
` + cssStyles + `
</style>
</head>
<body>
<article>
  <header>
    <h1 class="article-title">{{.Title}}</h1>
    {{- if or .Author .PublishedAt}}
    <div class="article-meta">
      {{- if .Author}}
      <span class="author">{{.Author}}</span>
      {{- end}}
      {{- if .PublishedAt}}
      <span class="date">{{.PublishedAt}}</span>
      {{- end}}
    </div>
    {{- end}}
    {{- if .CoverImageURL}}
    <figure class="cover-image">
      <img src="{{.CoverImageURL}}" alt="Cover image">
    </figure>
    {{- end}}
  </header>

  <div class="content">
    {{- range .Groups}}
    {{- if eq .Tag "ul"}}
    <ul>
      {{- range .Blocks}}
      <li>{{renderBlock . $.EntityMap}}</li>
      {{- end}}
    </ul>
    {{- else if eq .Tag "ol"}}
    <ol>
      {{- range .Blocks}}
      <li>{{renderBlock . $.EntityMap}}</li>
      {{- end}}
    </ol>
    {{- else if eq .Tag "pre"}}
    <pre><code>{{joinCodeLines .Blocks}}</code></pre>
    {{- else if eq .Tag "blockquote"}}
    <blockquote>
      {{- range .Blocks}}
      <p>{{renderBlock . $.EntityMap}}</p>
      {{- end}}
    </blockquote>
    {{- else}}
    {{- range .Blocks}}
    {{renderBlock . $.EntityMap}}
    {{- end}}
    {{- end}}
    {{- end}}
  </div>
</article>
</body>
</html>`

const cssStyles = `
body {
  font-family: 'Open Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  font-size: 12pt;
  line-height: 1.6;
  color: #1a1a1a;
  margin: 0;
  padding: 0 0.5in;
}

article {
  max-width: 100%;
}

.article-title {
  font-size: 24pt;
  line-height: 1.2;
  margin-bottom: 8pt;
}

.article-meta {
  color: #555;
  font-size: 10pt;
  margin-bottom: 16pt;
}

.article-meta .author::after {
  content: " · ";
}

.cover-image {
  margin: 16pt 0;
}

.cover-image img {
  max-width: 100%;
  height: auto;
}

header {
  border-bottom: 1pt solid #ddd;
  padding-bottom: 8pt;
  margin-bottom: 16pt;
}

.content h1 { font-size: 20pt; margin-top: 24pt; margin-bottom: 8pt; break-before: page; }
.content h2 { font-size: 17pt; margin-top: 20pt; margin-bottom: 6pt; break-before: page; }
h3 { font-size: 14pt; margin-top: 16pt; margin-bottom: 4pt; }
h4, h5, h6 { font-size: 12pt; margin-top: 12pt; margin-bottom: 4pt; }

p {
  margin: 0 0 10pt 0;
}

a {
  color: #1a73e8;
  text-decoration: underline;
}

blockquote {
  border-left: 3pt solid #ccc;
  padding-left: 12pt;
  margin: 10pt 0;
  color: #555;
}

blockquote p {
  margin: 0;
}

pre {
  background: #f5f5f5;
  padding: 12pt;
  font-family: 'Courier New', Courier, monospace;
  font-size: 9pt;
  line-height: 1.4;
  white-space: pre-wrap;
  word-wrap: break-word;
  border-radius: 4pt;
  break-inside: avoid;
  margin: 10pt 0;
}

code {
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.9em;
}

p code {
  background: #f5f5f5;
  padding: 1pt 3pt;
  border-radius: 2pt;
}

ul, ol {
  margin: 10pt 0;
  padding-left: 24pt;
}

li {
  margin-bottom: 4pt;
}

/* Remove inner <p> margin when list items contain styled text */
li p {
  margin: 0;
  display: inline;
}

figure {
  margin: 16pt auto;
  text-align: center;
  break-inside: avoid;
  max-width: 80%;
}

figure img {
  max-width: 100%;
  height: auto;
}

img {
  break-inside: avoid;
}

@media print {
  body {
    padding: 0;
  }
}
`
