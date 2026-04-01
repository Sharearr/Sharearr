package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"example/main/internal/sharearr"
)

func main() {
	_ = godotenv.Load()

	cfg, err := sharearr.LoadConfig(os.Args[1:])
	if errors.Is(err, sharearr.ErrHelp) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	db, err := sharearr.OpenDB(cfg.DB)
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

	if err := sharearr.NewUserServiceFromDB(db).Init(context.Background(), cfg.Init.User); err != nil {
		log.Printf("init user: %v", err)
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
	router.Run(fmt.Sprintf(":%d", cfg.Port))
}
