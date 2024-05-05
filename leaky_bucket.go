package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type LeakyBucketI interface {
	AllowRequest(ctx context.Context, key string) bool
	GetJwtKey() string
	GetType() string
	GetAllowOnError() bool
}

type leakyBucketService struct {
	Method       string
	Path         string
	RequestLimit int
	DurationType string
	Type         string
	JWTKey       string
	AllowOnError bool
	Id           int
	RedisClient  *redis.Client
}

func NewLeakyBucket(bucket *LeakyBucket, id int, redisClient *redis.Client) (LeakyBucketI, error) {
	message, ok := bucket.Validate()
	if !ok {
		return nil, fmt.Errorf("invalid LeakyBucket: %s", message)
	}

	return &leakyBucketService{
		Method:       bucket.Method,
		Path:         bucket.Path,
		RequestLimit: bucket.RequestLimit,
		DurationType: bucket.DurationType,
		JWTKey:       bucket.JWTKey,
		AllowOnError: bucket.AllowOnError,
		RedisClient:  redisClient,
		Type:         bucket.Type,
		Id:           id,
	}, nil
}

func (l *leakyBucketService) GetType() string {
	return l.Type
}

func (l *leakyBucketService) GetJwtKey() string {
	return l.JWTKey
}

func (l *leakyBucketService) GetAllowOnError() bool {
	return l.AllowOnError
}

func (l *leakyBucketService) AllowRequest(ctx context.Context, key string) bool {
	var (
		timeKey      = ""
		timeDuration = time.Second
	)

	switch l.DurationType {
	case "second":
		timeKey = time.Now().Format("2006-01-02 15:04:05")
	case "minute":
		timeKey = time.Now().Format("2006-01-02 15:04")
		timeDuration = time.Minute
	case "hour":
		timeKey = time.Now().Format("2006-01-02 15")
		timeDuration = time.Hour
	}

	key = fmt.Sprintf("%s%d%s", timeKey, l.Id, key)

	result, err := l.RedisClient.Get(ctx, key).Int()
	if err == redis.Nil {
		l.RedisClient.Set(ctx, key, l.RequestLimit-1, timeDuration)
	} else if err != nil {
		return l.AllowOnError
	}

	if result <= 0 {
		return false
	}

	l.RedisClient.DecrBy(ctx, key, 1).Err()
	if err != nil {
		return l.AllowOnError
	}

	return true
}
