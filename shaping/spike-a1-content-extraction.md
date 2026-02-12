# A1 Spike: X Article Content Extraction

## Context

Shape A needs to extract content from X articles (long-form, NOT regular tweets). We need to understand how articles work technically and what extraction approaches are viable.

## Goal

Understand the technical structure of X articles, available APIs, auth requirements, and identify concrete steps to extract article content programmatically.

## Questions & Answers

| #         | Question                                                | Answer                                                                                                                                                                                                                                                                               |
| --------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **A1-Q1** | What are X articles and how do they differ from tweets? | Distinct entity type. Up to ~100k chars, rich formatting (headings, lists, code blocks, LaTeX, images, embedded tweets). URL: `https://x.com/i/article/{snowflake_id}`. Require Premium subscription to create. Separate from note tweets (long posts up to 25k chars).              |
| **A1-Q2** | Is article content in the initial HTML response?        | **No.** X is a React SPA. The HTML is a shell with empty `articleEntities` in `__INITIAL_STATE__`. Zero content server-side. Bot user agents get a 404 page. No og:title, no og:description, no meta tags with content.                                                              |
| **A1-Q3** | Does the official X API v2 support articles?            | **No.** No article endpoints exist. The API has `note_tweet` fields on tweets for long posts, but articles are a completely separate entity type with no public API coverage.                                                                                                        |
| **A1-Q4** | What internal APIs exist for articles?                  | GraphQL API at `x.com/i/api/graphql/{queryId}/{operationName}`. Key endpoint: **`TweetResultByRestId`** (GET, fetches article content via its parent tweet ID). Also: `UserArticlesTweets`, `ArticleTimeline`, `ArticleEntitiesSlice`. **V1 finding:** articles are fetched as tweets — the article URL's snowflake ID is a tweet ID, and the article content is nested inside the tweet response. |
| **A1-Q5** | What authentication is required?                        | **Mandatory.** Cookie-based auth (`auth_token` + `ct0` cookies) from a real browser session is the most reliable. Guest tokens are heavily nerfed and likely insufficient for articles. Required headers: `Authorization: Bearer {hardcoded_token}`, `X-Csrf-Token: {ct0}`, cookies. |
| **A1-Q6** | How stable are the GraphQL endpoints?                   | **Unstable.** Query IDs (`queryId` in the URL) rotate every 2-4 weeks. Must be extracted from X's JS bundles dynamically or updated manually. Feature flags must also be sent with requests.                                                                                         |
| **A1-Q7** | What does the article content model look like?          | Block-based rich text (similar to ProseMirror/Slate). Up to 10,000 blocks, 25 media items, 100 char title. Supports: headings, paragraphs, bold/italic/strikethrough, bulleted/numbered lists, block quotes, code blocks, LaTeX, embedded tweets.                                    |
| **A1-Q8** | Do any existing tools/libraries handle articles?        | **No.** No tool in any language has dedicated article support. Go's `imperatrona/twitter-scraper` handles tweets and note tweets but not the article entity type. Would need to be built from scratch.                                                                               |
| **A1-Q9** | What are the ToS and rate limit implications?           | X's ToS (Sept 2023) explicitly bans scraping without written consent. Internal GraphQL: ~300 req/hr, datacenter IPs banned. However, *X v. Bright Data* ruling suggests ToS scraping bans may not be fully enforceable. Personal/low-volume use is low risk.                         |

## Key Findings

### Two viable extraction approaches

**Approach 1: Headless browser (chromedp)**
- Use Go's `chromedp` to load article page with real browser cookies
- Wait for content to render, then extract from DOM
- Pros: Most reliable, handles JS rendering, can screenshot/print-to-PDF directly
- Cons: Slow (~5-10s per article), heavy dependency, browser fingerprint needed

**Approach 2: Direct GraphQL API calls**
- Call `TweetResultByRestId` with cookie auth + correct headers
- Parse the tweet-wrapped JSON response to extract article content
- Pros: Fast, lightweight, precise structured data
- Cons: Query IDs rotate every 2-4 weeks (maintenance burden), need to reverse-engineer response format, feature flags must be kept current

### Authentication: cookie-based is the only viable path

- User provides `auth_token` and `ct0` cookies from their browser session
- These can be extracted from browser dev tools or a browser extension
- Cookies are long-lived (weeks to months for `auth_token`)
- No username/password login — X has made this unreliable since Fall 2023

### Article content is a block-based rich text model

The JSON response from GraphQL contains structured blocks:
- Each block has a type (paragraph, heading, list, code, quote, etc.)
- Text blocks contain inline formatting (bold, italic, strikethrough)
- Media blocks reference image/video URLs
- Embedded tweet blocks reference tweet IDs
- This structured data maps cleanly to intermediate representations for PDF rendering

## Implications for Shape A

| Part                   | Implication                                                                                                                                                                             |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **A1 (Extraction)**    | Two sub-approaches: headless browser vs direct GraphQL. Both need cookie auth. GraphQL gives structured data (better for translation + PDF), headless is more resilient to API changes. |
| **A2 (Translation)**   | GraphQL approach gives clean text blocks — easy to translate individually. Headless approach would need DOM text extraction first.                                                      |
| **A3 (PDF rendering)** | GraphQL's structured blocks map naturally to a document model. Headless browser could use print-to-PDF directly but loses translation control.                                          |
| **A4 (Quality)**       | GraphQL gives deterministic structured data — easy to validate completeness (count blocks, check media URLs, verify title/author). Headless is harder to validate.                      |

## Recommendation

**GraphQL API (Approach 2) is strongly preferred** for this use case because:
1. Structured block data enables clean translation of individual text segments
2. Structured data enables precise quality validation (count blocks, verify completeness)
3. Much faster execution (~200ms vs ~5-10s)
4. The rotating query ID problem can be mitigated by extracting IDs from X's JS bundles at runtime

The query ID rotation is the main risk. Mitigation options:
- Fetch and parse `main.js` bundle to extract current query IDs on each run
- Cache extracted IDs with a TTL (e.g., 24 hours)
- Fall back to headless browser if GraphQL fails (hybrid approach)

## Acceptance

Spike is complete. We can describe:
- How X articles are stored and served (block-based rich text, client-side rendered)
- The specific API endpoint to fetch article content (`TweetResultByRestId` — articles are fetched as tweets)
- What authentication is needed (cookie-based, `auth_token` + `ct0`)
- The key risk (rotating query IDs) and mitigation strategies
- Why GraphQL is preferred over headless browser for this use case
