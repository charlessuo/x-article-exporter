package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/annismckenzie/x-article-exporter/internal/config"
)

const (
	bearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

	// defaultQueryID is a hardcoded query ID for TweetResultByRestId.
	// This rotates every 2-4 weeks. Override with --query-id flag.
	defaultQueryID = "d6YKjvQ920F-D4Y1PruO-A"

	graphQLEndpoint = "https://x.com/i/api/graphql"
	operationName   = "TweetResultByRestId"
)

// features contains the boolean feature flags required by the GraphQL API.
// These are taken from the browser's actual request to TweetResultByRestId.
var features = map[string]bool{
	"creator_subscriptions_tweet_preview_api_enabled":                          true,
	"premium_content_api_read_enabled":                                         false,
	"communities_web_enable_tweet_community_results_fetch":                     true,
	"c9s_tweet_anatomy_moderator_badge_enabled":                                true,
	"responsive_web_grok_analyze_button_fetch_trends_enabled":                  false,
	"responsive_web_grok_analyze_post_followups_enabled":                       true,
	"responsive_web_jetfuel_frame":                                             true,
	"responsive_web_grok_share_attachment_enabled":                             true,
	"responsive_web_grok_annotations_enabled":                                  true,
	"articles_preview_enabled":                                                 true,
	"responsive_web_edit_tweet_api_enabled":                                    true,
	"graphql_is_translatable_rweb_tweet_is_translatable_enabled":               true,
	"view_counts_everywhere_api_enabled":                                       true,
	"longform_notetweets_consumption_enabled":                                  true,
	"responsive_web_twitter_article_tweet_consumption_enabled":                 true,
	"tweet_awards_web_tipping_enabled":                                         false,
	"responsive_web_grok_show_grok_translated_post":                            false,
	"responsive_web_grok_analysis_button_from_backend":                         true,
	"post_ctas_fetch_enabled":                                                  true,
	"freedom_of_speech_not_reach_fetch_enabled":                                true,
	"standardized_nudges_misinfo":                                              true,
	"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":  true,
	"longform_notetweets_rich_text_read_enabled":                               true,
	"longform_notetweets_inline_media_enabled":                                 true,
	"profile_label_improvements_pcf_label_in_post_enabled":                     true,
	"responsive_web_profile_redirect_enabled":                                  false,
	"rweb_tipjar_consumption_enabled":                                          false,
	"verified_phone_label_enabled":                                             false,
	"responsive_web_grok_image_annotation_enabled":                             true,
	"responsive_web_grok_imagine_annotation_enabled":                           true,
	"responsive_web_grok_community_note_auto_translation_is_enabled":           false,
	"responsive_web_graphql_skip_user_profile_image_extensions_enabled":        false,
	"responsive_web_graphql_timeline_navigation_enabled":                       true,
	"responsive_web_enhance_cards_enabled":                                     false,
}

// fieldToggles controls what content is returned in the response.
var fieldToggles = map[string]bool{
	"withArticleRichContentState": true,
	"withArticlePlainText":        false,
}

// FetchArticle makes the GraphQL request to fetch an X article by its tweet ID.
// Returns the raw response body bytes.
func FetchArticle(ctx context.Context, articleID string, cfg *config.Config) ([]byte, error) {
	queryID := defaultQueryID
	if cfg.QueryID != "" {
		queryID = cfg.QueryID
	}

	reqURL, err := buildRequestURL(queryID, articleID)
	if err != nil {
		return nil, fmt.Errorf("building request URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	setHeaders(req, cfg)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("authentication failed (HTTP %d) — check your auth_token and ct0 values", resp.StatusCode)
	case http.StatusNotFound:
		return nil, fmt.Errorf("article not found (HTTP 404)")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited (HTTP 429) — try again later")
	default:
		return nil, fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, truncate(string(body), 2000))
	}
}

func buildRequestURL(queryID, articleID string) (string, error) {
	variables, err := json.Marshal(map[string]any{
		"tweetId":                 articleID,
		"includePromotedContent":  false,
		"withBirdwatchNotes":      false,
		"withVoice":               false,
		"withCommunity":           false,
	})
	if err != nil {
		return "", err
	}

	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return "", err
	}

	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("variables", string(variables))
	params.Set("features", string(featuresJSON))
	params.Set("fieldToggles", string(togglesJSON))

	return fmt.Sprintf("%s/%s/%s?%s", graphQLEndpoint, queryID, operationName, params.Encode()), nil
}

func setHeaders(req *http.Request, cfg *config.Config) {
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("X-Csrf-Token", cfg.CT0)
	req.Header.Set("X-Twitter-Auth-Type", "OAuth2Session")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://x.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:146.0) Gecko/20100101 Firefox/146.0")
	req.Header.Set("Cookie", fmt.Sprintf("auth_token=%s; ct0=%s", cfg.AuthToken, cfg.CT0))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
