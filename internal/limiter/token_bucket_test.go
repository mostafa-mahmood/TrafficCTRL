package limiter

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestTokenBucketAllow(t *testing.T) {
	bucket := NewTokenBucketStore(10, 1.0/2.0)

	for i := 0; i < 50; i++ {
		allowed, remaining, tryAfter := bucket.Allow("khedr")

		usableTokens := int(math.Floor(remaining))

		fmt.Printf("allowed: %v, remaining: %d, tryAfter: %.0fs\n", allowed, usableTokens, tryAfter.Seconds())

		time.Sleep(100 * time.Millisecond)
	}
}
