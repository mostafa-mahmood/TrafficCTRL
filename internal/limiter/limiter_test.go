package limiter

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func setupTestRateLimiter(t *testing.T) (*RateLimiter, *miniredis.Miniredis) {

	mr, err := miniredis.Run()
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	rl := &RateLimiter{
		redisClient: rdb,
	}

	return rl, mr
}
