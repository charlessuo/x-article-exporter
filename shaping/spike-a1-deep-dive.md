# A1 Deep Dive: Three Unknowns for X Article Extraction

This document resolves three specific unknowns identified after the A1 spike, providing concrete implementation details for building the GraphQL-based article extraction.

---

## Unknown 1: Exact Response Format of Article Fetch

### Finding: Articles are fetched as tweets via `TweetResultByRestId`

The A1 spike assumed a dedicated article endpoint (`ArticleEntityResultByRestId`). The deep dive initially identified `TwitterArticleByRestId` from library research (twitter-api-client, fa0311/twitter-openapi). **V1 implementation discovered that articles are actually fetched as tweets via `TweetResultByRestId`** — the article URL's snowflake ID is a tweet ID, and the article content is nested inside the tweet response at `data.tweetResult.result.article.article_results.result`.

**Confidence: VERIFIED** — confirmed by successful V1 end-to-end test with real article data.

### Response envelope structure

The response follows X's standard GraphQL nesting pattern. **V1 update:** The actual response goes through a tweet wrapper. The envelope is:

```json
{
  "data": {
    "tweetResult": {
      "result": {
        "__typename": "Tweet",
        "rest_id": "1234567890123456789",
        "core": {
          "user_results": {
            "result": {
              "core": {
                "name": "Display Name",
                "screen_name": "username"
              }
            }
          }
        },
        "article": {
          "article_results": {
            "result": {
              "rest_id": "1234567890123456789",
              "id": "<base64-encoded-id>",
              "title": "Article Title Here",
              "preview_text": "First ~200 chars of article text...",
              "cover_media": {
                "media_info": {
                  "original_img_url": "https://pbs.twimg.com/media/...",
                  "original_img_width": 1200,
                  "original_img_height": 675
                }
              },
              "metadata": {
                "first_published_at_secs": 1700000000
              },
              "lifecycle_state": {
                "modified_at_secs": 1700100000
              },
              "content_state": { "blocks": [], "entityMap": [] }
            }
          }
        }
      }
    }
  }
}
```

**Confidence: VERIFIED** — confirmed by V1 implementation with real API responses. Note: author info comes from the tweet wrapper (`core.user_results`), not from the article result itself.

### Article body content: Draft.js `RawDraftContentState`

The article body content is NOT included in the base response documented by fa0311's OpenAPI spec. It is only returned when the `fieldToggles` parameter includes `"withArticleRichContentState": true`. This was confirmed in the `d60/twikit` library's `gql.py`:

```python
params = {
    'fieldToggles': {
        'withArticleRichContentState': True,
        'withArticlePlainText': False,
        'withGrokAnalyze': False
    }
}
```

Twitter maintains a fork of Facebook's Draft.js at `github.com/twitter-forks/draft-js` (last updated December 2024). Article content is stored and served as **Draft.js `RawDraftContentState`** format. **V1 finding:** The field is named `content_state` (not `rich_content_state`), returned as a nested JSON object within the article result.

The Draft.js raw content state format is:

```json
{
  "blocks": [
    {
      "key": "abc12",
      "text": "This is the article title",
      "type": "header-one",
      "depth": 0,
      "inlineStyleRanges": [],
      "entityRanges": [],
      "data": {}
    },
    {
      "key": "def34",
      "text": "This is a paragraph with bold and italic text.",
      "type": "unstyled",
      "depth": 0,
      "inlineStyleRanges": [
        { "offset": 27, "length": 4, "style": "BOLD" },
        { "offset": 36, "length": 6, "style": "ITALIC" }
      ],
      "entityRanges": [
        { "offset": 10, "length": 9, "key": 0 }
      ],
      "data": {}
    },
    {
      "key": "ghi56",
      "text": "A bulleted item",
      "type": "unordered-list-item",
      "depth": 0,
      "inlineStyleRanges": [],
      "entityRanges": [],
      "data": {}
    },
    {
      "key": "jkl78",
      "text": "console.log('hello');",
      "type": "code-block",
      "depth": 0,
      "inlineStyleRanges": [],
      "entityRanges": [],
      "data": {}
    },
    {
      "key": "mno90",
      "text": " ",
      "type": "atomic",
      "depth": 0,
      "inlineStyleRanges": [],
      "entityRanges": [
        { "offset": 0, "length": 1, "key": 1 }
      ],
      "data": {}
    },
    {
      "key": "pqr12",
      "text": "A blockquote paragraph",
      "type": "blockquote",
      "depth": 0,
      "inlineStyleRanges": [],
      "entityRanges": [],
      "data": {}
    }
  ],
  "entityMap": [
    {
      "key": "0",
      "value": {
        "type": "LINK",
        "mutability": "MUTABLE",
        "data": {
          "url": "https://example.com"
        }
      }
    },
    {
      "key": "1",
      "value": {
        "type": "MEDIA",
        "mutability": "Immutable",
        "data": {
          "src": "https://pbs.twimg.com/media/..."
        }
      }
    }
  ]
}
```

