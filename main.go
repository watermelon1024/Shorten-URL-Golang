package main

import (
	"context"
	"embed"
	"fmt"
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

	router.GET("/s/:url", func(ctx *gin.Context) {
		url := ctx.Param("url")
		if longURL, ok := urlCache[url]; ok {
			ctx.Redirect(http.StatusMovedPermanently, longURL)
			return
		}
	})

	apiRouter := router.Group("/api")
	apiRouter.POST("/shorten", func(ctx *gin.Context) {
		data := CreateData{}
		if err := ctx.BindJSON(&data); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid json"})
			ctx.Abort()
			return
		}

		fmt.Println(data)
		if !HasIsURL(data.URL) {
			ctx.JSON(400, gin.H{"error": "invalid url"})
			ctx.Abort()
			return
		}

		ctx.JSON(200, gin.H{"shorten": CreateShortULR(data.URL), "url": data.URL})
	})

	gin.ForceConsoleColor()
	srv := &http.Server{
		Addr:    "127.0.0.1:8080",
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
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
