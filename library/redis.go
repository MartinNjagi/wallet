package library

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func GetRedisKey(conn *redis.Client, key string) (string, error) {
	ctx := context.Background()
	data, err := conn.Get(ctx, key).Result()
	if err != nil {
		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}
	return data, nil
}

func SetRedisKey(conn *redis.Client, key string, value string) error {
	ctx := context.Background()
	_, err := conn.Set(ctx, key, value, 0).Result()
	if err != nil {
		v := value
		if len(v) > 15 {
			v = v[:12] + "..."
		}
		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}
	return nil
}

func SetRedisKeyWithExpiry(conn *redis.Client, key string, value string, seconds int) error {
	ctx := context.Background()
	_, err := conn.Set(ctx, key, value, time.Second*time.Duration(seconds)).Result()
	if err != nil {
		v := value
		if len(v) > 15 {
			v = v[:12] + "..."
		}
		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}
	return nil
}

func DeleteRedisKey(conn *redis.Client, key string) error {
	ctx := context.Background()
	_, err := conn.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("error deleting key %s: %v", key, err)
	}
	return nil
}

func GetAllKeys(conn *redis.Client, pattern string) (error, map[string]string) {
	ctx := context.Background()
	results := make(map[string]string)

	keys, err := conn.Keys(ctx, pattern).Result()
	if err != nil {
		return err, nil
	}

	for _, k := range keys {
		val, err := conn.Get(ctx, k).Result()
		if err != nil {
			continue
		}
		results[k] = val
	}
	return nil, results
}

func IncRedisKey(conn *redis.Client, key string) (int64, error) {
	ctx := context.Background()
	data, err := conn.Incr(ctx, key).Result()
	if err != nil {
		return data, fmt.Errorf("error incrementing key %s: %v", key, err)
	}
	return data, nil
}
