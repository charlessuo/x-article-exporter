package model

import (
	"fmt"
	"time"
)

// Article holds the parsed article metadata and content.
type Article struct {
	ID            string
	Title         string
	PreviewText   string
	Author        string // may be empty if not in API response
	PublishedAt   time.Time
	ModifiedAt    time.Time
	CoverImageURL string
	Blocks        []Block
	EntityMap     map[string]Entity
}

// Block represents a single Draft.js content block.
type Block struct {
	Key               string
	Text              string
	Type              string // "unstyled", "header-one", "code-block", "atomic", etc.
	Depth             int
	InlineStyleRanges []InlineStyleRange
	EntityRanges      []EntityRange
	Data              map[string]any
}

// InlineStyleRange marks a styled range within a block's text.
type InlineStyleRange struct {
	Offset int
	Length int
	Style  string // "BOLD", "ITALIC", "UNDERLINE", "CODE", "STRIKETHROUGH"
}

// EntityRange references an entity from the entity map within a block's text.
type EntityRange struct {
	Offset int
	Length int
	Key    int
}

// Entity represents a Draft.js entity (link, image, mention, etc.).
type Entity struct {
	Type       string // "LINK", "IMAGE", "MEDIA", "MENTION", "HASHTAG", "TWEMOJI", "MARKDOWN"
	Mutability string // "MUTABLE", "IMMUTABLE", "Immutable", "Mutable"
	Data       map[string]any
}

// BlockCounts returns a map of block type → count.
func (a *Article) BlockCounts() map[string]int {
	counts := make(map[string]int)
	for _, b := range a.Blocks {
		counts[b.Type]++
	}
	return counts
}

// EntityCounts returns a map of entity type → count.
func (a *Article) EntityCounts() map[string]int {
	counts := make(map[string]int)
	for _, e := range a.EntityMap {
		counts[e.Type]++
	}
	return counts
}

// ImageCount returns the number of IMAGE and MEDIA entities.
func (a *Article) ImageCount() int {
	n := 0
	for _, e := range a.EntityMap {
		if e.Type == "IMAGE" || e.Type == "MEDIA" {
			n++
		}
	}
	return n
}

// RenderedImageCount returns the number of images that actually appear in the
// article's content — i.e., atomic blocks that reference IMAGE/MEDIA entities.
// This may be less than ImageCount() if the entity map contains unreferenced entries.
func (a *Article) RenderedImageCount() int {
	n := 0
	for _, b := range a.Blocks {
		if b.Type != "atomic" || len(b.EntityRanges) == 0 {
			continue
		}
		key := fmt.Sprintf("%d", b.EntityRanges[0].Key)
		if e, ok := a.EntityMap[key]; ok && (e.Type == "IMAGE" || e.Type == "MEDIA") {
			n++
		}
	}
	return n
}
