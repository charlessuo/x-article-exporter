package render

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// typstEscape escapes Typst special characters in plain text.
func typstEscape(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	for _, r := range s {
		switch r {
		case '*', '_', '`', '#', '@', '$', '<', '>', '[', ']', '\\':
			buf.WriteByte('\\')
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// renderTypstStyledText converts a block's text with InlineStyleRanges and EntityRanges
// into a Typst markup string. Uses the same boundary-based segmentation as the HTML
// renderer: splits text at every point where any style or entity begins or ends,
// then wraps each segment with its active Typst markup.
func renderTypstStyledText(block model.Block, entityMap map[string]model.Entity) string {
	runes := []rune(block.Text)
	n := len(runes)
	if n == 0 {
		return ""
	}

	boundarySet := map[int]struct{}{0: {}, n: {}}
	for _, sr := range block.InlineStyleRanges {
		boundarySet[sr.Offset] = struct{}{}
		boundarySet[min(sr.Offset+sr.Length, n)] = struct{}{}
	}
	for _, er := range block.EntityRanges {
		boundarySet[er.Offset] = struct{}{}
		boundarySet[min(er.Offset+er.Length, n)] = struct{}{}
	}

	boundaries := make([]int, 0, len(boundarySet))
	for b := range boundarySet {
		boundaries = append(boundaries, b)
	}
	sort.Ints(boundaries)

	var buf strings.Builder
	for i := 0; i < len(boundaries)-1; i++ {
		start, end := boundaries[i], boundaries[i+1]
		if start >= n || start == end {
			continue
		}
		end = min(end, n)

		segment := typstEscape(string(runes[start:end]))

		var styles []string
		for _, sr := range block.InlineStyleRanges {
			srEnd := sr.Offset + sr.Length
			if sr.Offset <= start && start < srEnd {
				styles = append(styles, strings.ToUpper(sr.Style))
			}
		}

		var activeEntity *model.Entity
		for _, er := range block.EntityRanges {
			erEnd := er.Offset + er.Length
			if er.Offset <= start && start < erEnd {
				key := strconv.Itoa(er.Key)
				if e, ok := entityMap[key]; ok {
					activeEntity = &e
				}
				break
			}
		}

		// Apply inline styles to the segment text.
		styled := applyTypstStyles(segment, styles)

		// Wrap with entity markup.
		if activeEntity != nil && activeEntity.Type == "LINK" {
			if url, ok := activeEntity.Data["url"].(string); ok {
				styled = fmt.Sprintf("#link(\"%s\")[%s]", typstEscapeString(url), styled)
			}
		}

		buf.WriteString(styled)
	}

	return buf.String()
}

// applyTypstStyles wraps a segment with Typst markup for the given styles.
// Uses inline syntax (*bold*, _italic_) when possible, falls back to function
// syntax for combinations that need it.
func applyTypstStyles(segment string, styles []string) string {
	if len(styles) == 0 {
		return segment
	}

	// Typst inline markup (*, _) can't span across line boundaries.
	// Split on newlines, style each line separately, rejoin with Typst line breaks.
	if strings.Contains(segment, "\n") {
		lines := strings.Split(segment, "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = applyTypstStyles(line, styles)
			}
		}
		return strings.Join(lines, " \\\n")
	}

	hasBold := false
	hasItalic := false
	hasCode := false
	hasStrike := false
	hasUnderline := false

	for _, s := range styles {
		switch s {
		case "BOLD":
			hasBold = true
		case "ITALIC":
			hasItalic = true
		case "CODE":
			hasCode = true
		case "STRIKETHROUGH":
			hasStrike = true
		case "UNDERLINE":
			hasUnderline = true
		}
	}

	result := segment

	// Code is innermost — its content shouldn't have further markup.
	if hasCode {
		// Raw text in Typst doesn't interpret markup, so use the escaped segment directly
		// but wrapped in backticks. We need the unescaped text for raw.
		return fmt.Sprintf("`%s`", strings.NewReplacer(
			"\\*", "*", "\\_", "_", "\\`", "`", "\\#", "#",
			"\\@", "@", "\\$", "$", "\\<", "<", "\\>", ">",
			"\\[", "[", "\\]", "]", "\\\\", "\\",
		).Replace(segment))
	}

	if hasItalic {
		result = "_" + result + "_"
	}
	if hasBold {
		result = "*" + result + "*"
	}
	if hasStrike {
		result = "#strike[" + result + "]"
	}
	if hasUnderline {
		result = "#underline[" + result + "]"
	}

	return result
}

// typstEscapeString escapes a string for use inside Typst's #link("...") quotes.
func typstEscapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
