//go:build !dev

package web

import (
	"embed"
	"io/fs"
	"net/http"

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
		c.Writer.WriteHeader(http.StatusOK)
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
