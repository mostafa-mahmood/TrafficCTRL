package limiter

import "time"

type TokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type TokenBucketStore struct {
	capacity   int
	refillRate float64
	store      map[string]*TokenBucket
}

func NewTokenBucketStore(capacity int, refillRate float64) *TokenBucketStore {
	return &TokenBucketStore{
		capacity:   capacity,
		refillRate: refillRate,
		store:      make(map[string]*TokenBucket),
	}
}

func (t *TokenBucketStore) Allow(tenantKey string) (allowed bool, remaining float64, tryAfter time.Duration) {
	now := time.Now()

	if _, exists := t.store[tenantKey]; !exists {
		t.store[tenantKey] = &TokenBucket{
			tokens:     float64(t.capacity),
			lastRefill: now,
		}
	}

	refillBucket(t, tenantKey, now)
	bucket := t.store[tenantKey]

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, bucket.tokens, 0
	}

	needed := 1 - bucket.tokens
	waitSeconds := needed / t.refillRate
	tryAfter = time.Duration(waitSeconds * float64(time.Second))

	return false, bucket.tokens, tryAfter
}

func refillBucket(t *TokenBucketStore, tenantKey string, now time.Time) {
	bucket := t.store[tenantKey]
	elapsed := now.Sub(bucket.lastRefill).Seconds()

	bucket.tokens += elapsed * t.refillRate
	if bucket.tokens > float64(t.capacity) {
		bucket.tokens = float64(t.capacity)
	}
	bucket.lastRefill = now
}
