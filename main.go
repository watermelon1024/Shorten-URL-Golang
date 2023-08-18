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

	"github.com/compose-spec/compose-go/dotenv"
	"github.com/gin-gonic/gin"
)

var (
	//go:embed all:views
	webViews embed.FS

	HOSTNAME string
	GIN_MODE string
)

type CreateData struct {
	URL         string `json:"url" binding:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CustomURL   string `json:"customUrl"`
}

func main() {
	dotenv.Load()
	HOSTNAME = os.Getenv("HOSTNAME")
	GIN_MODE = os.Getenv("GIN_MODE")
	gin.SetMode(GIN_MODE)

	router := gin.Default()
	router.NoRoute(AddFileHandler(webViews), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "not found"})
			c.Abort()
		}
	})

	router.LoadHTMLFiles("views/redirect.html")
	router.GET("/:id", func(ctx *gin.Context) {
		shortenID := ctx.Param("id")
		if urlData, ok := urlCache[shortenID]; ok {
			urlData.increaseCount(shortenID)

			if len(urlData.Title) == 0 && len(urlData.Description) == 0 {
				ctx.Redirect(http.StatusTemporaryRedirect, urlData.TargetURL)
				return
			}

			ctx.HTML(http.StatusOK, "redirect.html", gin.H{
				"title":       urlData.Title,
				"description": urlData.Description,
				"targetURL":   urlData.TargetURL,
			})
			return
		}

		AddFileHandler(webViews)(ctx)
	})

	apiRouter := router.Group("/api")
	apiRouter.POST("/shorten", func(ctx *gin.Context) {
		data := CreateData{}
		if err := ctx.BindJSON(&data); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid JSON"})
			ctx.Abort()
			return
		}
		// check whether url format is valid
		if valid, errMessage := isValidURL(data.URL); !valid {
			ctx.JSON(400, gin.H{"error": errMessage})
			ctx.Abort()
			return
		}
		// check whether longURL is in cache
		if shortURL, ok := longURLCache[data.URL]; ok {
			ctx.JSON(200, gin.H{"shorten": shortURL, "url": data.URL})
			ctx.Abort()
			return
		}
		// check whether shortURL is used
		if _, ok := urlCache[data.CustomURL]; ok {
			ctx.JSON(400, gin.H{"error": "this custom url is already been used"})
			ctx.Abort()
			return
		}

		urlData := CreateShortURL(&data)
		ctx.JSON(201, gin.H{"shorten": urlData.ShortURL, "url": data.URL})
	})
	apiRouter.GET("/get/:id", func(ctx *gin.Context) {
		shortenID := ctx.Param("id")
		if urlData, ok := urlCache[shortenID]; ok {
			ctx.JSON(200, gin.H{
				"targetURL":   urlData.TargetURL,
				"title":       urlData.Title,
				"description": urlData.Description,
				"image":       urlData.ImageURL,
				"count":       urlData.Count,
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
