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
	router.LoadHTMLGlob("views/*.html")

	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			c.Abort()
		}
	}, AddFileHandler(webViews))

	router.Use(utils.RedirectLimiter).GET("/:id", func(ctx *gin.Context) {
		shortenID := utils.ShortURL(strings.TrimSpace(ctx.Param("id")))
		if urlData, err := shortenID.GetData(); urlData != nil {
			urlData.IncreaseCount()
			// no custom meta: header redirect
			if urlData.Meta == nil {
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
		} else if err != nil {
			// server error
			ctx.HTML(http.StatusInternalServerError, "500.html", gin.H{"error": "internal server error"})
			return
		}
		// short url not found
		ctx.HTML(http.StatusNotFound, "404.html", nil)
	})

	apiRouter := router.Group("/api")
	apiRouter.Use(utils.ShortenLimiter).POST("/shorten", func(ctx *gin.Context) {
		data := utils.CreateData{}
		if err := ctx.BindJSON(&data); err != nil {
			// check data is valid
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		data.URL = utils.LongURL(strings.TrimSpace(string(data.URL)))
		// check whether url is empty
		if data.URL == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "original URL is required"})
			return
		}
		// check whether url format is valid
		if err := data.URL.IsValid(); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// check whether custom url has been used
		data.CustomURL = utils.ShortURL(strings.TrimSpace(string(data.CustomURL)))
		if data.CustomURL == "" {
			// check whether meta data is same
			if urlDate, err := data.URL.CheckMetaSame(data); urlDate != nil {
				// data exists and same, return it
				ctx.JSON(http.StatusOK, gin.H{
					"short": string(urlDate.ShortURL),
					"url":   data.URL,
					"meta":  data.Meta})
				return
			} else if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
				return
			}
		} else if err := data.CustomURL.IsValid(); err != nil {
			// check whether shortURL format is valid
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else if old, err := data.CustomURL.GetData(); old != nil {
			// check whether shortURL has been used
			if data.URL != old.TargetURL || data.Meta != old.Meta {
				// used
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "this custom url is already been used"})
			} else {
				// same as old, return it
				ctx.JSON(http.StatusOK, old)
			}
			return
		} else if err != nil {
			// db error
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		// if has meta, fill meta field
		if data.Meta != nil {
			// check whether image url format is valid
			if data.Meta.ImageURL != "" && !data.Meta.ImageURLIsValid() {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid image url"})
				return
			}
			data.InsertMeta()
		}

		// create short url
		urlDate, err := data.CreateShortURL()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		ctx.JSON(http.StatusCreated, urlDate)
	})

	apiRouter.Use(utils.GetShortenLimiter).GET("/get/:id", func(ctx *gin.Context) {
		shortenID := utils.ShortURL(strings.TrimSpace(ctx.Param("id")))
		if urlData, err := shortenID.GetData(); urlData != nil {
			ctx.JSON(http.StatusOK, urlData)
			return
		} else if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
	utils.CloseDB()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Shutdown Error: ", err)
	}
	log.Println("Server has been shutdown.")
}
