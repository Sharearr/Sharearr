//go:build !dev

package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed app/dist
var dist embed.FS

func StaticHandler() gin.HandlerFunc {
	sub, err := fs.Sub(dist, "app/dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return func(c *gin.Context) {
		// Vite fingerprints everything under assets/ — safe to cache forever.
		// index.html must never be cached so users always get the latest entry point.
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Header("Cache-Control", "max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
