package broker

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func NewRedis(ctx context.Context, url string) (*Redis, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Redis{client: client}, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Push(ctx context.Context, queue string, data []byte) error {
	if err := r.client.LPush(ctx, queue, data).Err(); err != nil {
		return fmt.Errorf("lpush %s: %w", queue, err)
	}
	return nil
}

func (r *Redis) BlockPop(ctx context.Context, queue string) ([]byte, error) {
	result, err := r.client.BRPop(ctx, 0, queue).Result()
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil
		}
		return nil, fmt.Errorf("brpop %s: %w", queue, err)
	}
	return []byte(result[1]), nil
}

func (r *Redis) QueueLen(ctx context.Context, queue string) (int64, error) {
	n, err := r.client.LLen(ctx, queue).Result()
	if err != nil {
		return 0, fmt.Errorf("llen %s: %w", queue, err)
	}
	return n, nil
}

func (r *Redis) SetWithExpiry(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get %s: %w", key, err)
	}

	return val, nil
}

func (r *Redis) ScanKeys(ctx context.Context, pattern string) ([]string, error) {
	var allKeys []string
	var cursor uint64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", pattern, err)
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return allKeys, nil
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del %s: %w", key, err)
	}
	return nil
}
