package middleware_test

import (
	"testing"
	"time"

	"singlepage/internal/middleware"
)

func TestClientRateLimiterAllowUsesIndependentClientBuckets(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewClientRateLimiter(1, 1)
	now := time.Unix(1_700_000_000, 0)

	if !limiter.Allow("client-a", now) {
		t.Fatal("first request for client-a was rejected")
	}

	if !limiter.Allow("client-b", now) {
		t.Fatal("first request for client-b was rejected")
	}
}

func TestClientRateLimiterAllowRejectsExhaustedBucket(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewClientRateLimiter(1, 1)
	now := time.Unix(1_700_000_000, 0)

	if !limiter.Allow("client", now) {
		t.Fatal("first request was rejected")
	}

	if limiter.Allow("client", now) {
		t.Fatal("second request at the same instant was allowed")
	}
}

func TestClientRateLimiterAllowRefillsBucketOverTime(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewClientRateLimiter(1, 1)
	now := time.Unix(1_700_000_000, 0)

	if !limiter.Allow("client", now) {
		t.Fatal("first request was rejected")
	}

	if !limiter.Allow("client", now.Add(time.Second)) {
		t.Fatal("request after refill interval was rejected")
	}
}
