package extract

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// GraphQL response envelope types — navigates the TweetResultByRestId response
// down to the article content.

type graphQLResponse struct {
	Data struct {
		TweetResult struct {
			Result json.RawMessage `json:"result"`
		} `json:"tweetResult"`
	} `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type tweetResult struct {
	TypeName string `json:"__typename"`
	RestID   string `json:"rest_id"`
	Core     *struct {
		UserResults struct {
			Result struct {
				Core struct {
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				} `json:"core"`
			} `json:"result"`
		} `json:"user_results"`
	} `json:"core"`
	Article *struct {
		ArticleResults struct {
			Result json.RawMessage `json:"result"`
		} `json:"article_results"`
	} `json:"article"`
	Legacy *struct {
		CreatedAt string `json:"created_at"`
	} `json:"legacy"`
}

type articleResult struct {
	RestID         string                `json:"rest_id"`
	ID             string                `json:"id"`
	Title          string                `json:"title"`
	PreviewText    string                `json:"preview_text"`
	CoverMedia     *coverMedia           `json:"cover_media"`
	ContentState   json.RawMessage       `json:"content_state"`
	Metadata       *articleMeta          `json:"metadata"`
	LifecycleState *lifecycleState       `json:"lifecycle_state"`
	MediaEntities  []articleMediaEntity  `json:"media_entities"`
}

type articleMediaEntity struct {
	MediaID   string     `json:"media_id"`
	MediaInfo *mediaInfo `json:"media_info"`
}

type coverMedia struct {
	MediaInfo *mediaInfo `json:"media_info"`
}

type mediaInfo struct {
	OriginalImgURL    string `json:"original_img_url"`
	OriginalImgWidth  int    `json:"original_img_width"`
	OriginalImgHeight int    `json:"original_img_height"`
}

type articleMeta struct {
	FirstPublishedAtSecs int64 `json:"first_published_at_secs"`
}

type lifecycleState struct {
	ModifiedAtSecs int64 `json:"modified_at_secs"`
}

// Draft.js raw content state types for JSON unmarshaling.
// Note: in the TweetResultByRestId response, entityMap is an array of {key, value}
// pairs rather than a map.

type rawDraftContentState struct {
	Blocks    []rawBlock    `json:"blocks"`
	EntityMap json.RawMessage `json:"entityMap"`
}

type rawBlock struct {
	Key               string            `json:"key"`
	Text              string            `json:"text"`
	Type              string            `json:"type"`
	Depth             int               `json:"depth"`
	InlineStyleRanges []rawStyleRange   `json:"inlineStyleRanges"`
	EntityRanges      []rawEntityRange  `json:"entityRanges"`
	Data              map[string]any    `json:"data"`
}

type rawStyleRange struct {
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Style  string `json:"style"`
}

type rawEntityRange struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
	Key    int `json:"key"`
}

type rawEntity struct {
	Type       string         `json:"type"`
	Mutability string         `json:"mutability"`
	Data       map[string]any `json:"data"`
}

type rawEntityEntry struct {
	Key   string    `json:"key"`
	Value rawEntity `json:"value"`
}

// ParseArticle parses the raw GraphQL response bytes into an Article.
func ParseArticle(data []byte) (*model.Article, error) {
	var resp graphQLResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling GraphQL response: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s (code %d)", resp.Errors[0].Message, resp.Errors[0].Code)
	}

	if len(resp.Data.TweetResult.Result) == 0 {
		return nil, fmt.Errorf("empty tweet result in response")
	}

	var tweet tweetResult
	if err := json.Unmarshal(resp.Data.TweetResult.Result, &tweet); err != nil {
		return nil, fmt.Errorf("unmarshaling tweet result: %w", err)
	}

	if tweet.Article == nil {
		return nil, fmt.Errorf("tweet %s does not contain an article", tweet.RestID)
	}

	if len(tweet.Article.ArticleResults.Result) == 0 {
		return nil, fmt.Errorf("empty article result in tweet response")
	}

	var artResult articleResult
	if err := json.Unmarshal(tweet.Article.ArticleResults.Result, &artResult); err != nil {
		return nil, fmt.Errorf("unmarshaling article result: %w", err)
	}

	article := &model.Article{
		ID:          artResult.RestID,
		Title:       artResult.Title,
		PreviewText: artResult.PreviewText,
	}

	// Author from tweet's user info
	if tweet.Core != nil {
		name := tweet.Core.UserResults.Result.Core.Name
		screenName := tweet.Core.UserResults.Result.Core.ScreenName
		if screenName != "" {
			article.Author = "@" + screenName
			if name != "" {
				article.Author = name + " (@" + screenName + ")"
			}
		}
	}

	// Cover image
	if artResult.CoverMedia != nil && artResult.CoverMedia.MediaInfo != nil {
		article.CoverImageURL = artResult.CoverMedia.MediaInfo.OriginalImgURL
	}

	// Timestamps
	if artResult.Metadata != nil && artResult.Metadata.FirstPublishedAtSecs > 0 {
		article.PublishedAt = time.Unix(artResult.Metadata.FirstPublishedAtSecs, 0)
	}
	if artResult.LifecycleState != nil && artResult.LifecycleState.ModifiedAtSecs > 0 {
		article.ModifiedAt = time.Unix(artResult.LifecycleState.ModifiedAtSecs, 0)
	}

	// Parse content (Draft.js)
	if len(artResult.ContentState) > 0 {
		if err := parseContentState(artResult.ContentState, article); err != nil {
			return nil, fmt.Errorf("parsing content state: %w", err)
		}
	}

	// Resolve MEDIA entity image URLs from the separate media_entities array.
	resolveMediaURLs(article, artResult.MediaEntities)

	return article, nil
}

func parseContentState(raw json.RawMessage, article *model.Article) error {
	var content rawDraftContentState
	if err := json.Unmarshal(raw, &content); err != nil {
		return fmt.Errorf("unmarshaling content state: %w", err)
	}

	// Convert raw blocks to model blocks
	article.Blocks = make([]model.Block, len(content.Blocks))
	for i, rb := range content.Blocks {
		article.Blocks[i] = model.Block{
			Key:   rb.Key,
			Text:  rb.Text,
			Type:  rb.Type,
			Depth: rb.Depth,
			Data:  rb.Data,
		}

		for _, sr := range rb.InlineStyleRanges {
			article.Blocks[i].InlineStyleRanges = append(article.Blocks[i].InlineStyleRanges, model.InlineStyleRange{
				Offset: sr.Offset,
				Length: sr.Length,
				Style:  sr.Style,
			})
		}

		for _, er := range rb.EntityRanges {
			article.Blocks[i].EntityRanges = append(article.Blocks[i].EntityRanges, model.EntityRange{
				Offset: er.Offset,
				Length: er.Length,
				Key:    er.Key,
			})
		}
	}

	// Parse entity map — can be either an array of {key, value} pairs or a map.
	article.EntityMap = make(map[string]model.Entity)
	if len(content.EntityMap) > 0 {
		if err := parseEntityMap(content.EntityMap, article); err != nil {
			return fmt.Errorf("parsing entity map: %w", err)
		}
	}

	return nil
}

func parseEntityMap(raw json.RawMessage, article *model.Article) error {
	// Try as array of {key, value} pairs first (TweetResultByRestId format)
	var entries []rawEntityEntry
	if err := json.Unmarshal(raw, &entries); err == nil {
		for _, entry := range entries {
			article.EntityMap[entry.Key] = model.Entity{
				Type:       entry.Value.Type,
				Mutability: entry.Value.Mutability,
				Data:       entry.Value.Data,
			}
		}
		return nil
	}

	// Fall back to map format (Draft.js standard / test fixtures)
	var entityMap map[string]rawEntity
	if err := json.Unmarshal(raw, &entityMap); err != nil {
		return fmt.Errorf("entityMap is neither array nor map: %w", err)
	}
	for k, re := range entityMap {
		article.EntityMap[k] = model.Entity{
			Type:       re.Type,
			Mutability: re.Mutability,
			Data:       re.Data,
		}
	}
	return nil
}

// resolveMediaURLs matches MEDIA entities (which contain mediaItems[].mediaId)
// against the article-level media_entities array to populate Data["src"] with
// the actual image URL.
func resolveMediaURLs(article *model.Article, mediaEntities []articleMediaEntity) {
	if len(mediaEntities) == 0 {
		return
	}

	// Build lookup: mediaId → image URL
	urlByMediaID := make(map[string]string, len(mediaEntities))
	for _, me := range mediaEntities {
		if me.MediaInfo != nil && me.MediaInfo.OriginalImgURL != "" {
			urlByMediaID[me.MediaID] = me.MediaInfo.OriginalImgURL
		}
	}

	for key, entity := range article.EntityMap {
		if entity.Type != "MEDIA" {
			continue
		}
		// Data["mediaItems"] is []any where each item is map[string]any with "mediaId".
		items, ok := entity.Data["mediaItems"].([]any)
		if !ok || len(items) == 0 {
			continue
		}
		firstItem, ok := items[0].(map[string]any)
		if !ok {
			continue
		}
		mediaID, ok := firstItem["mediaId"].(string)
		if !ok {
			continue
		}
		if url, found := urlByMediaID[mediaID]; found {
			entity.Data["src"] = url
			article.EntityMap[key] = entity
		}
	}
}
