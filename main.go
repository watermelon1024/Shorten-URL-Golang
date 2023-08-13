package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed all:views
var webViews embed.FS

type CreateData struct {
	URL string `json:"url" binding:"required"`
}

func main() {
	router := gin.Default()
	router.NoRoute(AddFileHandler(webViews), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "not found"})
			c.Abort()
		}
	})

	router.GET("/:id", func(ctx *gin.Context) {
		shortenID := ctx.Param("id")
		if urlData, ok := urlCache[shortenID]; ok {
			urlData.increaseCount(shortenID)
			ctx.Redirect(http.StatusTemporaryRedirect, urlData.TargetURL)
			return
		}

		AddFileHandler(webViews)(ctx)
	})

	apiRouter := router.Group("/api")
	apiRouter.POST("/shorten", func(ctx *gin.Context) {
		data := CreateData{}
		if err := ctx.BindJSON(&data); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid json"})
			ctx.Abort()
			return
		}

		if !isValidURL(data.URL) {
			ctx.JSON(400, gin.H{"error": "invalid url"})
			ctx.Abort()
			return
		}

		ctx.JSON(200, gin.H{"shorten": CreateShortULR(data.URL), "url": data.URL})
	})
	apiRouter.GET("/get/:id", func(ctx *gin.Context) {
		shortenID := ctx.Param("id")
		if urlData, ok := urlCache[shortenID]; ok {
			ctx.JSON(200, gin.H{
				"targetURL": urlData.TargetURL,
				"count":     urlData.Count,
			})
			ctx.Abort()
			return
		}

		ctx.JSON(404, gin.H{"error": "not found"})
		ctx.Abort()
	})

	gin.ForceConsoleColor()
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	go func() {
		log.Println("Server starting...")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Shutdown Error: ", err)
	}
	log.Println("Server has been shutdown.")
}
