package render

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// typstResult holds the generated Typst source and the temp directory
// containing decoded image files. The caller must clean up TempDir.
type typstResult struct {
	Source  string
	TempDir string // empty if no images were decoded
}

// renderTypst generates Typst source for the article. Images referenced as
// base64 data URIs are decoded to a temp directory so Typst can read them.
func renderTypst(article *model.Article, darkMode bool) (*typstResult, error) {
	imgDir, imageMap, err := decodeImages(article)
	if err != nil {
		return nil, fmt.Errorf("decoding images: %w", err)
	}

	var buf strings.Builder

	writePageSetup(&buf, darkMode)
	writeTextSetup(&buf, darkMode)
	if darkMode {
		writeDarkModeShowRules(&buf)
	}
	writeHeader(&buf, article, darkMode, imageMap)
	writeContent(&buf, article, imageMap)

	return &typstResult{Source: buf.String(), TempDir: imgDir}, nil
}

func writePageSetup(buf *strings.Builder, darkMode bool) {
	buf.WriteString("#set page(\n")
	buf.WriteString("  paper: \"us-letter\",\n")
	buf.WriteString("  margin: (top: 0.75in, bottom: 0.75in, left: 0.75in, right: 0.75in),\n")
	buf.WriteString("  numbering: \"1 / 1\",\n")
	buf.WriteString("  number-align: center,\n")
	if darkMode {
		buf.WriteString("  fill: rgb(\"#000000\"),\n")
	}
	buf.WriteString(")\n\n")
}

func writeTextSetup(buf *strings.Builder, darkMode bool) {
	buf.WriteString("#set text(\n")
	buf.WriteString("  font: (\"Open Sans\", \"Noto Sans Symbols 2\"),\n")
	buf.WriteString("  size: 12pt,\n")
	if darkMode {
		buf.WriteString("  fill: rgb(\"#e7e9ea\"),\n")
	}
	buf.WriteString(")\n\n")
	buf.WriteString("#set par(leading: 0.6em, spacing: 1.2em)\n\n")
}

func writeDarkModeShowRules(buf *strings.Builder) {
	buf.WriteString("#show link: set text(fill: rgb(\"#1d9bf0\"))\n\n")

	buf.WriteString("#show raw.where(block: true): set block(\n")
	buf.WriteString("  fill: rgb(\"#1a1a1a\"),\n")
	buf.WriteString("  inset: 12pt,\n")
	buf.WriteString("  radius: 4pt,\n")
	buf.WriteString("  width: 100%,\n")
	buf.WriteString(")\n")
	buf.WriteString("#show raw.where(block: true): set text(fill: rgb(\"#e7e9ea\"))\n\n")

	buf.WriteString("#show raw.where(block: false): box.with(\n")
	buf.WriteString("  fill: rgb(\"#1a1a1a\"),\n")
	buf.WriteString("  inset: (x: 3pt, y: 1pt),\n")
	buf.WriteString("  radius: 2pt,\n")
	buf.WriteString(")\n")
	buf.WriteString("#show raw.where(block: false): set text(fill: rgb(\"#e7e9ea\"))\n\n")
}

func writeHeader(buf *strings.Builder, article *model.Article, darkMode bool, imageMap map[string]string) {
	buf.WriteString(fmt.Sprintf("#text(size: 24pt, weight: \"bold\")[%s]\n\n", typstEscape(article.Title)))

	meta := ""
	if article.Author != "" {
		meta += typstEscape(article.Author)
	}
	if !article.PublishedAt.IsZero() {
		if meta != "" {
			meta += " · "
		}
		meta += article.PublishedAt.Format("January 2, 2006")
	}
	if meta != "" {
		secondaryColor := "#555"
		if darkMode {
			secondaryColor = "#8b98a5"
		}
		buf.WriteString("#v(4pt)\n")
		buf.WriteString(fmt.Sprintf("#text(size: 10pt, fill: rgb(\"%s\"))[%s]\n", secondaryColor, meta))
		buf.WriteString("#v(8pt)\n\n")
	}

	if article.CoverImageURL != "" {
		path := resolveImagePath(article.CoverImageURL, imageMap)
		buf.WriteString("#figure(\n")
		buf.WriteString(fmt.Sprintf("  image(\"%s\", width: 100%%),\n", typstEscapeString(path)))
		buf.WriteString(")\n\n")
	}

	lineColor := "#ddd"
	if darkMode {
		lineColor = "#333"
	}
	buf.WriteString(fmt.Sprintf("#line(length: 100%%, stroke: 0.5pt + rgb(\"%s\"))\n", lineColor))
	buf.WriteString("#v(8pt)\n\n")
}

func writeContent(buf *strings.Builder, article *model.Article, imageMap map[string]string) {
	groups := groupBlocks(article.Blocks)
	for _, g := range groups {
		switch g.Tag {
		case "ul":
			for _, b := range g.Blocks {
				text := renderTypstStyledText(b, article.EntityMap)
				buf.WriteString(fmt.Sprintf("- %s\n", text))
			}
			buf.WriteByte('\n')
		case "ol":
			for _, b := range g.Blocks {
				text := renderTypstStyledText(b, article.EntityMap)
				buf.WriteString(fmt.Sprintf("+ %s\n", text))
			}
			buf.WriteByte('\n')
		case "pre":
			var lines []string
			for _, b := range g.Blocks {
				lines = append(lines, b.Text)
			}
			buf.WriteString("```\n")
			buf.WriteString(strings.Join(lines, "\n"))
			buf.WriteString("\n```\n\n")
		case "blockquote":
			buf.WriteString("#block(\n")
			buf.WriteString("  inset: (left: 12pt, rest: 0pt),\n")
			buf.WriteString("  stroke: (left: 3pt + rgb(\"#ccc\")),\n")
			buf.WriteString(")[\n")
			for _, b := range g.Blocks {
				text := renderTypstStyledText(b, article.EntityMap)
				if text == "" {
					buf.WriteString("  #v(0.5em)\n")
				} else {
					buf.WriteString(fmt.Sprintf("  %s\n", text))
				}
			}
			buf.WriteString("]\n\n")
		default:
			for _, b := range g.Blocks {
				writeBlock(buf, b, article.EntityMap, imageMap)
			}
		}
	}
}

