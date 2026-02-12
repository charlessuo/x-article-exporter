# A1 Deep Dive: Three Unknowns for X Article Extraction

This document resolves three specific unknowns identified after the A1 spike, providing concrete implementation details for building the GraphQL-based article extraction.

---

## Unknown 1: Exact Response Format of `TwitterArticleByRestId`

### Finding: The operation is called `TwitterArticleByRestId`, not `ArticleEntityResultByRestId`

The A1 spike used the name `ArticleEntityResultByRestId`. Based on the `twitter-api-client` library (trevorhobenshield/twitter-api-client), the actual operation name in X's JS bundles is **`TwitterArticleByRestId`**.

**Confidence: HIGH** -- confirmed across multiple sources (twitter-api-client constants.py, fa0311/twitter-openapi spec).

### Response envelope structure

The response follows X's standard GraphQL nesting pattern. Based on the fa0311/twitter-openapi OpenAPI 3.0 specification (`src/openapi/schemas/tweet.yaml`), the envelope is:

```json
{
  "data": {
    "article": {
      "article_results": {
        "result": {
          "__typename": "ArticleResult",
          "rest_id": "1234567890123456789",
          "id": "<base64-encoded-id>",
          "title": "Article Title Here",
          "preview_text": "First ~200 chars of article text...",
          "cover_media": {
            "id": "<media-id>",
            "media_key": "<media-key>",
            "media_id": "9876543210987654321",
            "media_info": {
              "original_img_url": "https://pbs.twimg.com/media/...",
              "original_img_width": 1200,
              "original_img_height": 675,
              "color_info": {
                "palette": [
                  {
                    "rgb": { "red": 42, "green": 99, "blue": 156 },
                    "percentage": 35.5
                  }
                ]
              }
            }
          },
          "metadata": {
            "first_published_at_secs": 1700000000
          },
          "lifecycle_state": {
            "modified_at_secs": 1700100000
          }
        }
      }
    }
  }
}
```

**Confidence: HIGH** for the metadata fields above -- these are directly from the fa0311 OpenAPI spec which is auto-generated from X's actual API responses.

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

Twitter maintains a fork of Facebook's Draft.js at `github.com/twitter-forks/draft-js` (last updated December 2024). This strongly indicates that article content is stored and served as **Draft.js `RawDraftContentState`** format. The field is likely named `rich_content_state` or similar, returned as a nested object (or JSON string) within the `ArticleResult`.

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
  "entityMap": {
    "0": {
      "type": "LINK",
      "mutability": "MUTABLE",
      "data": {
        "url": "https://example.com"
      }
    },
    "1": {
      "type": "IMAGE",
      "mutability": "IMMUTABLE",
      "data": {
        "src": "https://pbs.twimg.com/media/...",
        "width": 800,
        "height": 600
      }
    }
  }
}
```

**Confidence: MEDIUM-HIGH.** The Draft.js format is confirmed by Twitter's fork. The exact field name in the response (`rich_content_state` vs `content_state` vs something else) needs verification by making an actual API call.

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

Entities are referenced by key in `entityRanges` and defined in `entityMap`. Standard entity types:

| Entity Type  | Mutability  | Data Fields                                    | Notes                         |
|--------------|-------------|------------------------------------------------|-------------------------------|
| `LINK`       | `MUTABLE`   | `url`                                          | Hyperlinks                    |
| `IMAGE`      | `IMMUTABLE` | `src`, `width`, `height`, possibly `media_key` | Used with `atomic` blocks     |
| `MENTION`    | `IMMUTABLE` | `user_id`, `screen_name`                       | @mentions (Twitter-specific)  |
| `HASHTAG`    | `IMMUTABLE` | `tag`                                          | #hashtags (Twitter-specific)  |

**Embedded tweets** are likely represented as `atomic` blocks with an entity of a custom type (e.g., `EMBEDDED_TWEET` or `TWEET`) with a `tweet_id` in the data field. This needs verification.

**Confidence: MEDIUM** -- LINK and IMAGE are standard Draft.js. Twitter-specific entity types (MENTION, HASHTAG, embedded tweets) are educated guesses based on Twitter's domain; the exact type strings need verification.

### What must be verified in code

1. **The exact field name** for the article body content in the response (when `withArticleRichContentState: true`)
2. **Whether the content is a JSON object or a JSON string** that needs parsing
3. **The exact entity types** used for embedded tweets, images, and Twitter-specific entities
4. **Whether any custom block types** beyond standard Draft.js are used (e.g., for LaTeX)
5. **How author information** is returned -- likely as a separate `user` field at the tweet/article level, not within the blocks

---

## Unknown 2: How to Extract Query IDs from X's JS Bundles

### How query IDs are embedded

X's frontend is a React SPA bundled with Webpack. The JavaScript is split into chunks, including a specific `api` chunk that contains all GraphQL operation definitions. Each operation is a module export with this structure:

```javascript
{
  queryId: "hwrvh-Qt24lcprL-BDfqRA",
  operationName: "TwitterArticleByRestId",
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
console.log(operations["TwitterArticleByRestId"]);
// { queryId: "...", operationName: "TwitterArticleByRestId", ... }
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
GET https://x.com/i/api/graphql/{queryId}/TwitterArticleByRestId?variables={...}&features={...}&fieldToggles={...}
```

Where:
- `{queryId}` is the current query ID extracted from the JS bundle (e.g., `hwrvh-Qt24lcprL-BDfqRA` -- this rotates)
- `variables`, `features`, and `fieldToggles` are URL-encoded JSON strings

**Confidence: HIGH** -- URL format confirmed across multiple sources.

### Required HTTP headers

```http
GET /i/api/graphql/{queryId}/TwitterArticleByRestId?variables=...&features=...&fieldToggles=... HTTP/2
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
  "rest_id": "<article_snowflake_id>"
}
```

The article ID is the numeric snowflake ID from the article URL (`https://x.com/i/article/<id>`).

