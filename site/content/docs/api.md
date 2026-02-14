---
title: HTTP API
weight: 3
---

Start the server with `--serve`:

```sh
x-article-exporter --serve
```

Requires a [config file](configuration) with `auth_token`, `ct0`, and at least one API key in the `server` section.

## Authentication

All endpoints require a Bearer token matching a configured API key:

```
Authorization: Bearer <api-key>
```

## Endpoints

### POST /export

Submit an article URL for export. Returns immediately with a job ID.

**Request body:**

```json
{
  "url": "https://x.com/user/article/1234567890",
  "translate": "de",
  "dark_mode": true
}
```

Only `url` is required. `translate` and `dark_mode` are optional.

**Response (202 Accepted):**

```json
{
  "id": "a1b2c3d4e5f6...",
  "status": "processing"
}
```

**Errors:**

| Status | Meaning |
|--------|---------|
| 400 | Invalid URL or malformed JSON |
| 401 | Missing or invalid auth token |
| 429 | Rate limit exceeded |

### GET /export/{id}

Check the status of an export job.

**Processing (200 OK):**

```json
{
  "id": "a1b2c3d4...",
  "status": "processing"
}
```

**Complete (200 OK):**

```json
{
  "id": "a1b2c3d4...",
  "status": "complete",
  "title": "Article Title",
  "author": "Author Name (@handle)",
  "page_count": 5,
  "word_count": 2400,
  "links": {
    "pdf": "/export/a1b2c3d4.../pdf"
  }
}
```

**Failed (200 OK):**

```json
{
  "id": "a1b2c3d4...",
  "status": "failed",
  "error": "error description"
}
```

### GET /export/{id}/pdf

Download the completed PDF.

| Status | Response |
|--------|----------|
| 200 OK | PDF file (`Content-Type: application/pdf`) |
| 202 Accepted | JSON status (still processing) |
| 404 Not Found | Job does not exist |

## Rate limiting

`POST /export` is rate-limited per API key using a token bucket algorithm. The default is 20 requests per hour. Per-key overrides can be set in the [config file](configuration). Exceeding the limit returns `429 Too Many Requests`.

## Example workflow

```sh
KEY="your-api-key"
URL="https://x.com/user/article/1234567890"

# Submit
JOB_ID=$(curl -s -X POST \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"$URL\"}" \
  http://localhost:8080/export | jq -r .id)

echo "Job: $JOB_ID"

# Poll status
curl -s -H "Authorization: Bearer $KEY" \
  http://localhost:8080/export/$JOB_ID | jq .

# Download PDF
curl -s -H "Authorization: Bearer $KEY" \
  http://localhost:8080/export/$JOB_ID/pdf -o article.pdf
```

## Server configuration

See the [configuration reference](configuration) for all `server:` options including port, host, concurrency limits, job TTL, and per-key rate limits.
