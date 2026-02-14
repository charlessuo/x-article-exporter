# API Reference

Start the server:

```sh
make serve
# or: go run main.go --serve
```

Requires a config file at `~/.config/x-article-exporter/config.yaml` with `auth_token`, `ct0`, and at least one API key. See `config.example.yaml`.

## Authentication

All endpoints require a Bearer token matching a configured API key:

```
Authorization: Bearer <api-key>
```

## Endpoints

### POST /export

Submit an article URL for export. Returns immediately with a job ID.

**Request:**

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

**Errors:** 400 (bad URL/JSON), 401 (auth), 429 (rate limit).

### GET /export/{id}

Check the status of an export job.

**Response (200 OK):**

Processing:

```json
{
  "id": "a1b2c3d4...",
  "status": "processing"
}
```

Complete:

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

Failed:

```json
{
  "id": "a1b2c3d4...",
  "status": "failed",
  "error": "error description"
}
```

### GET /export/{id}/pdf

Download the completed PDF.

- **200 OK** with `Content-Type: application/pdf` when complete.
- **202 Accepted** with JSON status when still processing.
- **404 Not Found** if job doesn't exist.

## Rate Limiting

POST /export is rate-limited per API key using a token bucket algorithm. Default: 20 requests/hour. Per-key overrides can be set in the config file. Exceeding the limit returns 429.

## Example Workflow

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
