package jobs

import (
	"testing"
	"time"
)

func TestStorage_PutAndGet(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	now := time.Now()
	s.Put(&Job{ID: "a", URL: "https://x.com/i/article/1", Status: StatusProcessing, CreatedAt: now, UpdatedAt: now})

	j := s.Get("a")
	if j == nil {
		t.Fatal("expected job, got nil")
	}
	if j.URL != "https://x.com/i/article/1" {
		t.Errorf("URL = %q", j.URL)
	}
}

func TestStorage_GetReturnsNilForMissing(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	if j := s.Get("nonexistent"); j != nil {
		t.Errorf("expected nil, got %+v", j)
	}
}

func TestStorage_GetReturnsCopy(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	now := time.Now()
	s.Put(&Job{ID: "a", Status: StatusProcessing, CreatedAt: now, UpdatedAt: now})

	j := s.Get("a")
	j.Status = StatusComplete // mutate the copy

	original := s.Get("a")
	if original.Status != StatusProcessing {
		t.Errorf("original was mutated: Status = %q", original.Status)
	}
}

func TestStorage_Update(t *testing.T) {
	s := NewStorage(time.Hour)
	defer s.Close()

	now := time.Now()
	s.Put(&Job{ID: "a", Status: StatusProcessing, CreatedAt: now, UpdatedAt: now})

	s.Update("a", func(j *Job) {
		j.Status = StatusComplete
		j.UpdatedAt = time.Now()
	})

	j := s.Get("a")
	if j.Status != StatusComplete {
		t.Errorf("Status = %q, want complete", j.Status)
	}
}

func TestStorage_RemoveExpired(t *testing.T) {
	s := NewStorage(time.Millisecond)
	defer s.Close()

	past := time.Now().Add(-time.Second)
	s.Put(&Job{ID: "old", Status: StatusComplete, CreatedAt: past, UpdatedAt: past})
	s.Put(&Job{ID: "new", Status: StatusProcessing, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	s.removeExpired()

	if j := s.Get("old"); j != nil {
		t.Error("expected old job to be removed")
	}
	if j := s.Get("new"); j == nil {
		t.Error("expected new job to still exist")
	}
}
