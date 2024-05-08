# Rate Limiter
Rate limiter is a Golang package that makes it easy to implement rate limiting of gin apis.

# Use rate limiter when 
- serving more users is high priority.
- preventing crash of the whole system because of some molicious users is important.
- accessing specific apis are limited to user or ip address.

# Getting started
## Getting ratelimiter
With Go module support, simply add the following import
```
import "github.com/golanguzb70/ratelimiter"
```
to your code, and then `go mod tidy` will automatically fetch the necessary dependencies

Otherwise, run the following Go command to install the ratelimiter package
```
go get -u github.com/golanguzb70/ratelimiter
```

## Configuration fields.
```
+------------------+------------------------------------+------------------------------------------------------+
| Columns          | types  |  enum values              | description                                          |
|------------------|------------------------------------|------------------------------------------------------|
| method           | string |  GET, POST, PUT, DELETE   | http method                                          |
| path             | string |  *                        | http full path                                       |
| limit            | uint   |  *                        | the request limit per interval                       |
| interval         | string |  second, minute, hour     | interval type                                        |
| type             | string |  ip, jwt, header, query   | according to this field rate limiting key is choosen |
| key_field        | string |  *                        | this is name of key of jwt, header or query          |
| allow_on_failure | bool   |  *                        | if true and failure occurs, the request is served    |
| not_allow_msg    | string |  *                        | this message is sent to client when disallowed       |
| not_allow_code   | string |  *                        | this code is sent to client when disallowed          |
+--------------------------------------------------------------------------------------------------------------+
```

## Code level config example.
```
rateLimiterConfig := &ratelimiter.Config{
		RedisHost:    "localhost",
		RedisPort:    "6379",
		JwtSignInKey: "jwt_sign_in_key",
		LeakyBuckets: []*ratelimiter.LeakyBucket{
			{
				Method:         "GET",
				Path:           "/ping/:id",
				RequestLimit:   10,
				Interval:       "minute",
				Type:           "leaky_bucket",
				KeyField:       "session_id",
				AllowOnFailure: true,
				NotAllowMsg:    "Sorry we prefer to serve more people instead of serving you more",
				NotAllowCode:   "TOO_MANY_REQUESTS"
			},
		},
	}
```

## Parse config from yaml file.
```
cfg, err := ratelimiter.ParseYamlFile("rate-limit.yaml")
if err != nil {
    // handle error
}
```

## YAML config file example.
```
redis_host: localhost
redis_port: "6379"
jwt_sign_in_key: "your_jwt_sign_in_key"
leaky_buckets:
  - name: "bucket1"
    method: "GET"
    path: "/ping/:id"
    limit: 10
    interval: minute
    type: jwt
    jwt_key: session_id
    allow_on_failure: false
	not_allow_msg: "Sorry we prefer to serve more people instead of serving you more"
	not_allow_code: "TOO_MANY_REQUESTS"
```

## Gin middleware example
```
package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/temp/ratelimiter"
)

func main() {
	router := gin.Default()

	cfg, err := ratelimiter.ParseYamlFile("rate-limit.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}

	limiter, err := ratelimiter.NewRateLimiter(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	router.Use(limiter.GinMiddleware())

	router.GET("/ping/:id", func(c *gin.Context) {
		fmt.Println(c.FullPath())

		c.JSON(200, gin.H{
			"message": "pong " + c.Param("id"),
		})
	})

	router.Run(":8080")
}
```

