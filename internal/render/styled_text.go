package render

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// renderStyledText converts a block's text with InlineStyleRanges and EntityRanges
// into an HTML string. Uses boundary-based segmentation: splits text at every point
// where any style or entity begins or ends, then wraps each segment with its active
// tags. Each segment is self-contained (open+close per segment) to avoid nesting
// issues with overlapping ranges.
func renderStyledText(block model.Block, entityMap map[string]model.Entity) string {
	runes := []rune(block.Text)
	n := len(runes)
	if n == 0 {
		return ""
	}

	// Collect all boundary positions.
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

		segment := html.EscapeString(string(runes[start:end]))

		// Determine active styles at this position.
		var styles []string
		for _, sr := range block.InlineStyleRanges {
			srEnd := sr.Offset + sr.Length
			if sr.Offset <= start && start < srEnd {
				styles = append(styles, sr.Style)
			}
		}

		// Determine active entity (at most one in practice).
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

		// Open entity tag.
		if activeEntity != nil && activeEntity.Type == "LINK" {
			if url, ok := activeEntity.Data["url"].(string); ok {
				fmt.Fprintf(&buf, `<a href="%s">`, html.EscapeString(url))
			}
		}

		// Open style tags.
		for _, s := range styles {
			buf.WriteString(styleOpenTag(s))
		}

		buf.WriteString(segment)

		// Close style tags (reverse order).
		for j := len(styles) - 1; j >= 0; j-- {
			buf.WriteString(styleCloseTag(styles[j]))
		}

		// Close entity tag.
		if activeEntity != nil && activeEntity.Type == "LINK" {
			buf.WriteString("</a>")
		}
	}

	return buf.String()
}

func styleOpenTag(style string) string {
	switch style {
	case "BOLD":
		return "<strong>"
	case "ITALIC":
		return "<em>"
	case "CODE":
		return "<code>"
	case "UNDERLINE":
		return `<span style="text-decoration:underline">`
	case "STRIKETHROUGH":
		return "<del>"
	default:
		return ""
	}
}

func styleCloseTag(style string) string {
	switch style {
	case "BOLD":
		return "</strong>"
	case "ITALIC":
		return "</em>"
	case "CODE":
		return "</code>"
	case "UNDERLINE":
		return "</span>"
	case "STRIKETHROUGH":
		return "</del>"
	default:
		return ""
	}
}
