package redis

import (
	"context"
	"fmt"
	"ticketing/pkg/config"

	"github.com/redis/go-redis/v9"
)

func Connect(cfg config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0,
	})
}

func CheckHealth(rdb *redis.Client) error {
	return rdb.Ping(context.Background()).Err()
}
