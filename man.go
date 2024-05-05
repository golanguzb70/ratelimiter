package ratelimiter

import (
	"github.com/redis/go-redis/v9"
)

type Config struct {
	RedisHost    string
	RedisPort    string
	LeakyBuckets []*LeakyBucket
}

type RateLimiterI interface {
	LeakyBucket() map[string]LeakyBucketI
}

type ratelimiter struct {
	leakyBuckets map[string]LeakyBucketI
}

func NewRateLimiter(cfg *Config) (RateLimiterI, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.RedisHost + ":" + cfg.RedisPort,
	})

	leakyBuckets := map[string]LeakyBucketI{}
	for i, e := range cfg.LeakyBuckets {
		bucket, err := NewLeakyBucket(e, i, client)
		if err != nil {
			return nil, err
		}
		leakyBuckets[Hash(e.Method+e.Path)] = bucket
	}

	return &ratelimiter{
		leakyBuckets: leakyBuckets,
	}, nil
}

func (r *ratelimiter) LeakyBucket() map[string]LeakyBucketI {
	return r.leakyBuckets
}
