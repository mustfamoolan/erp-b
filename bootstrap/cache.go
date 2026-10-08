package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"m3aml-erp/config"
	"github.com/fatih/color"
	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client
var Ctx = context.Background()

func InitializeCache() {
	cacheConfig := config.Global.Cache

	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cacheConfig.Host, cacheConfig.Port),
		Password: cacheConfig.Password,
		DB:       0, // use default DB
	})

	// Test connection with short timeout so it doesn't hang startup
	ctxTimeout, cancel := context.WithTimeout(Ctx, 2*time.Second)
	defer cancel()

	_, err := Redis.Ping(ctxTimeout).Result()
	if err != nil {
		fmt.Printf("%s Redis connection failed: %v (Cache falling back to direct DB queries)\n", color.YellowString("⚠️"), err)
		Redis = nil
		return
	}

	fmt.Printf("%s Redis connected successfully\n", color.CyanString("⚙️"))
}

// IsRedisAvailable checks if the Redis client is initialized and reachable.
func IsRedisAvailable() bool {
	return Redis != nil
}

// Cache Wrapper (Laravel-style)
func CacheGet(key string) (string, error) {
	if !IsRedisAvailable() {
		return "", redis.Nil
	}
	return Redis.Get(Ctx, key).Result()
}

func CacheSet(key string, value interface{}, expiration time.Duration) error {
	if !IsRedisAvailable() {
		return nil
	}
	return Redis.Set(Ctx, key, value, expiration).Err()
}

func CacheForget(key string) error {
	if !IsRedisAvailable() {
		return nil
	}
	return Redis.Del(Ctx, key).Err()
}

// CacheGetJSON retrieves a JSON-serialized object from Redis and unmarshals it into dest.
// Returns true on cache hit, false on miss or error.
func CacheGetJSON[T any](key string, dest *T) bool {
	if !IsRedisAvailable() {
		return false
	}
	val, err := Redis.Get(Ctx, key).Bytes()
	if err != nil {
		return false
	}
	if err := json.Unmarshal(val, dest); err != nil {
		return false
	}
	return true
}

// CacheSetJSON serializes value into JSON and stores it in Redis with the given expiration.
func CacheSetJSON(key string, value interface{}, expiration time.Duration) error {
	if !IsRedisAvailable() {
		return nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Redis.Set(Ctx, key, bytes, expiration).Err()
}

// CacheRemember implements the Cache-Aside pattern with generics:
// 1. Checks Redis for key. If hit, deserializes and returns.
// 2. If miss, invokes fallback function.
// 3. Caches result in Redis and returns.
func CacheRemember[T any](key string, expiration time.Duration, fallback func() (T, error)) (T, error) {
	var result T
	if CacheGetJSON(key, &result) {
		return result, nil
	}

	// Cache miss: execute fallback
	val, err := fallback()
	if err != nil {
		return val, err
	}

	// Cache valid result asynchronously or synchronously
	_ = CacheSetJSON(key, val, expiration)
	return val, nil
}

// CacheDeletePattern deletes all keys matching the glob pattern using SCAN (non-blocking for production).
func CacheDeletePattern(pattern string) error {
	if !IsRedisAvailable() {
		return nil
	}

	iter := Redis.Scan(Ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(Ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 200 {
			_ = Redis.Del(Ctx, keys...).Err()
			keys = nil
		}
	}
	if len(keys) > 0 {
		_ = Redis.Del(Ctx, keys...).Err()
	}
	return iter.Err()
}

// InvalidateMasterDataCache clears all master data keys
func InvalidateMasterDataCache() {
	_ = CacheDeletePattern("md:*")
}

// InvalidateOrgCache clears scopes, factories, areas, and users caches
func InvalidateOrgCache() {
	_ = CacheDeletePattern("org:*")
}

// InvalidateRequestsCache clears financial requests caches
func InvalidateRequestsCache(scopeIDs ...string) {
	_ = CacheDeletePattern("requests:*")
}
