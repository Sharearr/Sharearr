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

	"example/main/internal/sharearr"
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

	db, err := sharearr.OpenDB(cfg.db)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		peers := sharearr.NewPeerServiceFromDB(db)
		for {
			if err := peers.DeleteStale(context.Background()); err != nil {
				log.Printf("peer cleanup: %v", err)
			}
			<-ticker.C
		}
	}()

	if err := sharearr.NewUserServiceFromDB(db).Provision(context.Background()); err != nil {
		log.Printf("provision user: %v", err)
	}

	router := gin.Default()
	tracker := sharearr.NewTrackerHandlerFromDB(db)
	torznab := sharearr.NewTorznabHandlerFromDB(db)
	torrents := sharearr.NewTorrentHandlerFromDB(db)

	root := router.Group("/")
	root.Use(sharearr.Auth(db))
	{
		root.GET("announce", tracker.Announce)
		root.GET("announce/:apikey", tracker.Announce)
		api := root.Group("api")
		{
			api.GET("", torznab.Handle)
			v1 := api.Group("v1")
			{
				v1.GET("torrent/:id/download", torrents.Download)
				v1.POST("torrent", torrents.Upload)
				v1.POST("torrent/:cat", torrents.Upload)
			}
		}
	}
	router.Run(fmt.Sprintf(":%s", cfg.port))
}
