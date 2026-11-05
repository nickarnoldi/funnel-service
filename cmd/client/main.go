package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/nickarnoldi/funnel-service/internal/client"
	"github.com/nickarnoldi/funnel-service/internal/client/backfill"
	"github.com/nickarnoldi/funnel-service/internal/client/stats"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/env"
	"github.com/nickarnoldi/funnel-service/internal/flags"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	env.Setup()

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if len(os.Args) < 2 {
		return errors.New("expected a command")
	}

	store := db.NewStore(ctx)
	defer store.Close()

	switch os.Args[1] {
	case "sync":
		return client.SyncAll(ctx, store)
	case "sync:categories":
		return client.SyncCategories(ctx, store)
	case "sync:affiliates":
		return client.SyncAffiliates(ctx, store)
	case "sync:advertisers":
		return client.SyncAdvertisers(ctx, store)
	case "sync:offers":
		return client.SyncOffers(ctx, store)
	case "sync:offer_urls":
		return client.SyncOfferURLs(ctx, store)
	case "sync:funnels":
		return client.SyncFunnels(ctx, store)
	case "sync:clicks":
		cmd := flags.NewFlagSet("sync:clicks", flag.ExitOnError)
		startDate, endDate := cmd.DateRange()
		network := cmd.Network()
		cmd.Parse(os.Args[2:])

		if network.Valid {
			return client.SyncClicks(ctx, store, *startDate, *endDate, network.Enum)
		}

		return client.SyncClicks(ctx, store, *startDate, *endDate)
	case "sync:conversions":
		cmd := flags.NewFlagSet("sync:conversions", flag.ExitOnError)
		startDate, endDate := cmd.DateRange()
		network := cmd.Network()
		cmd.Parse(os.Args[2:])

		if network.Valid {
			return client.SyncConversions(ctx, store, *startDate, *endDate, network.Enum)
		}

		return client.SyncConversions(ctx, store, *startDate, *endDate)
	case "sync:leads":
		cmd := flags.NewFlagSet("sync:leads", flag.ExitOnError)
		startDate, endDate := cmd.DateRange()
		leadIDs := cmd.Int64Slice("id", "AbandonedLead ID to sync leads for")
		funnelIDs := cmd.Int64Slice("funnel", "Funnel ID to sync leads for")
		cmd.Parse(os.Args[2:])

		if len(*leadIDs) > 0 {
			return client.SyncLeadsByIDs(ctx, store, *leadIDs)
		}

		if len(*funnelIDs) > 0 {
			return client.SyncLeadsByFunnelIDs(ctx, store, *funnelIDs)
		}

		return client.SyncLeadsByDateRange(ctx, store, *startDate, *endDate)
	case "backfill:leads":
		cmd := flags.NewFlagSet("backfill:leads", flag.ExitOnError)
		startDate, endDate := cmd.DateRange()
		cmd.Parse(os.Args[2:])

		return backfill.LeadsMissingClicks(ctx, store, *startDate, *endDate)
	case "backfill:clicks":
		cmd := flags.NewFlagSet("backfill:clicks", flag.ExitOnError)
		id := cmd.Int64("id", 0, "ID to start the process after")
		cmd.Parse(os.Args[2:])

		return backfill.Clicks(ctx, store, *id)
	case "backfill:conversions":
		cmd := flags.NewFlagSet("backfill:conversions", flag.ExitOnError)
		id := cmd.Int64("id", 0, "ID to start the process after")
		cmd.Parse(os.Args[2:])

		return backfill.Conversions(ctx, store, *id)
	case "sync:stats":
		cmd := flags.NewFlagSet("sync:stats", flag.ExitOnError)
		funnelIDs := cmd.Int64Slice("id", "Funnel ID to process stats for")
		cmd.Parse(os.Args[2:])

		_, err := stats.ProcessHourlyStatsByFunnelIDs(ctx, store, *funnelIDs)
		return err
	case "backfill:stats":
		return stats.ProcessHourlyStats(ctx, store, stats.HourlyStatsParams{Backfill: true})
	case "stats":
		return stats.ProcessHourlyStats(ctx, store)
	case "performance":
		serverPort := env.Get("LISTEN_PORT", "5000")

		cmd := flags.NewFlagSet("performance", flag.ExitOnError)
		perfUrl := cmd.String("url", fmt.Sprintf("http://localhost:%s/health", serverPort), fmt.Sprintf("Provide a URL to perform the test against, defaults to http://localhost:%s/health", serverPort))
		perfSampleSize := cmd.Int("size", client.SampleSize, "Number of samples to run, defaults to 128")
		cmd.Parse(os.Args[2:])

		log.Printf("Running %d concurrent requests against %s\n", *perfSampleSize, *perfUrl)
		return client.CheckPerformance(ctx, *perfUrl, *perfSampleSize)
	default:
		return fmt.Errorf("unsupported command: %s", os.Args[1])
	}
}
