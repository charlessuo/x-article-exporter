package extract

import (
	"encoding/json"
	"testing"
)

func TestParseArticle(t *testing.T) {
	t.Run("valid response with rich content", func(t *testing.T) {
		response := buildTestResponse(t, "user123", "TestUser", articleResult{
			RestID:      "1234567890",
			Title:       "Test Article",
			PreviewText: "This is a preview",
			Metadata:    &articleMeta{FirstPublishedAtSecs: 1700000000},
			CoverMedia: &coverMedia{
				MediaInfo: &mediaInfo{
					OriginalImgURL: "https://pbs.twimg.com/media/test.jpg",
				},
			},
			ContentState: mustMarshal(t, map[string]any{
				"blocks": []rawBlock{
					{Key: "a1", Text: "Title", Type: "header-one"},
					{Key: "a2", Text: "Hello world", Type: "unstyled", InlineStyleRanges: []rawStyleRange{
						{Offset: 6, Length: 5, Style: "BOLD"},
					}},
					{Key: "a3", Text: "Code here", Type: "code-block"},
					{Key: "a4", Text: " ", Type: "atomic", EntityRanges: []rawEntityRange{
						{Offset: 0, Length: 1, Key: 0},
					}},
				},
				"entityMap": map[string]rawEntity{
					"0": {Type: "IMAGE", Mutability: "IMMUTABLE", Data: map[string]any{
						"src": "https://pbs.twimg.com/media/img1.jpg",
					}},
				},
			}),
		})

		article, err := ParseArticle(response)
		if err != nil {
			t.Fatalf("ParseArticle error: %v", err)
		}

		if article.Title != "Test Article" {
			t.Errorf("Title = %q, want %q", article.Title, "Test Article")
		}
		if article.ID != "1234567890" {
			t.Errorf("ID = %q, want %q", article.ID, "1234567890")
		}
		if article.Author != "TestUser (@user123)" {
			t.Errorf("Author = %q, want %q", article.Author, "TestUser (@user123)")
		}
		if article.CoverImageURL != "https://pbs.twimg.com/media/test.jpg" {
			t.Errorf("CoverImageURL = %q, want test.jpg URL", article.CoverImageURL)
		}
		if article.PublishedAt.Unix() != 1700000000 {
			t.Errorf("PublishedAt = %v, want Unix 1700000000", article.PublishedAt)
		}
		if len(article.Blocks) != 4 {
			t.Fatalf("len(Blocks) = %d, want 4", len(article.Blocks))
		}
		if article.Blocks[0].Type != "header-one" {
			t.Errorf("Blocks[0].Type = %q, want %q", article.Blocks[0].Type, "header-one")
		}
		if len(article.Blocks[1].InlineStyleRanges) != 1 {
			t.Errorf("Blocks[1] inline styles = %d, want 1", len(article.Blocks[1].InlineStyleRanges))
		}
		if article.Blocks[1].InlineStyleRanges[0].Style != "BOLD" {
			t.Errorf("Blocks[1] style = %q, want BOLD", article.Blocks[1].InlineStyleRanges[0].Style)
		}
		if len(article.EntityMap) != 1 {
			t.Fatalf("len(EntityMap) = %d, want 1", len(article.EntityMap))
		}
		if article.EntityMap["0"].Type != "IMAGE" {
			t.Errorf("EntityMap[0].Type = %q, want IMAGE", article.EntityMap["0"].Type)
		}
		if article.ImageCount() != 1 {
			t.Errorf("ImageCount = %d, want 1", article.ImageCount())
		}

		blockCounts := article.BlockCounts()
		if blockCounts["unstyled"] != 1 {
			t.Errorf("block count unstyled = %d, want 1", blockCounts["unstyled"])
		}
		if blockCounts["header-one"] != 1 {
			t.Errorf("block count header-one = %d, want 1", blockCounts["header-one"])
		}
	})

	t.Run("entity map as array of key-value pairs", func(t *testing.T) {
		// This is the format returned by TweetResultByRestId
		response := buildTestResponse(t, "user1", "User", articleResult{
			RestID: "999",
			Title:  "Array Entity Map",
			ContentState: mustMarshal(t, map[string]any{
				"blocks": []rawBlock{
					{Key: "b1", Text: " ", Type: "atomic", EntityRanges: []rawEntityRange{
						{Offset: 0, Length: 1, Key: 12},
					}},
				},
				"entityMap": []map[string]any{
					{
						"key": "12",
						"value": map[string]any{
							"type":       "MEDIA",
							"mutability": "Immutable",
							"data":       map[string]any{"src": "https://example.com/img.jpg"},
						},
					},
				},
			}),
		})

		article, err := ParseArticle(response)
		if err != nil {
			t.Fatalf("ParseArticle error: %v", err)
		}
		if len(article.EntityMap) != 1 {
			t.Fatalf("len(EntityMap) = %d, want 1", len(article.EntityMap))
		}
		if article.EntityMap["12"].Type != "MEDIA" {
			t.Errorf("EntityMap[12].Type = %q, want MEDIA", article.EntityMap["12"].Type)
		}
	})

	t.Run("empty content state", func(t *testing.T) {
		response := buildTestResponse(t, "user1", "User", articleResult{
			RestID: "111",
			Title:  "Empty",
		})

		article, err := ParseArticle(response)
		if err != nil {
			t.Fatalf("ParseArticle error: %v", err)
		}
		if len(article.Blocks) != 0 {
			t.Errorf("len(Blocks) = %d, want 0", len(article.Blocks))
		}
	})

	t.Run("tweet without article", func(t *testing.T) {
		resp := map[string]any{
			"data": map[string]any{
				"tweetResult": map[string]any{
					"result": map[string]any{
						"__typename": "Tweet",
						"rest_id":    "12345",
					},
				},
			},
		}
		data := mustMarshal(t, resp)

		_, err := ParseArticle(data)
		if err == nil {
			t.Fatal("expected error for tweet without article")
		}
	})

	t.Run("GraphQL error response", func(t *testing.T) {
		response, _ := json.Marshal(map[string]any{
			"errors": []map[string]any{
				{"message": "Not found", "code": 34},
			},
		})

		_, err := ParseArticle(response)
		if err == nil {
			t.Fatal("expected error for GraphQL error response")
		}
	})
}

// buildTestResponse wraps an articleResult in the full TweetResultByRestId GraphQL response envelope.
func buildTestResponse(t *testing.T, screenName, displayName string, artResult articleResult) []byte {
	t.Helper()
	artJSON := mustMarshal(t, artResult)
	resp := map[string]any{
		"data": map[string]any{
			"tweetResult": map[string]any{
				"result": map[string]any{
					"__typename": "Tweet",
					"rest_id":    "tweet_" + artResult.RestID,
					"core": map[string]any{
						"user_results": map[string]any{
							"result": map[string]any{
								"core": map[string]any{
									"name":        displayName,
									"screen_name": screenName,
								},
							},
						},
					},
					"article": map[string]any{
						"article_results": map[string]any{
							"result": json.RawMessage(artJSON),
						},
					},
				},
			},
		},
	}
	return mustMarshal(t, resp)
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}
