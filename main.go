package ratelimiter

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

type Config struct {
	RedisHost    string         `yaml:"redis_host"`
	RedisPort    string         `yaml:"redis_port"`
	JwtSignInKey string         `yaml:"jwt_sign_in_key"`
	LeakyBuckets []*LeakyBucket `yaml:"leaky_buckets"`
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

func ParseYamlFile(path string) (*Config, error) {
	cfg := &Config{}
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %w", path, err)
	}

	return cfg, err
}

func (r *ratelimiter) LeakyBucket() map[string]LeakyBucketI {
	return r.leakyBuckets
}

func (r *ratelimiter) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		bucket, ok := r.leakyBuckets[Hash(c.Request.Method+c.FullPath())]
		if !ok {
			c.Next()
			return
		}

		key := ""
		switch bucket.GetType() {
		case "header":
			key = c.GetHeader(bucket.GetJwtKey())
		case "ip":
			key = c.ClientIP()
		case "jwt":
			claims, err := r.ParseJwt(c)
			if err != nil {
				if bucket.GetAllowOnFailure() {
					c.Next()
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
					c.Abort()
				}
				return
			}

			key, ok = claims[bucket.GetJwtKey()].(string)
			if !ok {
				if bucket.GetAllowOnFailure() {
					c.Next()
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid jwt key"})
					c.Abort()
				}
				return
			}
		case "query":
			key = c.Query(bucket.GetJwtKey())
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
		return []byte(r.jwtSignInKey), nil
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
