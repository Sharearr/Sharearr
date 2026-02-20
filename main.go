package main

import (
	"github.com/gin-gonic/gin"

	httpTrackerServer "github.com/anacrolix/torrent/tracker/http/server"
	trackerServer "github.com/anacrolix/torrent/tracker/server"
)

func main() {
	handler := &httpTrackerServer.Handler{
		Announce: &trackerServer.AnnounceHandler{
			AnnounceTracker: NewMemTracker(),
		},
	}

	router := gin.Default()
	router.GET("/announce", gin.WrapH(handler))
	router.Run()
}
