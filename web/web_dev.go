//go:build dev

package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

func StaticHandler() gin.HandlerFunc {
	viteURL := os.Getenv("VITE_URL")
	if viteURL == "" {
		viteURL = "http://localhost:5173"
	}
	target, _ := url.Parse(viteURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	return func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusOK)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
