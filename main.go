package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shorten-url/utils"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/dotenv"
	"github.com/gin-gonic/gin"
)

//go:embed all:views
var webViews embed.FS

func init() {
	dotenv.Load()
	gin.SetMode(strings.ToLower(os.Getenv("GIN_MODE")))
}

func main() {
	router := gin.Default()
	router.NoRoute(AddFileHandler(webViews), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "not found"})
			c.Abort()
		}
	})

	router.SetHTMLTemplate(template.Must(template.New("").ParseFS(webViews, "views/redirect.html")))

	router.GET("/:id", func(ctx *gin.Context) {
		shortenID := utils.ShortURL(ctx.Param("id"))
		if urlData, ok := shortenID.GetData(); ok {
			urlData.IncreaseCount()

			if len(urlData.Title) == 0 && len(urlData.Description) == 0 {
				ctx.Redirect(http.StatusTemporaryRedirect, string(urlData.TargetURL))
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
		data := utils.CreateData{}
		if err := ctx.BindJSON(&data); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid JSON"})
			return
		}
		// check whether url format is valid
		if valid, errMessage := utils.IsValidURL(string(data.URL)); !valid {
			ctx.JSON(400, gin.H{"error": errMessage})
			return
		}
		// check whether longURL is in cache
		if shortURL, ok := data.URL.GetData(); ok {
			// check whether meta data is same
			if shortURL.Title == data.Title || shortURL.Description == data.Description {
				ctx.JSON(200, shortURL)
				return
			}
		}
		// check whether shortURL is used
		if _, ok := data.CustomURL.GetData(); ok {
			ctx.JSON(400, gin.H{"error": "this custom url is already been used"})
			return
		}

		data.InsertMeta()

		ctx.JSON(200, data.CreateShortURL())
	})
	apiRouter.GET("/get/:id", func(ctx *gin.Context) {
		shortenID := utils.ShortURL(ctx.Param("id"))
		if urlData, ok := shortenID.GetData(); ok {
			ctx.JSON(200, gin.H{
				"targetURL":   urlData.TargetURL,
				"title":       urlData.Title,
				"description": urlData.Description,
				"image":       urlData.ImageURL,
				"count":       urlData.Count,
			})
			return
		}

		ctx.JSON(404, gin.H{"error": "not found"})
	})

	gin.ForceConsoleColor()
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	if gin.Mode() != gin.ReleaseMode {
		srv.Addr = "127.0.0.1:8080"
	}

	go func() {
		log.Println("Server starting...")
		log.Println("listen", srv.Addr)

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
