package services

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"sync"
	"time"
)

// RedisManager handles Redis connections for a multi-core environment
type RedisManager struct {
	pools map[int]*redis.Client
	mu    sync.Mutex
}

// NewRedisManager creates Redis connection pools for each core
func NewRedisManager(numCores int) *RedisManager {
	manager := &RedisManager{
		pools: make(map[int]*redis.Client),
	}
	for i := 0; i < numCores; i++ {
		client := redis.NewClient(&redis.Options{
			Addr:         os.Getenv("REDIS_ADDR"),
			PoolSize:     10,
			MinIdleConns: 5,
		})
		manager.pools[i] = client
	}
	return manager
}

// GetClientForCore returns a Redis client for a specific core
func (rm *RedisManager) GetClientForCore(core int) *redis.Client {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.pools[core%len(rm.pools)]
}

// StoreRequest stores a request number in Redis for a specific core
func (rm *RedisManager) StoreRequest(core int, requestID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redisClient := rm.GetClientForCore(core)
	key := fmt.Sprintf("request:core:%d", core)

	return redisClient.RPush(ctx, key, requestID).Err()
}

// RetrieveRequests retrieves stored requests for all cores
func (rm *RedisManager) RetrieveRequests(numCores int) (map[int][]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := make(map[int][]int)

	for core := 0; core < numCores; core++ {
		redisClient := rm.GetClientForCore(core)
		key := fmt.Sprintf("request:core:%d", core)

		stored, err := redisClient.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, err
		}

		coreRequests := make([]int, len(stored))
		for i, strVal := range stored {
			fmt.Sscanf(strVal, "%d", &coreRequests[i])
		}

		results[core] = coreRequests
	}

	return results, nil
}

// Close closes all connection pools
func (pm *RedisManager) Close() {
	for _, pool := range pm.pools {
		pool.Close()
	}
}
