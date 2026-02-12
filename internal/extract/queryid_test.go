package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractQueryID_MatchesTarget(t *testing.T) {
	js := `e.exports={queryId:"abc123",operationName:"TweetResultByRestId",operationType:"query"}`
	id, err := extractQueryID(js, "TweetResultByRestId")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "abc123" {
		t.Errorf("got %q, want %q", id, "abc123")
	}
}

func TestExtractQueryID_MultipleOperations(t *testing.T) {
	js := `e.exports={queryId:"aaa",operationName:"UserByScreenName",operationType:"query"};` +
		`e.exports={queryId:"bbb",operationName:"TweetResultByRestId",operationType:"query"};` +
		`e.exports={queryId:"ccc",operationName:"Followers",operationType:"query"}`
	id, err := extractQueryID(js, "TweetResultByRestId")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "bbb" {
		t.Errorf("got %q, want %q", id, "bbb")
	}
}

func TestExtractQueryID_NotFound(t *testing.T) {
	js := `e.exports={queryId:"aaa",operationName:"UserByScreenName",operationType:"query"}`
	_, err := extractQueryID(js, "TweetResultByRestId")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindMainBundleURL_ValidHTML(t *testing.T) {
	html := `<link rel="preload" href="https://abs.twimg.com/responsive-web/client-web/main.1fbef63a.js" as="script">` +
		`<script src="https://abs.twimg.com/responsive-web/client-web/main.1fbef63a.js"></script>`
	url, err := findMainBundleURL(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://abs.twimg.com/responsive-web/client-web/main.1fbef63a.js" {
		t.Errorf("got %q", url)
	}
}

func TestFindMainBundleURL_NoMatch(t *testing.T) {
	html := `<script src="https://example.com/other.js"></script>`
	_, err := findMainBundleURL(html)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadWriteCache(t *testing.T) {
	dir := t.TempDir()
	cacheDir = dir
	t.Cleanup(func() { cacheDir = "" })

	now := time.Now().Truncate(time.Second)
	writeCache(&queryIDCache{
		QueryID:   "test-id",
		Operation: "TweetResultByRestId",
		FetchedAt: now,
	})

	cached, err := readCache()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cached.QueryID != "test-id" {
		t.Errorf("QueryID = %q, want %q", cached.QueryID, "test-id")
	}
	if cached.Operation != "TweetResultByRestId" {
		t.Errorf("Operation = %q, want %q", cached.Operation, "TweetResultByRestId")
	}
	if !cached.FetchedAt.Equal(now) {
		t.Errorf("FetchedAt = %v, want %v", cached.FetchedAt, now)
	}
}

func TestReadCache_FileNotExists(t *testing.T) {
	dir := t.TempDir()
	cacheDir = dir
	t.Cleanup(func() { cacheDir = "" })

	cached, err := readCache()
	if err == nil && cached != nil {
		t.Fatal("expected error or nil cache for missing file")
	}
}

func TestResolveQueryID_UsesCache(t *testing.T) {
	dir := t.TempDir()
	cacheDir = dir
	t.Cleanup(func() { cacheDir = "" })

	// Write a fresh cache entry.
	data, _ := json.Marshal(queryIDCache{
		QueryID:   "cached-id",
		Operation: "TweetResultByRestId",
		FetchedAt: time.Now(),
	})
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "query-id.json"), data, 0644)

	id, err := ResolveQueryID(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cached-id" {
		t.Errorf("got %q, want %q", id, "cached-id")
	}
}

// failingBundleFetcher simulates a bundle fetch failure for testing fallback logic.
func failingBundleFetcher(_ context.Context) (string, error) {
	return "", fmt.Errorf("simulated bundle fetch failure")
}

func TestResolveQueryID_ExpiredCacheFallsToDefault(t *testing.T) {
	dir := t.TempDir()
	cacheDir = dir
	bundleFetcher = failingBundleFetcher
	t.Cleanup(func() { cacheDir = ""; bundleFetcher = fetchQueryIDFromBundle })

	// Write an expired cache entry.
	data, _ := json.Marshal(queryIDCache{
		QueryID:   "old-id",
		Operation: "TweetResultByRestId",
		FetchedAt: time.Now().Add(-25 * time.Hour),
	})
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "query-id.json"), data, 0644)

	// Bundle fetch fails, flag is empty → falls to hardcoded default.
	id, err := ResolveQueryID(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != defaultQueryID {
		t.Errorf("got %q, want default %q", id, defaultQueryID)
	}
}

func TestResolveQueryID_FlagOverride(t *testing.T) {
	dir := t.TempDir()
	cacheDir = dir
	bundleFetcher = failingBundleFetcher
	t.Cleanup(func() { cacheDir = ""; bundleFetcher = fetchQueryIDFromBundle })
	// No cache, bundle fetch fails → should use flag override.

	id, err := ResolveQueryID(context.Background(), "my-override")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "my-override" {
		t.Errorf("got %q, want %q", id, "my-override")
	}
}
