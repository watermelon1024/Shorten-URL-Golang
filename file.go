// reference: https://gist.github.com/CJEnright/bc2d8b8dc0c1389a9feeddb110f822d7
package main

import (
	"compress/gzip"
	"embed"
	"errors"
	"io"
	fsLib "io/fs"
	"net/http"
	pathLib "path"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// HTML contains the given interface object.
type HTML struct{ Data string }

// Render (HTML) writes data with custom ContentType.
func (r HTML) Render(w http.ResponseWriter) (err error) {
	if _, err = w.Write([]byte(r.Data)); err != nil {
		panic(err)
	}
	return
}

// WriteContentType (JSON) writes JSON ContentType.
func (r HTML) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if len(header["Content-Type"]) == 0 {
		header["Content-Type"] = []string{"text/html; charset=utf-8"}
	}
}

var checkDynamicRoute = regexp.MustCompile(`/\[[^/]*\]`)

var gzPool = sync.Pool{
	New: func() any {
		w := gzip.NewWriter(io.Discard)
		gzip.NewWriterLevel(w, gzip.BestCompression)
		return w
	},
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func AddFileHandler(path embed.FS) func(c *gin.Context) {
	dir, err := fsLib.Sub(path, "views")
	if err != nil {
		panic(err)
	}
	fs := http.FS(dir)
	fileServer := http.FileServer(fs)
	notFoundPage, _ := path.ReadFile("views/404.html")

	routes := getRoutes(path)
	return func(c *gin.Context) {
		/* ---------- 404 page ---------- */
		UPath := pathLib.Clean(c.Request.URL.Path)

		if ok, path := routes.HasIs(UPath); ok {
			// suffix is `/` is important
			// if not, will be redirect to `${path}/${path}` ( is unlimited loop )
			c.Request.URL.Path = path + "/"
		}

		_, err := fs.Open(UPath)
		if err != nil && errors.Is(err, fsLib.ErrNotExist) {
			c.Render(http.StatusNotFound, HTML{Data: string(notFoundPage)})
			return
		}

		/* ---------- gzip ---------- */
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// support gzip
		c.Header("Content-Encoding", "gzip")

		gz := gzPool.Get().(*gzip.Writer)
		defer gzPool.Put(gz)

		gz.Reset(c.Writer)
		defer gz.Close()

		fileServer.ServeHTTP(&gzipResponseWriter{ResponseWriter: c.Writer, Writer: gz}, c.Request)
	}
}