**Confidence: HIGH** -- confirmed from twitter-api-client `constants.py` where `TwitterArticleByRestId` is defined as `{'rest_id': str}, 'hwrvh-Qt24lcprL-BDfqRA', 'TwitterArticleByRestId'`.

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
curl -s -G 'https://x.com/i/api/graphql/hwrvh-Qt24lcprL-BDfqRA/TwitterArticleByRestId' \
  --data-urlencode 'variables={"rest_id":"1234567890123456789"}' \
  --data-urlencode 'features={"rweb_tipjar_consumption_enabled":true,"responsive_web_graphql_exclude_directive_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"rweb_video_timestamps_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false,"responsive_web_twitter_article_data_v2_enabled":true}' \
  --data-urlencode 'fieldToggles={"withArticleRichContentState":true,"withArticlePlainText":false,"withGrokAnalyze":false}' \
  -H 'Authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA' \
  -H 'X-Csrf-Token: YOUR_CT0_VALUE' \
  -H 'X-Twitter-Auth-Type: OAuth2Session' \
  -H 'X-Twitter-Active-User: yes' \
  -H 'X-Twitter-Client-Language: en' \
  -H 'Cookie: auth_token=YOUR_AUTH_TOKEN; ct0=YOUR_CT0_VALUE' \
  -H 'Referer: https://x.com/' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36'
```

**NOTE**: The query ID `hwrvh-Qt24lcprL-BDfqRA` is from twitter-api-client and WILL have rotated by now. You must extract the current one first.

### What must be verified in code

1. **Whether the query ID from twitter-api-client is still current** (almost certainly not -- it rotates every 2-4 weeks)
2. **Whether all listed features are required** or if a minimal subset works
3. **Whether `fieldToggles` is passed as a separate query parameter** or nested inside `variables`
4. **The exact response shape** when `withArticleRichContentState: true` is set
5. **Whether the article body is a JSON object or a JSON-encoded string** within the response
6. **Error responses** -- what does a 400/403/404 look like? What errors indicate an expired query ID vs bad auth vs missing article?

---

## Summary: Confidence Matrix

| Item | Confidence | Source(s) |
|------|------------|-----------|
| Operation name is `TwitterArticleByRestId` | HIGH | twitter-api-client, fa0311/twitter-openapi |
| Variables: `{"rest_id": "<snowflake_id>"}` | HIGH | twitter-api-client constants.py |
| Bearer token value | HIGH | twitter-api-client, yt-dlp, twikit, HN discussion |
| Required HTTP headers | HIGH | twitter-api-client util.py, twikit, yt-dlp |
| Feature flags dict | MEDIUM-HIGH | RSSHub issue, twitter-api-client, twikit |
| `fieldToggles.withArticleRichContentState: true` unlocks body | HIGH | twikit gql.py |
| Article body is Draft.js `RawDraftContentState` format | MEDIUM-HIGH | twitter-forks/draft-js, A1 spike block model description |
| Draft.js block types and inline styles | HIGH | Draft.js official docs |
| Query IDs are in webpack `api` chunk | HIGH | Zenn.dev reverse engineering doc |
| Browser console extraction of query IDs | HIGH | Documented and tested approach |
| Regex extraction from minified JS | MEDIUM | Concept is sound; exact regex needs tuning |
| All scrapers hardcode query IDs | HIGH | Verified for 5 major libraries |

---

## Recommended Next Steps

1. **Manual verification first**: Open browser dev tools on an actual X article page, find the `TwitterArticleByRestId` request in the Network tab, and capture the full request/response. This single action resolves all remaining uncertainties.

2. **Build the curl test**: Use the curl example above with a real `auth_token` + `ct0` and a current query ID (extracted from browser dev tools) to confirm the request format works.

3. **Implement query ID extraction**: Start with the browser console method to get a known-good query ID, then build the Go regex extraction as a second step.

4. **Build Go types**: Define Go structs for the Draft.js `RawDraftContentState` format -- this is stable and well-documented regardless of any API changes.