**Confidence: VERIFIED.** The Draft.js format is confirmed by Twitter's fork and by V1 implementation. The field name is `content_state`. **V1 finding:** The `entityMap` in the `TweetResultByRestId` response is an **array of `{key, value}` pairs** (not a map as in standard Draft.js). The parser handles both formats.

### Block types supported by articles

Standard Draft.js block types (confirmed by Draft.js documentation):

| Block Type               | HTML Equivalent | Used In Articles |
|--------------------------|-----------------|------------------|
| `unstyled`               | `<p>`           | Yes (paragraphs) |
| `header-one`             | `<h1>`          | Yes              |
| `header-two`             | `<h2>`          | Yes              |
| `header-three`           | `<h3>`          | Yes              |
| `header-four`            | `<h4>`          | Likely           |
| `header-five`            | `<h5>`          | Unlikely         |
| `header-six`             | `<h6>`          | Unlikely         |
| `unordered-list-item`    | `<li>` (ul)     | Yes              |
| `ordered-list-item`      | `<li>` (ol)     | Yes              |
| `blockquote`             | `<blockquote>`  | Yes              |
| `code-block`             | `<pre>`         | Yes              |
| `atomic`                 | `<figure>`      | Yes (media/embeds)|

### Inline formatting (inlineStyleRanges)

Draft.js default inline styles:

| Style           | Meaning       |
|-----------------|---------------|
| `BOLD`          | Bold text     |
| `ITALIC`        | Italic text   |
| `UNDERLINE`     | Underlined    |
| `CODE`          | Inline code   |
| `STRIKETHROUGH` | Strikethrough |

Each range is `{ "offset": <int>, "length": <int>, "style": "<STYLE>" }`. Multiple styles can overlap on the same character range.

**Confidence: HIGH** -- these are standard Draft.js. Twitter may add custom styles but these are the baseline.

### Entity types (entityMap)

Entities are referenced by key in `entityRanges` and defined in `entityMap`. **V1 finding:** In the `TweetResultByRestId` response, `entityMap` is an **array of `{key, value}` pairs**, not a standard Draft.js map. Entity types observed in real articles:

| Entity Type  | Mutability   | Data Fields                                    | Notes                              |
|--------------|--------------|------------------------------------------------|------------------------------------|
| `LINK`       | `MUTABLE`    | `url`                                          | Hyperlinks                         |
| `MEDIA`      | `Immutable`  | `src`                                          | Images — used with `atomic` blocks |
| `TWEMOJI`    | `IMMUTABLE`  | (emoji data)                                   | Twitter emoji entities             |
| `MARKDOWN`   | `IMMUTABLE`  | (formatting data)                              | Markdown-style formatting          |
| `MENTION`    | `IMMUTABLE`  | `user_id`, `screen_name`                       | @mentions (Twitter-specific)       |
| `HASHTAG`    | `IMMUTABLE`  | `tag`                                          | #hashtags (Twitter-specific)       |

**V1 finding:** Images use entity type `MEDIA` (not `IMAGE`). Mutability values use mixed casing (`Immutable` vs `IMMUTABLE`). The `ImageCount()` method checks for both `IMAGE` and `MEDIA` types.

