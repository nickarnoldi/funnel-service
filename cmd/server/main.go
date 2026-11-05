package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/nickarnoldi/funnel-service/internal/client"
	"github.com/nickarnoldi/funnel-service/internal/client/stats"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/env"
	"github.com/nickarnoldi/funnel-service/internal/server/api"
	"github.com/nickarnoldi/funnel-service/internal/server/api/handler"
	"github.com/nickarnoldi/funnel-service/internal/server/middleware"
)

const ResyncThreshold = 12 * time.Hour

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	env.Setup()

	store := db.NewStore(ctx)
	defer store.Close()

	serverPort := env.Get("LISTEN_PORT", "5000")

	handler := handler.Handler{
		Store: store,
	}

	stack := middleware.CreateStack(
		middleware.CORS,
		middleware.Logging,
	)

	router := http.NewServeMux()
	api.AddRoutes(router, &handler)

	server := &http.Server{
		Addr:    net.JoinHostPort("", serverPort),
		Handler: stack(router),
	}

	startDataResync(ctx, store)
	startHourlyStatsProcessor(ctx, store)

	log.Printf("Starting server on :%s\n", serverPort)
	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func startHourlyStatsProcessor(ctx context.Context, store *db.Store) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := stats.ProcessHourlyStats(ctx, store); err != nil {
					log.Printf("Error processing hourly click stats: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func startDataResync(ctx context.Context, store *db.Store) {
	availableNetworks := enum.Network

	// collect the last update times for each network, and then kick off a resync in the background
	// if the last update was within the last 24 hours
	lastUpdates := make(map[enum.NetworkEnum]time.Time)
	for networkID, networkName := range availableNetworks {
		lastClickUpdate, err := store.Queries.GetLastRawClickTimestampByNetwork(ctx, int32(networkID))
		if err != nil || !lastClickUpdate.Valid {
			log.Printf("Failed to get last raw click timestamp for network %s: %v", networkName, err)
			continue
		}

		lastUpdates[networkID] = lastClickUpdate.Time
	}

	go func() {
		for networkID, lastUpdate := range lastUpdates {
			networkName := availableNetworks[networkID]
			duration := time.Since(lastUpdate)

			if duration > ResyncThreshold {
				log.Printf("Skipping resync for network %s, last update was more than 24 hours ago", networkName)
				continue
			}

			log.Printf("Resyncing clicks/conversions for network %s since %v (%s ago)", networkName, lastUpdate, duration)

			client.SyncClicks(ctx, store, lastUpdate, time.Time{}, networkID)
			client.SyncConversions(ctx, store, lastUpdate, time.Time{}, networkID)
		}
	}()
}
