package limiter

import (
	"time"
)

type fixedWindow struct {
	windowSize time.Duration
	limit      int64
}

type windowCounter struct {
	windowId int64
	counter  int64
}

var (
	fw       = &fixedWindow{windowSize: 10 * time.Second, limit: 5}
	counters = map[string]*windowCounter{}
)

func FixedWindowLimiter(tenant string) (bool, error) {
	now := time.Now().Unix()

	windowId := now / int64(fw.windowSize.Seconds())

	if counters[tenant] == nil {
		counters[tenant] = &windowCounter{
			windowId: windowId,
			counter:  1,
		}
		return true, nil
	}

	userCounter := counters[tenant]

	if userCounter.windowId != windowId {
		userCounter.windowId = windowId
		userCounter.counter = 1
		return true, nil
	}

	userCounter.counter++

	if userCounter.counter > fw.limit {
		return false, nil
	} else {
		return true, nil
	}
}
