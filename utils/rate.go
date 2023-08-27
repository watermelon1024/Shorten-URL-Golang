package utils

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

var (
	RedirectLimiter gin.HandlerFunc
	GetShortenLimiter  gin.HandlerFunc
	ShortenLimiter  gin.HandlerFunc
)

func init() {
	// GET "/:id"
	redirectRateLimiter := limiter.New(memory.NewStore(), limiter.Rate{
		Period: 3 * time.Second,
		Limit:  3,
	})
	RedirectLimiter = mgin.NewMiddleware(redirectRateLimiter, mgin.WithLimitReachedHandler(limitReachedHandler))
	
	// GET "/api/get/:id"
	getShortenRateLimiter := limiter.New(memory.NewStore(), limiter.Rate{
		Period: 10 * time.Minute,
		Limit:  50,
	})
	GetShortenLimiter = mgin.NewMiddleware(getShortenRateLimiter, mgin.WithLimitReachedHandler(limitReachedHandler))

	// POST "/api/shorten"
	shortenRateLimiter := limiter.New(memory.NewStore(), limiter.Rate{
		Period: 10 * time.Minute,
		Limit:  50,
	})
	ShortenLimiter = mgin.NewMiddleware(shortenRateLimiter, mgin.WithLimitReachedHandler(limitReachedHandler))
}

func limitReachedHandler(c *gin.Context) {
	c.JSON(429, gin.H{"error": "too many requests"})
	c.Abort()
}
