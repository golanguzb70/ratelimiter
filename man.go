package ratelimiter

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	RedisHost    string
	RedisPort    string
	JwtSignInKey string
	LeakyBuckets []*LeakyBucket
}

type RateLimiterI interface {
	GinMiddleware() gin.HandlerFunc
}

type ratelimiter struct {
	leakyBuckets map[string]LeakyBucketI
	jwtSignInKey string
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
		jwtSignInKey: cfg.JwtSignInKey,
	}, nil
}

func (r *ratelimiter) LeakyBucket() map[string]LeakyBucketI {
	return r.leakyBuckets
}

func (r *ratelimiter) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the bucket
		bucket, ok := r.leakyBuckets[Hash(c.Request.Method+c.FullPath())]
		if !ok {
			c.Next()
			return
		}

		claims, err := r.ParseJwt(c)
		if err != nil {
			if bucket.GetAllowOnError() {
				c.Next()
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
			}
			return
		}

		key, ok := claims[bucket.GetJwtKey()].(string)
		if !ok {
			if bucket.GetAllowOnError() {
				c.Next()
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
				c.Abort()
			}
			return
		}

		if !bucket.AllowRequest(c, key) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (r *ratelimiter) ParseJwt(c *gin.Context) (jwt.MapClaims, error) {
	token, err := jwt.Parse(c.GetHeader("Authorization"), func(token *jwt.Token) (interface{}, error) {
		return r.jwtSignInKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