**Confidence: VERIFIED** for LINK, MEDIA, TWEMOJI, MARKDOWN — observed in real V1 test. MENTION and HASHTAG are plausible but not yet confirmed in article content.

### Verified in V1

1. **Field name:** `content_state` (not `rich_content_state`)
2. **Content format:** JSON object (not a JSON string) — no double-parsing needed
3. **Entity types:** `LINK`, `MEDIA`, `TWEMOJI`, `MARKDOWN` confirmed. `MEDIA` (not `IMAGE`) for images.
4. **Entity map format:** Array of `{key, value}` pairs in the API response (not a standard Draft.js map)
5. **Author information:** Comes from the tweet wrapper at `core.user_results.result.core.{name, screen_name}`, not from the article result

### Still to verify

1. **Custom block types** beyond standard Draft.js (e.g., for LaTeX) — not yet encountered
2. **Embedded tweet blocks** — exact entity type not yet observed

---

## Unknown 2: How to Extract Query IDs from X's JS Bundles

### How query IDs are embedded

X's frontend is a React SPA bundled with Webpack. The JavaScript is split into chunks, including a specific `api` chunk that contains all GraphQL operation definitions. Each operation is a module export with this structure:

```javascript
{
  queryId: "d6YKjvQ920F-D4Y1PruO-A",
  operationName: "TweetResultByRestId",
  operationType: "query",
  metadata: {
    featureSwitches: [
      "responsive_web_twitter_article_data_v2_enabled",
      "responsive_web_graphql_exclude_directive_enabled",
      // ... more feature switch names
    ]
  }
}
```

**Confidence: HIGH** -- confirmed from the Zenn.dev article documenting Twitter Mobile's GraphQL API structure, and consistent with the webpack chunk inspection approach.

### Bundle URL discovery

The JS bundles are hosted at `abs.twimg.com` with hash-based filenames:

- **Main bundle**: `https://abs.twimg.com/responsive-web/client-web-legacy/main.{hash}.js` (contains the bearer token)
- **API chunk**: `https://abs.twimg.com/responsive-web/client-web/api.{hash}.js` (contains all GraphQL operation definitions)

The process to discover the current bundle URLs:

1. Fetch `https://x.com` (no auth needed, just a regular GET)
2. Parse the HTML for `<script>` tags or `<link rel="preload">` tags
3. Find the URL matching the pattern `*/client-web*/api.*.js` or `*/main.*.js`

**Confidence: HIGH** -- confirmed from multiple sources.

### Browser console extraction method (for testing)

From the Zenn.dev reverse engineering documentation, you can extract all query IDs from the browser console:

```javascript
// Step 1: Get the API webpack chunk
let apiChunk = window.webpackChunk_twitter_responsive_web
  .filter(e => e[0][0] == "api")[0];

// Step 2: Execute each module to get its exports
let modules = {};
let operations = {};
Object.keys(apiChunk[1]).forEach(k => {
  modules[k] = {};
  try { apiChunk[1][k](modules[k]); } catch(e) {}
});

// Step 3: Collect all GraphQL operations
Object.keys(modules).forEach(k => {
  if (modules[k].exports && modules[k].exports.queryId) {
    operations[modules[k].exports.operationName] = modules[k].exports;
  }
});

// Step 4: Get the article endpoint
console.log(operations["TweetResultByRestId"]);
// { queryId: "...", operationName: "TweetResultByRestId", ... }
```

**Confidence: HIGH** -- this is a documented, tested approach.

### Programmatic extraction in Go (for the CLI tool)

The approach for our Go CLI:

1. **Fetch the X homepage** (`GET https://x.com`) and parse out `<script>` tag `src` URLs
2. **Find the API bundle URL** by matching against `api.*.js` in the script URLs
3. **Fetch the API bundle** JavaScript file
4. **Extract query ID + operation name pairs** using a regex pattern

The regex pattern to extract from the minified JS bundle:

