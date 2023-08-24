package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

var (
	LimiterMiddleware gin.HandlerFunc
)

func init() {
	rateLimiter := limiter.New(memory.NewStore(), limiter.Rate{
		Period: 10 * time.Minute,
		Limit:  50,
	})

	LimiterMiddleware = mgin.NewMiddleware(rateLimiter, mgin.WithLimitReachedHandler(func(c *gin.Context) {
		c.JSON(429, gin.H{"error": "too many requests"})
		c.Abort()
	}))
}
