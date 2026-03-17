package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	httpTrackerServer "github.com/anacrolix/torrent/tracker/http/server"
	trackerServer "github.com/anacrolix/torrent/tracker/server"
)

type config struct {
	port string
	db   string
}

func main() {
	_ = godotenv.Load()

	var cfg config

	defaultPort := os.Getenv("SHAREARR_PORT")
	if defaultPort == "" {
		defaultPort = "8787"
	}
	defaultDb, present := os.LookupEnv("SHAREARR_DB")
	if !present {
		defaultDb = "sharearr.db"
	}
	flag.StringVar(&cfg.port, "port", defaultPort, "http port")
	flag.StringVar(&cfg.db, "db", defaultDb, "SQLite connection")

	flag.Parse()

	db, err := openDB(cfg.db)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		peers := NewPeerServiceFromDB(db)
		for {
			if err := peers.DeleteStale(context.Background()); err != nil {
				log.Printf("peer cleanup: %v", err)
			}
			<-ticker.C
		}
	}()

	if err := NewUserServiceFromDB(db).Provision(context.Background()); err != nil {
		log.Printf("provision user: %v", err)
	}

	handler := &httpTrackerServer.Handler{
		Announce: &trackerServer.AnnounceHandler{
			AnnounceTracker: NewDBTrackerFromDB(db),
		},
	}

	wrapped := gin.WrapH(handler)

	router := gin.Default()
	torznab := NewTorznabHandlerFromDB(db)
	torrents := NewTorrentHandlerFromDB(db)

	authorized := router.Group("/")
	authorized.Use(Auth(db))
	{
		authorized.GET("announce", wrapped)
		authorized.GET("announce/:apikey", wrapped)
		authorized.GET("torznab", torznab.Handle)
		authorized.GET("torrent/:id/download", torrents.Download)
		authorized.POST("torrent", torrents.Upload)
		authorized.POST("torrent/:cat", torrents.Upload)
	}
	router.Run(fmt.Sprintf(":%s", cfg.port))
}