func writeBlock(buf *strings.Builder, block model.Block, entityMap map[string]model.Entity, imageMap map[string]string) {
	switch block.Type {
	case "unstyled":
		text := renderTypstStyledText(block, entityMap)
		if text == "" {
			buf.WriteString("#v(0.5em)\n")
		} else {
			text = strings.ReplaceAll(text, "\n", " \\\n")
			buf.WriteString(text + "\n\n")
		}
	case "header-one":
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(fmt.Sprintf("= %s\n\n", text))
	case "header-two":
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(fmt.Sprintf("== %s\n\n", text))
	case "header-three":
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(fmt.Sprintf("=== %s\n\n", text))
	case "header-four":
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(fmt.Sprintf("==== %s\n\n", text))
	case "header-five":
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(fmt.Sprintf("===== %s\n\n", text))
	case "header-six":
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(fmt.Sprintf("====== %s\n\n", text))
	case "atomic":
		writeAtomicBlock(buf, block, entityMap, imageMap)
	default:
		text := renderTypstStyledText(block, entityMap)
		buf.WriteString(text + "\n\n")
	}
}

func writeAtomicBlock(buf *strings.Builder, block model.Block, entityMap map[string]model.Entity, imageMap map[string]string) {
	if len(block.EntityRanges) == 0 {
		return
	}
	key := strconv.Itoa(block.EntityRanges[0].Key)
	entity, ok := entityMap[key]
	if !ok {
		return
	}

	switch entity.Type {
	case "IMAGE", "MEDIA":
		src, _ := entity.Data["src"].(string)
		if src == "" {
			return
		}
		path := resolveImagePath(src, imageMap)
		buf.WriteString("#figure(\n")
		buf.WriteString(fmt.Sprintf("  image(\"%s\", width: 60%%),\n", typstEscapeString(path)))
		buf.WriteString(")\n\n")
	}
}

// resolveImagePath returns the temp file path for a data URI, or the original URL.
func resolveImagePath(src string, imageMap map[string]string) string {
	if p, ok := imageMap[src]; ok {
		return p
	}
	return src
}

// decodeImages extracts base64 data URI images from the article into a temp directory.
// Returns the temp dir path and a map from original data URI → file path.
// Returns ("", nil, nil) if there are no data URI images.
func decodeImages(article *model.Article) (string, map[string]string, error) {
	// Collect all data URIs that need decoding.
	var dataURIs []string
	if isDataURI(article.CoverImageURL) {
		dataURIs = append(dataURIs, article.CoverImageURL)
	}
	for _, b := range article.Blocks {
		if b.Type != "atomic" || len(b.EntityRanges) == 0 {
			continue
		}
		key := strconv.Itoa(b.EntityRanges[0].Key)
		e, ok := article.EntityMap[key]
		if !ok {
			continue
		}
		if e.Type == "IMAGE" || e.Type == "MEDIA" {
			if src, _ := e.Data["src"].(string); isDataURI(src) {
				dataURIs = append(dataURIs, src)
			}
		}
	}

	if len(dataURIs) == 0 {
		return "", nil, nil
	}

	tmpDir, err := os.MkdirTemp("", "typst-images-*")
	if err != nil {
		return "", nil, err
	}

	imageMap := make(map[string]string, len(dataURIs))
	for i, uri := range dataURIs {
		if _, done := imageMap[uri]; done {
			continue
		}
		ext := dataURIExtension(uri)
		data, err := decodeDataURI(uri)
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", nil, fmt.Errorf("decoding image %d: %w", i, err)
		}
		path := filepath.Join(tmpDir, fmt.Sprintf("image-%d%s", i, ext))
		if err := os.WriteFile(path, data, 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", nil, err
		}
		imageMap[uri] = path
	}

	return tmpDir, imageMap, nil
}

func isDataURI(s string) bool {
	return strings.HasPrefix(s, "data:")
}

func dataURIExtension(uri string) string {
	// data:image/jpeg;base64,... → .jpg
	if strings.HasPrefix(uri, "data:image/jpeg") {
		return ".jpg"
	}
	if strings.HasPrefix(uri, "data:image/png") {
		return ".png"
	}
	if strings.HasPrefix(uri, "data:image/gif") {
		return ".gif"
	}
	if strings.HasPrefix(uri, "data:image/webp") {
		return ".webp"
	}
	return ".bin"
}

func decodeDataURI(uri string) ([]byte, error) {
	// Format: data:<mediatype>;base64,<data>
	idx := strings.Index(uri, ",")
	if idx < 0 {
		return nil, fmt.Errorf("invalid data URI: no comma found")
	}
	return base64.StdEncoding.DecodeString(uri[idx+1:])
}