```go
// Pattern matches the webpack module export format for GraphQL operations
// The queryId and operationName are adjacent string literals in the minified code
pattern := `\{queryId:"([^"]+)",operationName:"([^"]+)",operationType:"([^"]+)"`
```

Alternative pattern (if the format differs slightly in the minified output):

```go
// Some builds use Object.defineProperty or e.exports patterns
pattern := `queryId:"([^"]+)"[^}]*operationName:"([^"]+)"`
```

**Confidence: MEDIUM** -- the extraction concept is well-proven, but the exact regex pattern depends on the current minification format. The minified JS changes with each build. We should test against the actual current bundle and be prepared to adjust the regex.

### How existing scrapers handle this

| Library | Language | Approach |
|---------|----------|----------|
| **twitter-api-client** (trevorhobenshield) | Python | **Hardcoded** query IDs in `constants.py`, manually updated |
| **twikit** (d60) | Python | **Hardcoded** query IDs in endpoint URL constants, manually updated |
| **twscrape** (vladkens) | Python | **Hardcoded** query IDs, manually updated |
| **yt-dlp** | Python | **Hardcoded** in `_GRAPHQL_ENDPOINT` class attribute |
| **fa0311/twitter-openapi** | Spec | Uses `tools/build.py` to **auto-generate** the spec (method unclear, likely involves bundle parsing) |
| **imperatrona/twitter-scraper** | Go | **Hardcoded** query IDs |

**Key insight**: Every major scraper library hardcodes query IDs and updates them manually. None dynamically extract from JS bundles at runtime. This is a strong signal that:
- Dynamic extraction is harder than it sounds (minification changes, bundle URL changes)
- Manual updates every 2-4 weeks are the pragmatic approach most projects take
- Our tool should support BOTH: try dynamic extraction, fall back to hardcoded

### Recommended implementation strategy

```
1. On first run (or when cache is stale):
   a. Fetch https://x.com
   b. Parse <script> tags for API bundle URL
   c. Fetch the API bundle JS
   d. Apply regex to extract queryId + operationName pairs
   e. Cache the results to ~/.config/x-article-exporter/query_ids.json with timestamp

2. On subsequent runs:
   a. Check cache age (TTL: 24 hours)
   b. If fresh, use cached query IDs
   c. If stale, re-extract (step 1)

3. On API call failure (400/403/404):
   a. Force re-extract query IDs
   b. Retry once
   c. If still failing, prompt user to update manually

4. Fallback: support --query-id flag for manual override
```

### What must be verified in code

1. **The exact HTML structure** of `https://x.com` to find the script tags (may be in `<link rel="preload">` instead)
2. **The exact regex** that matches the current minified bundle format
3. **Whether fetching `x.com` without auth** returns the full page with script tags (it likely does since the JS is needed for the SPA shell)
4. **The URL prefix** (`abs.twimg.com` vs another CDN domain)

---

## Unknown 3: Exact Request Format for Article Fetch

### Complete URL format

```
GET https://x.com/i/api/graphql/{queryId}/TweetResultByRestId?variables={...}&features={...}&fieldToggles={...}
```

Where:
- `{queryId}` is the current query ID extracted from the JS bundle (e.g., `d6YKjvQ920F-D4Y1PruO-A` -- this rotates every 2-4 weeks)
- `variables`, `features`, and `fieldToggles` are URL-encoded JSON strings

**Confidence: VERIFIED** — confirmed by V1 implementation.

### Required HTTP headers

```http
GET /i/api/graphql/{queryId}/TweetResultByRestId?variables=...&features=...&fieldToggles=... HTTP/2
Host: x.com
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA
X-Csrf-Token: <ct0 cookie value>
X-Twitter-Auth-Type: OAuth2Session
X-Twitter-Active-User: yes
X-Twitter-Client-Language: en
Cookie: auth_token=<auth_token>; ct0=<ct0>
Referer: https://x.com/
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36
Content-Type: application/json
```

**Confidence: HIGH** -- confirmed from twitter-api-client `util.py` (`get_headers` function) and twikit source code.

### The hardcoded bearer token

```
AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA
```

