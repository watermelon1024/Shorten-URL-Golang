package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	limiter "github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

var (
	Middleware gin.HandlerFunc
)

func init() {
	rate := limiter.Rate{
		Period: 10 * time.Minute,
		Limit:  50,
	}
	option, _ := redis.ParseURL("redis://localhost:6379/0")
	client := redis.NewClient(option)

	store, _ := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   "limiter_prefix",
	})

	Middleware = mgin.NewMiddleware(limiter.New(store, rate))
}