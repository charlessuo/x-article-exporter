package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/annismckenzie/x-article-exporter/internal/pipeline"
)

// RunFunc is the pipeline function signature. Defaults to pipeline.Run.
type RunFunc func(ctx context.Context, opts pipeline.Options) (*pipeline.Result, error)

// Manager coordinates job submission and execution with bounded concurrency.
type Manager struct {
	storage *Storage
	auth    AuthConfig
	runFn   RunFunc
	sem     chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// AuthConfig holds the X API credentials used for all pipeline runs.
type AuthConfig struct {
	AuthToken   string
	CT0         string
	OllamaModel string
}

// ManagerConfig configures the job manager.
type ManagerConfig struct {
	Storage        *Storage
	Auth           AuthConfig
	MaxConcurrent  int
	RunFn          RunFunc // optional, defaults to pipeline.Run
}

// NewManager creates a Manager with bounded concurrency.
func NewManager(cfg ManagerConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	runFn := cfg.RunFn
	if runFn == nil {
		runFn = pipeline.Run
	}
	return &Manager{
		storage: cfg.Storage,
		auth:    cfg.Auth,
		runFn:   runFn,
		sem:     make(chan struct{}, cfg.MaxConcurrent),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Submit creates a new job and starts processing it.
// Returns the job ID immediately.
func (m *Manager) Submit(url, translate string, darkMode bool) string {
	id := generateID()
	now := time.Now()
	job := &Job{
		ID:        id,
		URL:       url,
		Translate: translate,
		DarkMode:  darkMode,
		Status:    StatusProcessing,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.storage.Put(job)

	m.wg.Add(1)
	go m.process(id, url, translate, darkMode)
	return id
}

// Get returns a job by ID (copy from storage).
func (m *Manager) Get(id string) *Job {
	return m.storage.Get(id)
}

// Shutdown cancels in-flight jobs and waits for goroutines to exit.
func (m *Manager) Shutdown() {
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) process(id, url, translate string, darkMode bool) {
	defer m.wg.Done()

	// Acquire semaphore slot (bounded concurrency).
	select {
	case m.sem <- struct{}{}:
		defer func() { <-m.sem }()
	case <-m.ctx.Done():
		m.storage.Update(id, func(j *Job) {
			j.Status = StatusFailed
			j.Error = "cancelled"
			j.UpdatedAt = time.Now()
		})
		return
	}

	result, err := m.runFn(m.ctx, pipeline.Options{
		URL:         url,
		TranslateTo: translate,
		AuthToken:   m.auth.AuthToken,
		CT0:         m.auth.CT0,
		OllamaModel: m.auth.OllamaModel,
		DarkMode:    darkMode,
	})

	now := time.Now()
	if err != nil {
		m.storage.Update(id, func(j *Job) {
			j.Status = StatusFailed
			j.Error = err.Error()
			j.UpdatedAt = now
		})
		return
	}

	m.storage.Update(id, func(j *Job) {
		j.Status = StatusComplete
		j.Result = result
		j.UpdatedAt = now
	})
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