This is X's web client bearer token. It is:
- **Hardcoded** in X's JavaScript frontend (in the `main.js` bundle)
- **The same for all users** -- it identifies the X web client application, not a specific user
- **Long-lived** -- it has been the same token for years (confirmed in a 2023 Hacker News discussion and still used in 2025+ libraries)
- There is also a legacy version: `AAAAAAAAAAAAAAAAAAAAAIK1zgAAAAAA2tUWuhGZ2JceoId5GwYWU5GspY4%3DUq7gzFoCZs1QfwGoVdvSac3IniczZEYXIcDyumCauIXpcAPorE`

**Confidence: HIGH** -- confirmed in twitter-api-client, yt-dlp, twikit, and multiple other sources.

### The `variables` parameter

```json
{
  "tweetId": "<tweet_snowflake_id>",
  "includePromotedContent": true,
  "withBirdwatchNotes": true,
  "withVoice": true,
  "withCommunity": true
}
```

**V1 finding:** The article URL's snowflake ID (`https://x.com/{user}/article/<id>` or `https://x.com/i/article/<id>`) IS a tweet ID. The variable name is `tweetId` (not `rest_id`). Additional boolean parameters are required.

**Confidence: VERIFIED** — confirmed by V1 implementation.

### The `features` parameter

Based on multiple sources (RSSHub issue #18894, twitter-api-client, twikit), the features flags needed:

```json
{
  "rweb_tipjar_consumption_enabled": true,
  "responsive_web_graphql_exclude_directive_enabled": true,
  "verified_phone_label_enabled": false,
  "creator_subscriptions_tweet_preview_api_enabled": true,
  "responsive_web_graphql_timeline_navigation_enabled": true,
  "responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
  "communities_web_enable_tweet_community_results_fetch": true,
  "c9s_tweet_anatomy_moderator_badge_enabled": true,
  "articles_preview_enabled": true,
  "responsive_web_edit_tweet_api_enabled": true,
  "graphql_is_translatable_rweb_tweet_is_translatable_enabled": true,
  "view_counts_everywhere_api_enabled": true,
  "longform_notetweets_consumption_enabled": true,
  "responsive_web_twitter_article_tweet_consumption_enabled": true,
  "tweet_awards_web_tipping_enabled": false,
  "creator_subscriptions_quote_tweet_preview_enabled": false,
  "freedom_of_speech_not_reach_fetch_enabled": true,
  "standardized_nudges_misinfo": true,
  "tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
  "rweb_video_timestamps_enabled": true,
  "longform_notetweets_rich_text_read_enabled": true,
  "longform_notetweets_inline_media_enabled": true,
  "responsive_web_enhance_cards_enabled": false,
  "responsive_web_twitter_article_data_v2_enabled": true
}
```

Note: `responsive_web_twitter_article_data_v2_enabled` and `responsive_web_twitter_article_tweet_consumption_enabled` are the most important for article content.

**Confidence: MEDIUM-HIGH** -- the exact set of required features may vary; the server likely tolerates extra features but may require certain ones. The article-specific flags are critical.

### The `fieldToggles` parameter

```json
{
  "withArticleRichContentState": true,
  "withArticlePlainText": false,
  "withGrokAnalyze": false
}
```

This is the critical parameter that controls whether the article body content is included. Without `withArticleRichContentState: true`, only metadata (title, preview_text, cover_media) is returned.

**Confidence: HIGH** -- confirmed from twikit `gql.py` source code.

### Complete curl example

```bash
curl -s -G 'https://x.com/i/api/graphql/d6YKjvQ920F-D4Y1PruO-A/TweetResultByRestId' \
  --data-urlencode 'variables={"tweetId":"1234567890123456789","includePromotedContent":true,"withBirdwatchNotes":true,"withVoice":true,"withCommunity":true}' \
  --data-urlencode 'features={...}' \
  --data-urlencode 'fieldToggles={"withArticleRichContentState":true,"withArticlePlainText":false,"withGrokAnalyze":false}' \
  -H 'Authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA' \
  -H 'X-Csrf-Token: YOUR_CT0_VALUE' \
  -H 'X-Twitter-Auth-Type: OAuth2Session' \
  -H 'X-Twitter-Active-User: yes' \
  -H 'X-Twitter-Client-Language: en' \
  -H 'Cookie: auth_token=YOUR_AUTH_TOKEN; ct0=YOUR_CT0_VALUE' \
  -H 'Referer: https://x.com/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:137.0) Gecko/20100101 Firefox/137.0'
```

**NOTE**: The query ID `d6YKjvQ920F-D4Y1PruO-A` was current as of V1 implementation. Query IDs rotate every 2-4 weeks. The `x-client-transaction-id` header is NOT required (verified in V1).

### Verified in V1

1. **Query ID:** `d6YKjvQ920F-D4Y1PruO-A` for `TweetResultByRestId` (as of V1 implementation — rotates every 2-4 weeks)
2. **Features:** A set of ~32 boolean flags is needed; the server tolerates extras but may require certain ones (see `client.go` for the full list)
3. **`fieldToggles`:** Passed as a separate query parameter (not nested inside `variables`)
4. **Response shape:** `data.tweetResult.result.article.article_results.result` contains the article with `content_state` as Draft.js
5. **Article body:** JSON object (not a JSON-encoded string)
6. **Error responses:** HTTP 403 = missing/invalid auth; HTTP 400 = missing required features; empty `result: {}` = wrong operation or missing `fieldToggles`

---

## Summary: Confidence Matrix

| Item | Confidence | Source(s) |
|------|------------|-----------|
| Operation is `TweetResultByRestId` (articles are tweets) | VERIFIED | V1 implementation with real API |
| Variables: `{"tweetId": "<snowflake_id>", ...}` | VERIFIED | V1 implementation |
| Bearer token value | VERIFIED | V1 implementation (same token used by all libraries) |
| Required HTTP headers | VERIFIED | V1 implementation |
| Feature flags dict (~32 boolean flags) | VERIFIED | V1 implementation (from browser dev tools) |
| `fieldToggles.withArticleRichContentState: true` unlocks body | VERIFIED | V1 implementation |
| Article body field is `content_state` (not `rich_content_state`) | VERIFIED | V1 implementation |
| Article body is Draft.js `RawDraftContentState` format | VERIFIED | V1 implementation |
| Entity map is array of `{key, value}` pairs (not a map) | VERIFIED | V1 implementation |
| Entity types: MEDIA (not IMAGE), LINK, TWEMOJI, MARKDOWN | VERIFIED | V1 implementation |
| Author info from tweet wrapper `core.user_results` | VERIFIED | V1 implementation |
| `x-client-transaction-id` header NOT required | VERIFIED | V1 implementation |
| Draft.js block types and inline styles | HIGH | Draft.js official docs |
| Query IDs are in webpack `api` chunk | HIGH | Zenn.dev reverse engineering doc |
| Browser console extraction of query IDs | HIGH | Documented and tested approach |
| Regex extraction from minified JS | MEDIUM | Concept is sound; exact regex needs tuning |
| All scrapers hardcode query IDs | HIGH | Verified for 5 major libraries |

---

## Status: Resolved by V1

All unknowns have been resolved by V1 implementation. Key discoveries vs. initial research:

| Item | Research prediction | V1 reality |
|------|-------------------|------------|
| Operation | `TwitterArticleByRestId` | `TweetResultByRestId` (articles are tweets) |
| Variable | `{"rest_id": "..."}` | `{"tweetId": "...", ...}` |
| Response path | `data.article.article_results.result` | `data.tweetResult.result.article.article_results.result` |
| Content field | `rich_content_state` (guessed) | `content_state` |
| Entity map | Standard Draft.js map | Array of `{key, value}` pairs |
| Image entity type | `IMAGE` | `MEDIA` |
| Author info | Assumed in article result | In tweet wrapper at `core.user_results` |

The library-based research was directionally correct (GraphQL + Draft.js + cookie auth) but wrong on specifics. The browser dev tools `Copy as cURL` approach resolved everything in one step.
