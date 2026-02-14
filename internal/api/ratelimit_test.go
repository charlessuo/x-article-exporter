package api

import "testing"

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := newRateLimiter(5)
	for i := 0; i < 5; i++ {
		if !rl.allow("key1") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := newRateLimiter(3)
	for i := 0; i < 3; i++ {
		rl.allow("key1")
	}
	if rl.allow("key1") {
		t.Error("expected request to be blocked after exceeding limit")
	}
}

func TestRateLimiter_KeysAreIndependent(t *testing.T) {
	rl := newRateLimiter(1)
	if !rl.allow("key1") {
		t.Error("key1 first request should be allowed")
	}
	if !rl.allow("key2") {
		t.Error("key2 first request should be allowed (independent)")
	}
	if rl.allow("key1") {
		t.Error("key1 second request should be blocked")
	}
}

func TestRateLimiter_PerKeyOverride(t *testing.T) {
	rl := newRateLimiter(1)
	rl.setKeyLimit("vip", 3)

	// Default key gets 1 request.
	if !rl.allow("normal") {
		t.Error("normal first request should be allowed")
	}
	if rl.allow("normal") {
		t.Error("normal second request should be blocked")
	}

	// VIP key gets 3 requests.
	for i := 0; i < 3; i++ {
		if !rl.allow("vip") {
			t.Fatalf("vip request %d should be allowed", i+1)
		}
	}
	if rl.allow("vip") {
		t.Error("vip request after limit should be blocked")
	}
}
