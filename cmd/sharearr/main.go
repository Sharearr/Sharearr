package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"example/main/internal/sharearr"
	"example/main/web"
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
	logMiddleware := sharearr.SetupLogger(cfg.Log)

	db, err := sharearr.OpenDB(cfg.DB)
	if err != nil {
		panic(err)
	}
	defer db.Close() //nolint:errcheck

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		peers := sharearr.NewPeerServiceFromDB(db)
		for {
			slog.Debug("Deleting stale peers")
			if err := peers.DeleteStale(context.Background()); err != nil {
				slog.Error("Deleting stale peers failed", "error", err)
			}
			<-ticker.C
		}
	}()

	if err := sharearr.NewUserServiceFromDB(db).Init(context.Background(), cfg.Init.User); err != nil {
		panic(err)
	}

	router := gin.New()
	router.Use(logMiddleware)
	router.Use(gin.Recovery())

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
	router.NoRoute(web.StaticHandler())

	if err := router.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
