package limiter

import "time"

type Bucket struct {
	level    float64
	lastLeak time.Time
}

type LeakyBucketLimiter struct {
	capacity int
	leakRate float64
	store    map[string]*Bucket
}

func NewLeakyBucketLimiter(capacity int, leakRate float64) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		capacity: capacity,
		leakRate: leakRate,
		store:    make(map[string]*Bucket),
	}
}

func (l *LeakyBucketLimiter) Allow(tenantKey string) (allowed bool, remaining float64, retryAfter time.Duration) {
	now := time.Now()

	if _, exists := l.store[tenantKey]; !exists {
		l.store[tenantKey] = &Bucket{level: 1, lastLeak: now}
		return true, float64(l.capacity) - 1, 0
	}

	bucketLeak(l, tenantKey, now)

	bucket := l.store[tenantKey]

	if bucket.level < float64(l.capacity) {
		bucket.level++
		return true, float64(l.capacity) - bucket.level, 0
	} else {
		retry := (bucket.level - float64(l.capacity) + 1) / l.leakRate
		return false, 0, time.Duration(retry * float64(time.Second))
	}
}

func bucketLeak(l *LeakyBucketLimiter, tenantKey string, now time.Time) {
	lastLeak := l.store[tenantKey].lastLeak

	elapsed := now.Sub(lastLeak).Seconds()

	leaked := elapsed * l.leakRate

	level := l.store[tenantKey].level - leaked

	if level < 0 {
		l.store[tenantKey].level = 0
	} else {
		l.store[tenantKey].level = level
	}

	l.store[tenantKey].lastLeak = now
}
