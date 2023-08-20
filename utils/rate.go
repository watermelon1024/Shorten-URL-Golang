package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/ratelimit"
)

const RPS = 100

var limit = ratelimit.New(RPS)

func LeakBucket() gin.HandlerFunc {
	prev := time.Now()

	return func(c *gin.Context) {
		now := limit.Take()
		if now.Sub(prev) < 10*time.Second {
			c.JSON(429, gin.H{"error": "too many requests"})
			c.Abort()
		}
		prev = now
	}
}
