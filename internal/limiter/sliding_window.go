package limiter

import "time"

type SlidingWindow struct {
	limit      int
	windowSize time.Duration
	store      map[string][]time.Time
}

func NewSlidingWindow(limit int, windowSize time.Duration) *SlidingWindow {
	return &SlidingWindow{
		limit:      limit,
		windowSize: windowSize,
		store:      make(map[string][]time.Time),
	}
}

func pruneExpired(timestamps []time.Time, windowStart time.Time) []time.Time {
	cut := 0
	for cut < len(timestamps) && timestamps[cut].Before(windowStart) {
		cut++
	}
	return timestamps[cut:]
}

func (s *SlidingWindow) Allow(tenantKey string) (allowed bool, remaining int, retryAfter time.Duration) {
	now := time.Now()
	windowStart := now.Add(-s.windowSize)

	if _, exists := s.store[tenantKey]; !exists {
		s.store[tenantKey] = []time.Time{now}
		return true, s.limit - 1, 0
	}

	timestamps := pruneExpired(s.store[tenantKey], windowStart)

	if len(timestamps) >= s.limit {
		return false, 0, timestamps[0].Add(s.windowSize).Sub(now)
	}

	timestamps = append(timestamps, now)
	s.store[tenantKey] = timestamps

	remaining = s.limit - len(timestamps)

	return true, remaining, 0
}
