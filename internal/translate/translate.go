package translate

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

const batchSize = 8

// translatableBlockTypes lists block types whose text should be translated.
var translatableBlockTypes = map[string]bool{
	"unstyled":            true,
	"header-one":          true,
	"header-two":          true,
	"header-three":        true,
	"header-four":         true,
	"header-five":         true,
	"header-six":          true,
	"unordered-list-item": true,
	"ordered-list-item":   true,
	"blockquote":          true,
}

// TranslateArticle translates all translatable blocks and the title.
// It modifies the article in-place. For translated blocks, InlineStyleRanges
// and EntityRanges are cleared (plain text translation only).
func TranslateArticle(ctx context.Context, article *model.Article, targetLang string, client *Client) error {
	// Collect indices of translatable blocks with non-empty text.
	var indices []int
	for i, block := range article.Blocks {
		if translatableBlockTypes[block.Type] && strings.TrimSpace(block.Text) != "" {
			indices = append(indices, i)
		}
	}

	if len(indices) == 0 && strings.TrimSpace(article.Title) == "" {
		log.Println("No translatable content found.")
		return nil
	}

	// Translate title.
	if strings.TrimSpace(article.Title) != "" {
		log.Println("Translating title...")
		translated, err := translateSingle(ctx, client, article.Title, targetLang)
		if err != nil {
			return fmt.Errorf("translating title: %w", err)
		}
		article.Title = translated
	}

	// Batch and translate blocks.
	totalBatches := (len(indices) + batchSize - 1) / batchSize
	for batchIdx := range totalBatches {
		start := batchIdx * batchSize
		end := min(start+batchSize, len(indices))
		batchIndices := indices[start:end]

		log.Printf("Translating batch %d/%d (%d blocks)...", batchIdx+1, totalBatches, len(batchIndices))

		texts := make([]string, len(batchIndices))
		for i, idx := range batchIndices {
			texts[i] = article.Blocks[idx].Text
		}

		translated, err := translateBatch(ctx, client, texts, targetLang)
		if err != nil {
			return fmt.Errorf("translating batch %d/%d: %w", batchIdx+1, totalBatches, err)
		}

		if len(translated) != len(batchIndices) {
			return fmt.Errorf("batch %d/%d: expected %d translated texts, got %d",
				batchIdx+1, totalBatches, len(batchIndices), len(translated))
		}

		for i, idx := range batchIndices {
			text := strings.TrimSpace(translated[i])
			if text == "" {
				log.Printf("warning: block %d translated to empty text", idx)
			}
			article.Blocks[idx].Text = text
			article.Blocks[idx].InlineStyleRanges = nil
			article.Blocks[idx].EntityRanges = nil
		}
	}

	log.Printf("Translation complete: %d blocks translated.", len(indices))
	return nil
}

func translateSingle(ctx context.Context, client *Client, text, targetLang string) (string, error) {
	prompt := buildSinglePrompt(text, targetLang)
	resp, err := client.Chat(ctx, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

func translateBatch(ctx context.Context, client *Client, texts []string, targetLang string) ([]string, error) {
	prompt := buildBatchPrompt(texts, targetLang)
	resp, err := client.Chat(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return parseBatchResponse(resp, len(texts))
}

func buildSinglePrompt(text, targetLang string) string {
	return fmt.Sprintf(
		"Translate the following text to %s. Preserve technical terms, identifiers, proper nouns, and brand names unchanged. Return only the translated text, nothing else.\n\n%s",
		targetLang, text,
	)
}

func buildBatchPrompt(texts []string, targetLang string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb,
		"Translate the following numbered paragraphs to %s. Preserve technical terms, identifiers, proper nouns, and brand names unchanged. Return the translated paragraphs in the exact same numbered format.\n\n",
		targetLang,
	)
	for i, text := range texts {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, text)
	}
	return sb.String()
}

// parseBatchResponse extracts numbered paragraphs from the model's response.
func parseBatchResponse(response string, expectedCount int) ([]string, error) {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	results := make([]string, 0, expectedCount)

	var current strings.Builder
	currentNum := 0
	lastWasBlank := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentNum > 0 {
				lastWasBlank = true
			}
			continue
		}

		num, rest, isNumbered := parseNumberedLine(trimmed)
		if isNumbered && num == currentNum+1 {
			// Save previous paragraph if any.
			if currentNum > 0 {
				results = append(results, strings.TrimSpace(current.String()))
			}
			currentNum = num
			current.Reset()
			current.WriteString(rest)
			lastWasBlank = false
		} else if currentNum > 0 {
			// Continuation of current paragraph.
			if lastWasBlank {
				current.WriteString("\n")
				lastWasBlank = false
			} else if current.Len() > 0 {
				current.WriteString(" ")
			}
			current.WriteString(trimmed)
		}
	}

	// Save last paragraph.
	if currentNum > 0 {
		results = append(results, strings.TrimSpace(current.String()))
	}

	if len(results) != expectedCount {
		return nil, fmt.Errorf("expected %d paragraphs, parsed %d from response", expectedCount, len(results))
	}

	return results, nil
}

// parseNumberedLine checks if a line starts with "N. " pattern.
func parseNumberedLine(line string) (int, string, bool) {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(line) || line[i] != '.' {
		return 0, "", false
	}
	num := 0
	for _, c := range line[:i] {
		num = num*10 + int(c-'0')
	}
	rest := strings.TrimLeft(line[i+1:], " ")
	return num, rest, true
}
