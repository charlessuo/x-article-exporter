# A5 Spike: Tweet Thread Extraction

## Context

Shape A currently extracts X articles (long-form Draft.js content via `TweetResultByRestId`). Thread export (V7) would expand this to tweet threads — chains of self-replies that form a narrative. Threads are the dominant content format on X, vastly outnumbering articles.

## Goal

Understand how threads work technically on X's API, how the tweet content model differs from articles, and what rendering approach produces readable thread PDFs.

## Questions & Answers

**A5-Q1: How do you fetch a complete thread from the X API?**

Unknown. Threads are self-reply chains (tweet A → reply-to-A by same author → reply-to-that → ...). Possible approaches: `TweetDetail` endpoint returns conversation context, or walk the chain via `in_reply_to_status_id`. Need to determine which endpoint returns the full chain in one call vs requiring multiple requests.

**A5-Q2: How do you identify all tweets in a thread?**

Unknown. Given any tweet URL in a thread, we need to find the root (walk up via `in_reply_to_status_id`) and then collect all replies by the same author (walk down). The conversation endpoint may handle this, or we may need to do it manually.

**A5-Q3: How does the tweet content model differ from the article content model?**

Articles use Draft.js `RawDraftContentState` (blocks with types, inline styles, entity map). Tweets use a simpler model: plain text (280 chars max) with `entities` (mentions, hashtags, URLs) and `extended_entities` (media). The key question is whether we can map tweet entities into the existing block model or need a parallel representation.

**A5-Q4: How should thread tweets render as PDF?**

Unknown. Articles flow as prose. Threads are discrete messages. Options:

- **Card layout**: Each tweet as a visual card with avatar, @handle, timestamp, text, media. Preserves the "thread" feel.
- **Flowing prose**: Concatenate tweet text into paragraphs, losing the per-tweet structure. Simpler but loses context.
- **Hybrid**: Cards for short threads, flowing for long ones.

The card layout is probably the right default — it's what readers expect from threads and makes each tweet's contribution clear.

**A5-Q5: What about media in thread tweets?**

Tweets can contain up to 4 images, 1 video, or 1 GIF per tweet. Media URLs are in `extended_entities.media[].media_url_https`. These can be downloaded and base64-encoded the same way article images are handled. Videos would need a poster frame or be skipped (PDF can't embed video).

**A5-Q6: What about quote tweets within threads?**

Threads sometimes quote-tweet other content. Quote tweets have a nested tweet object. We'd need to render the quoted tweet inline — probably as a smaller card within the thread card. This adds complexity and could be deferred to a later iteration.

## Acceptance

Spike is complete when we can describe:

- The specific API endpoint(s) to fetch a complete thread
- The response format and how to identify all tweets in the chain
- A mapping from tweet entities to the existing block model (or why a new model is needed)
- A concrete HTML/CSS layout for thread PDFs
- What can reuse the existing pipeline and what needs new code
