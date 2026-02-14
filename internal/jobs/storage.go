package jobs

import (
	"sync"
	"time"
)

// Storage provides thread-safe in-memory storage for jobs with TTL cleanup.
type Storage struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	ttl  time.Duration

	done chan struct{}
	wg   sync.WaitGroup
}

// NewStorage creates a Storage that automatically removes jobs older than ttl.
func NewStorage(ttl time.Duration) *Storage {
	s := &Storage{
		jobs: make(map[string]*Job),
		ttl:  ttl,
		done: make(chan struct{}),
	}
	s.wg.Add(1)
	go s.cleanup()
	return s
}

// Put stores a job. It replaces any existing job with the same ID.
func (s *Storage) Put(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Get returns a copy of the job with the given ID, or nil if not found.
// Returning a copy prevents data races between readers and the manager's updates.
func (s *Storage) Get(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil
	}
	cp := *j
	if j.Result != nil {
		rCopy := *j.Result
		cp.Result = &rCopy
	}
	return &cp
}

// Update modifies the job in place under a write lock.
func (s *Storage) Update(id string, fn func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		fn(j)
	}
}

// Close stops the cleanup goroutine and waits for it to exit.
func (s *Storage) Close() {
	close(s.done)
	s.wg.Wait()
}

func (s *Storage) cleanup() {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.removeExpired()
		}
	}
}

func (s *Storage) removeExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-s.ttl)
	for id, j := range s.jobs {
		if j.UpdatedAt.Before(cutoff) {
			delete(s.jobs, id)
		}
	}
}
