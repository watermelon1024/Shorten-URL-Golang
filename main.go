package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"shorten-url/utils"

	"github.com/compose-spec/compose-go/dotenv"
	"github.com/gin-gonic/gin"
)

var GIT_COMMIT string

//go:embed all:views
var webViews embed.FS

func init() {
	dotenv.Load()
	gin.SetMode(strings.ToLower(os.Getenv("GIN_MODE")))
}

func main() {
	log.Println("Git Commit:", GIT_COMMIT)

	router := gin.Default()
	router.NoRoute(AddFileHandler(webViews), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "not found"})
			c.Abort()
		}
	})

	router.SetHTMLTemplate(template.Must(template.New("").ParseFS(webViews, "views/redirect.html")))

	router.GET("/:id", func(ctx *gin.Context) {
		shortenID := utils.ShortURL(strings.TrimSpace(ctx.Param("id")))
		if urlData, ok := shortenID.GetData(); ok {
			urlData.IncreaseCount()
			// no custom meta: header redirect
			if !urlData.Meta.HasData() {
				ctx.Redirect(http.StatusTemporaryRedirect, string(urlData.TargetURL))
				return
			}
			// has custom meta: js redirect
			ctx.HTML(http.StatusOK, "redirect.html", gin.H{
				"title":       urlData.Meta.Title,
				"description": urlData.Meta.Description,
				"image":       urlData.Meta.ImageURL,
				"color":       urlData.Meta.ThemeColor,
				"targetURL":   urlData.TargetURL,
			})
			return
		}

		AddFileHandler(webViews)(ctx)
	})

	apiRouter := router.Group("/api")
	apiRouter.Use(utils.LimiterMiddleware).POST("/shorten", func(ctx *gin.Context) {
		data := utils.CreateData{}
		if err := ctx.BindJSON(&data); err != nil {
			ctx.JSON(400, gin.H{"error": "invalid JSON"})
			return
		}
		data.URL = utils.LongURL(strings.TrimSpace(string(data.URL)))
		// check whether url is empty
		if data.URL == "" {
			ctx.JSON(400, gin.H{"error": "original URL is required"})
			return
		}
		// check whether url format is valid
		if err := data.URL.IsValid(); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		// if theme color is default (#000000), remove value
		if data.Meta.ThemeColor == "#000000" {
			data.Meta.ThemeColor = ""
		}
		data.CustomURL = utils.ShortURL(strings.TrimSpace(string(data.CustomURL)))
		if data.CustomURL == "" {
			// check whether longURL is in cache
			if old, ok := data.URL.GetData(); ok {
				// check whether meta data is same
				if old.Meta == data.Meta {
					ctx.JSON(200, old)
					return
				}
			}
			// check whether shortURL format is valid
		} else if err := data.CustomURL.IsValid(); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
			// check whether shortURL is used
		} else if old, ok := data.CustomURL.GetData(); ok {
			if data.URL != old.TargetURL {
				ctx.JSON(400, gin.H{"error": "this custom url is already been used"})
			} else {
				ctx.JSON(200, old)
			}
			return
		}

		if data.Meta.HasData() {
			data.InsertMeta()
		}

		ctx.JSON(201, data.CreateShortURL())
	})
	apiRouter.GET("/get/:id", func(ctx *gin.Context) {
		shortenID := utils.ShortURL(strings.TrimSpace(ctx.Param("id")))
		if urlData, ok := shortenID.GetData(); ok {
			ctx.JSON(200, urlData)
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
