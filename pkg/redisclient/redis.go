package redisclient

import (
	"community-backend/config"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type Client struct {
	*redis.Client
}

func ConnectRedis(cfg config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Client{rdb}, nil
}

func (c *Client) HealthCheck(ctx context.Context) (float64, error) {
	start := time.Now()
	if err := c.Ping(ctx).Err(); err != nil {
		return 0, err
	}
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	return latency, nil
}
