package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const cacheTTL = 24 * time.Hour

// cacheDir overrides the cache directory in tests. Empty means use default.
var cacheDir string

type queryIDCache struct {
	QueryID   string    `json:"query_id"`
	Operation string    `json:"operation"`
	FetchedAt time.Time `json:"fetched_at"`
}

// bundleFetcher is the function used to extract the query ID from X's JS bundle.
// Overridden in tests to avoid hitting the network.
var bundleFetcher = fetchQueryIDFromBundle

// ResolveQueryID returns the GraphQL query ID for TweetResultByRestId.
// Resolution order: cache (24h TTL) → JS bundle extraction → flagOverride → hardcoded fallback.
func ResolveQueryID(ctx context.Context, flagOverride string) (string, error) {
	// 1. Try cache.
	if cached, err := readCache(); err == nil && cached != nil && time.Since(cached.FetchedAt) < cacheTTL {
		log.Printf("Using cached query ID: %s", cached.QueryID)
		return cached.QueryID, nil
	}

	// 2. Try bundle extraction.
	log.Println("Fetching query ID from X...")
	if id, err := bundleFetcher(ctx); err == nil {
		log.Printf("Resolved query ID: %s", id)
		writeCache(&queryIDCache{QueryID: id, Operation: operationName, FetchedAt: time.Now()})
		return id, nil
	} else {
		log.Printf("Warning: could not extract query ID from bundle: %v", err)
	}

	// 3. Try flag override.
	if flagOverride != "" {
		log.Printf("Using --query-id override: %s", flagOverride)
		return flagOverride, nil
	}

	// 4. Hardcoded fallback.
	log.Printf("Using fallback query ID: %s", defaultQueryID)
	return defaultQueryID, nil
}

func fetchQueryIDFromBundle(ctx context.Context) (string, error) {
	htmlBody, err := httpGet(ctx, "https://x.com")
	if err != nil {
		return "", fmt.Errorf("fetching x.com: %w", err)
	}

	bundleURL, err := findMainBundleURL(htmlBody)
	if err != nil {
		return "", fmt.Errorf("finding main bundle URL: %w", err)
	}

	jsBody, err := httpGet(ctx, bundleURL)
	if err != nil {
		return "", fmt.Errorf("fetching JS bundle: %w", err)
	}

	return extractQueryID(jsBody, operationName)
}

// mainBundlePattern matches the main.*.js URL in X's HTML.
var mainBundlePattern = regexp.MustCompile(`(?:src|href)="(https://abs\.twimg\.com/responsive-web/client-web/main\.[a-f0-9]+\.js)"`)

func findMainBundleURL(htmlBody string) (string, error) {
	matches := mainBundlePattern.FindStringSubmatch(htmlBody)
	if matches == nil {
		return "", fmt.Errorf("no main bundle URL found in HTML")
	}
	return matches[1], nil
}

// queryIDPattern matches queryId/operationName pairs in minified JS.
var queryIDPattern = regexp.MustCompile(`queryId:"([^"]+)",operationName:"([^"]+)"`)

func extractQueryID(jsBody string, targetOperation string) (string, error) {
	matches := queryIDPattern.FindAllStringSubmatch(jsBody, -1)
	for _, m := range matches {
		if m[2] == targetOperation {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("query ID for %s not found in JS bundle (%d operations found)", targetOperation, len(matches))
}

func getCachePath() (string, error) {
	dir := cacheDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".cache", "x-article-exporter")
	}
	return filepath.Join(dir, "query-id.json"), nil
}

func readCache() (*queryIDCache, error) {
	path, err := getCachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c queryIDCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func writeCache(c *queryIDCache) {
	path, err := getCachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

func httpGet(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:146.0) Gecko/20100101 Firefox/146.0")
	// Accept both compressed and plain responses.
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Follow redirects automatically (http.DefaultClient handles this).
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}
