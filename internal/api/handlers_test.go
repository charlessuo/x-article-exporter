package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/config"
	"github.com/annismckenzie/x-article-exporter/internal/jobs"
	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
)

func testServer(t *testing.T, runFn jobs.RunFunc) (*Server, *jobs.Manager, *jobs.Storage) {
	t.Helper()
	storage := jobs.NewStorage(time.Hour)
	t.Cleanup(storage.Close)

	mgr := jobs.NewManager(jobs.ManagerConfig{
		Storage:       storage,
		MaxConcurrent: 2,
		RunFn:         runFn,
	})
	t.Cleanup(mgr.Shutdown)

	cfg := &config.ServerConfig{
		DefaultRateLimitPerHour: 100,
		APIKeys: []config.APIKey{
			{Key: "test-key", Name: "test"},
		},
	}
	srv := NewServer(cfg, mgr)
	return srv, mgr, storage
}

func doRequest(t *testing.T, handler http.Handler, method, path, body, authKey string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if authKey != "" {
		r.Header.Set("Authorization", "Bearer "+authKey)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestHandleExport_Success(t *testing.T) {
	srv, _, _ := testServer(t, func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		return &pipeline.Result{Title: "Test", PDFBytes: []byte("pdf"), ValidationOK: true}, nil
	})

	w := doRequest(t, srv.Handler(), "POST", "/export",
		`{"url":"https://x.com/i/article/123"}`, "test-key")

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202. body: %s", w.Code, w.Body.String())
	}

	var resp exportResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == "" {
		t.Error("expected non-empty job ID")
	}
	if resp.Status != "processing" {
		t.Errorf("status = %q, want processing", resp.Status)
	}
}

func TestHandleExport_MissingURL(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "POST", "/export", `{}`, "test-key")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleExport_InvalidURL(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "POST", "/export",
		`{"url":"https://google.com"}`, "test-key")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleExport_BadJSON(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "POST", "/export", `not json`, "test-key")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestHandleExport_MissingAuth(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "POST", "/export",
		`{"url":"https://x.com/i/article/123"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestHandleExport_InvalidAuth(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "POST", "/export",
		`{"url":"https://x.com/i/article/123"}`, "wrong-key")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestHandleExportStatus_NotFound(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "GET", "/export/nonexistent", "", "test-key")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleExportStatus_Complete(t *testing.T) {
	done := make(chan struct{})
	srv, mgr, _ := testServer(t, func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		<-done
		return &pipeline.Result{Title: "My Article", Author: "Author", PageCount: 3, WordCount: 500, PDFBytes: []byte("pdf"), ValidationOK: true}, nil
	})

	id := mgr.Submit("https://x.com/i/article/123", "", false)
	close(done)

	// Wait for completion.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		j := mgr.Get(id)
		if j != nil && j.Status == jobs.StatusComplete {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	w := doRequest(t, srv.Handler(), "GET", "/export/"+id, "", "test-key")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", w.Code, w.Body.String())
	}

	var resp statusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Title != "My Article" {
		t.Errorf("title = %q, want My Article", resp.Title)
	}
	if resp.Links["pdf"] == "" {
		t.Error("expected pdf link")
	}
}

func TestHandleExportPDF_NotFound(t *testing.T) {
	srv, _, _ := testServer(t, nil)

	w := doRequest(t, srv.Handler(), "GET", "/export/nonexistent/pdf", "", "test-key")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleExportPDF_Processing(t *testing.T) {
	block := make(chan struct{})
	srv, mgr, _ := testServer(t, func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		<-block
		return &pipeline.Result{}, nil
	})
	defer close(block)

	id := mgr.Submit("https://x.com/i/article/123", "", false)
	// Give goroutine time to start.
	time.Sleep(20 * time.Millisecond)

	w := doRequest(t, srv.Handler(), "GET", "/export/"+id+"/pdf", "", "test-key")
	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202. body: %s", w.Code, w.Body.String())
	}
}

func TestHandleExportPDF_Success(t *testing.T) {
	done := make(chan struct{})
	srv, mgr, _ := testServer(t, func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		<-done
		return &pipeline.Result{Title: "Test PDF", PDFBytes: []byte("fakepdf"), ValidationOK: true}, nil
	})

	id := mgr.Submit("https://x.com/i/article/123", "", false)
	close(done)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		j := mgr.Get(id)
		if j != nil && j.Status == jobs.StatusComplete {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	w := doRequest(t, srv.Handler(), "GET", "/export/"+id+"/pdf", "", "test-key")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
	if w.Body.String() != "fakepdf" {
		t.Errorf("body = %q, want fakepdf", w.Body.String())
	}
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "Test PDF.pdf") {
		t.Errorf("Content-Disposition = %q, expected Test PDF.pdf", cd)
	}
}

func TestHandleExport_RateLimited(t *testing.T) {
	cfg := &config.ServerConfig{
		DefaultRateLimitPerHour: 1,
		APIKeys: []config.APIKey{
			{Key: "test-key", Name: "test"},
		},
	}
	storage := jobs.NewStorage(time.Hour)
	t.Cleanup(storage.Close)
	mgr := jobs.NewManager(jobs.ManagerConfig{
		Storage:       storage,
		MaxConcurrent: 2,
		RunFn: func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
			return &pipeline.Result{}, nil
		},
	})
	t.Cleanup(mgr.Shutdown)
	srv := NewServer(cfg, mgr)
	handler := srv.Handler()

	// First request should succeed.
	w := doRequest(t, handler, "POST", "/export",
		`{"url":"https://x.com/i/article/123"}`, "test-key")
	if w.Code != http.StatusAccepted {
		t.Fatalf("first request: status = %d, want 202", w.Code)
	}

	// Second request should be rate limited.
	w = doRequest(t, handler, "POST", "/export",
		`{"url":"https://x.com/i/article/123"}`, "test-key")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: status = %d, want 429", w.Code)
	}
}
