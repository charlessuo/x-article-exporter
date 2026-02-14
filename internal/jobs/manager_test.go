package jobs

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
)

func mockRunFunc(result *pipeline.Result, err error) RunFunc {
	return func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
		return result, err
	}
}

func TestManager_SubmitSuccess(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	m := NewManager(ManagerConfig{
		Storage:       s,
		MaxConcurrent: 2,
		RunFn: mockRunFunc(&pipeline.Result{
			Title:        "Test",
			PDFBytes:     []byte("pdf"),
			ValidationOK: true,
		}, nil),
	})
	defer m.Shutdown()

	id := m.Submit("https://x.com/i/article/1", "", false)
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	// Wait for processing to complete.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		j := m.Get(id)
		if j != nil && j.Status == StatusComplete {
			if j.Result.Title != "Test" {
				t.Errorf("Title = %q, want Test", j.Result.Title)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not complete within deadline")
}

func TestManager_SubmitFailure(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	m := NewManager(ManagerConfig{
		Storage:       s,
		MaxConcurrent: 2,
		RunFn:         mockRunFunc(nil, fmt.Errorf("boom")),
	})
	defer m.Shutdown()

	id := m.Submit("https://x.com/i/article/1", "", false)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		j := m.Get(id)
		if j != nil && j.Status == StatusFailed {
			if j.Error != "boom" {
				t.Errorf("Error = %q, want boom", j.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("job did not fail within deadline")
}

func TestManager_BoundedConcurrency(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	var concurrent atomic.Int32
	var maxSeen atomic.Int32
	const maxConcurrent = 2

	m := NewManager(ManagerConfig{
		Storage:       s,
		MaxConcurrent: maxConcurrent,
		RunFn: func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
			n := concurrent.Add(1)
			for {
				old := maxSeen.Load()
				if n <= old || maxSeen.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
			return &pipeline.Result{}, nil
		},
	})
	defer m.Shutdown()

	// Submit more jobs than the concurrency limit.
	for i := 0; i < 6; i++ {
		m.Submit("https://x.com/i/article/1", "", false)
	}

	// Wait for all to complete.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		allDone := true
		for i := 0; i < 6; i++ {
			// Jobs are submitted with unique IDs, but we just need all goroutines to finish.
		}
		_ = allDone
		if maxSeen.Load() > 0 {
			// Give a bit more time for all to run.
			time.Sleep(200 * time.Millisecond)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	m.Shutdown()

	if got := maxSeen.Load(); got > int32(maxConcurrent) {
		t.Errorf("max concurrent = %d, want <= %d", got, maxConcurrent)
	}
}

func TestManager_ShutdownCancelsJobs(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	var cancelled atomic.Bool
	m := NewManager(ManagerConfig{
		Storage:       s,
		MaxConcurrent: 1,
		RunFn: func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
			<-ctx.Done()
			cancelled.Store(true)
			return nil, ctx.Err()
		},
	})

	m.Submit("https://x.com/i/article/1", "", false)
	// Give the goroutine time to start and block on ctx.Done().
	time.Sleep(50 * time.Millisecond)

	m.Shutdown()

	if !cancelled.Load() {
		t.Error("expected RunFunc to observe context cancellation")
	}
}

func TestManager_PassesOptions(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	var gotOpts pipeline.Options
	m := NewManager(ManagerConfig{
		Storage:       s,
		Auth:          AuthConfig{AuthToken: "tok", CT0: "ct", OllamaModel: "model"},
		MaxConcurrent: 1,
		RunFn: func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error) {
			gotOpts = opts
			return &pipeline.Result{}, nil
		},
	})
	defer m.Shutdown()

	m.Submit("https://x.com/i/article/1", "de", true)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gotOpts.URL != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if gotOpts.URL != "https://x.com/i/article/1" {
		t.Errorf("URL = %q", gotOpts.URL)
	}
	if gotOpts.TranslateTo != "de" {
		t.Errorf("TranslateTo = %q", gotOpts.TranslateTo)
	}
	if gotOpts.DarkMode != true {
		t.Error("DarkMode = false")
	}
	if gotOpts.AuthToken != "tok" {
		t.Errorf("AuthToken = %q", gotOpts.AuthToken)
	}
	if gotOpts.CT0 != "ct" {
		t.Errorf("CT0 = %q", gotOpts.CT0)
	}
	if gotOpts.OllamaModel != "model" {
		t.Errorf("OllamaModel = %q", gotOpts.OllamaModel)
	}
}
