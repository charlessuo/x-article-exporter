# A2 Spike: Translation Service Selection

## Context

Shape A translates extracted article text blocks before PDF rendering. Translation is opt-in per export (`--translate <lang>`). The input is structured blocks (paragraphs, headings, list items, quotes) — code blocks and LaTeX must NOT be translated.

## Goal

Evaluate translation service options for a Go CLI tool. Understand cost, quality, Go ecosystem support, formatting preservation, and implications for the quality pipeline (A4).

## Questions & Answers

**A2-Q1: Which service has the best quality for European languages?**

**DeepL** is best-in-class for European languages (DE, FR, ES, IT). Claude 3.5 ranked #1 in WMT24 for 9/11 language pairs. Google is very good but DeepL consistently edges it out. LibreTranslate quality is poor — not suitable for polished output.

**A2-Q2: Which service best preserves formatting / skips code blocks?**

**DeepL** — native `tag_handling: "xml"` with `ignore_tags` param lets you mark code/LaTeX as untranslatable. LLMs can be prompt-instructed to skip code. Google has best-effort HTML handling but no `ignore_tags` equivalent.

**A2-Q3: What's the cost per article (~5K words / ~30K chars)?**

DeepL Free: $0 (500K chars/month ≈ 16 articles). DeepL Pro: ~$0.75. Google NMT: ~$0.60. Claude Haiku: ~$0.05. GPT-4o-mini: ~$0.006. LibreTranslate: free (self-hosted).

**A2-Q4: Is there an official Go SDK?**

DeepL: **No** (community libs only, but REST API is trivial). Google: **Yes** (`cloud.google.com/go/translate/apiv3`). Claude: **Yes** (`anthropic-sdk-go`). OpenAI: **Yes** (`openai-go`).

**A2-Q5: What's the latency for a full article?**

DeepL: 1-3s. Google NMT: 1-3s. Claude Haiku: 3-8s. GPT-4o-mini: 3-8s. LibreTranslate: 2-5s (hardware-dependent).

**A2-Q6: Is round-trip translation viable for quality validation?**

**Research says no.** EAMT 2020 paper found BLEU scores on round-trip text don't correlate with actual translation quality. If used at all, embedding-based similarity (SBERT cosine ≥ 0.85) is better than BLEU. Doubles cost and latency.

**A2-Q7: Paragraph-level vs full-document translation?**

Document-level is better (contextual coherence, consistent terminology). Recommended: concatenate all translatable blocks into one request with XML/HTML tags preserving structure. Split only if exceeding request size limits.

## Comparison

| Criterion                    | DeepL                             | Google Cloud v3             | Claude Haiku        | LibreTranslate |
| ---------------------------- | --------------------------------- | --------------------------- | ------------------- | -------------- |
| Quality (European)           | Excellent                         | Very good                   | Excellent           | Poor           |
| Cost/article                 | ~$0.75 (Pro), free tier available | ~$0.60, free tier available | ~$0.05              | Free           |
| Go SDK                       | Community only                    | Official                    | Official            | Community      |
| `ignore_tags` for code/LaTeX | Native support                    | No (manual pre-processing)  | Prompt-instructable | No             |
| Latency                      | 1-3s                              | 1-3s                        | 3-8s                | 2-5s           |
| Offline                      | No                                | No                          | No                  | Yes            |
| Languages                    | ~36                               | ~189                        | All major           | ~45            |

## Recommendation

**DeepL API as primary translation backend** because:
1. Best quality for the target languages (German, French, Spanish, Italian)
2. Native `ignore_tags` — exactly what we need for code blocks and LaTeX
3. Free tier (500K chars/month ≈ 16 articles) covers moderate personal use
4. Fast (1-3s per article)
5. Simple REST API — no official Go SDK needed, ~100 lines of custom client

**No official Go SDK is not a blocker.** DeepL's REST API needs two endpoints: `/v2/translate` and `/v2/usage`. A thin `net/http` + `encoding/json` client is straightforward.

**Translation strategy:** Send all translatable blocks as a single XML-tagged document to DeepL with `tag_handling: "xml"`. Code blocks and LaTeX pass through untranslated via `ignore_tags`. Split only if content exceeds 128 KiB request limit.

## Impact on A4 (Quality Pipeline)

**Round-trip translation is NOT a reliable quality signal.** This changes the quality pipeline design:
- Drop round-trip verification as a quality check
- Instead: validate that translated output has same block structure as input (same number of paragraphs, headings, lists)
- Validate that code blocks and LaTeX are byte-identical to source (were not modified by translation)
- Validate that translated text is non-empty and approximately similar in length to source (translations typically ±30% of source length)

## Acceptance

Spike is complete. We can describe:
- Which translation service to use (DeepL) and why
- How to integrate it (thin Go HTTP client, XML tag handling, ignore_tags for code/LaTeX)
- The translation strategy (full-document with XML structure preservation)
- Why round-trip validation doesn't work (and what to do instead for A4)
- Cost expectations (~$0.75/article on Pro, free tier for ~16 articles/month)
