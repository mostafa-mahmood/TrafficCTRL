package db

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func InitRedis(addr, password string, db int) {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func Ping(ctx context.Context) error {
	_, err := Rdb.Ping(ctx).Result()
	return err
}

func Close() error {
	err := Rdb.Close()
	return err
}
