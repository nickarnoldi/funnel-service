package client

import (
	"context"
	"fmt"
	"time"

	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow"
	"github.com/nickarnoldi/funnel-service/internal/services/leads"
)

// SyncAll synchronizes all Everflow data into the local database.
// It sequentially calls each specific sync function and returns an error if any of them fail.
// Order is important as some data depends on parent relationship.
func SyncAll(ctx context.Context, store *db.Store) error {
	if err := SyncCategories(ctx, store); err != nil {
		return fmt.Errorf("failed to sync categories: %w", err)
	}

	if err := SyncAdvertisers(ctx, store); err != nil {
		return fmt.Errorf("failed to sync advertisers: %w", err)
	}

	if err := SyncOffers(ctx, store); err != nil {
		return fmt.Errorf("failed to sync offers: %w", err)
	}

	if err := SyncOfferURLs(ctx, store); err != nil {
		return fmt.Errorf("failed to sync offer URLs: %w", err)
	}

	if err := SyncAffiliates(ctx, store); err != nil {
		return fmt.Errorf("failed to sync affiliates: %w", err)
	}

	return nil
}

func SyncCategories(ctx context.Context, store *db.Store) error {
	if err := everflow.SyncCategories(ctx, store); err != nil {
		return fmt.Errorf("failed to sync categories: %w", err)
	}
	return nil
}

func SyncAdvertisers(ctx context.Context, store *db.Store) error {
	if err := everflow.SyncAdvertisers(ctx, store); err != nil {
		return fmt.Errorf("failed to sync advertisers: %w", err)
	}
	return nil
}

func SyncOffers(ctx context.Context, store *db.Store) error {
	if err := everflow.SyncOffers(ctx, store); err != nil {
		return fmt.Errorf("failed to sync offers: %w", err)
	}
	return nil
}

func SyncOfferURLs(ctx context.Context, store *db.Store) error {
	if err := everflow.SyncOfferURLs(ctx, store); err != nil {
		return fmt.Errorf("failed to sync offer URLs: %w", err)
	}
	return nil
}

func SyncAffiliates(ctx context.Context, store *db.Store, networkID ...enum.NetworkEnum) error {
	if err := everflow.SyncAffiliates(ctx, store, networkID...); err != nil {
		return fmt.Errorf("failed to sync affiliates: %w", err)
	}
	return nil
}

func SyncClicks(ctx context.Context, store *db.Store, startDate, endDate time.Time, networkIDs ...enum.NetworkEnum) error {
	if err := everflow.SyncRawClicks(ctx, store, startDate, endDate, networkIDs...); err != nil {
		return fmt.Errorf("failed to sync clicks: %w", err)
	}
	return nil
}

func SyncConversions(ctx context.Context, store *db.Store, startDate, endDate time.Time, networkIDs ...enum.NetworkEnum) error {
	if err := everflow.SyncRawConversions(ctx, store, startDate, endDate, networkIDs...); err != nil {
		return fmt.Errorf("failed to sync conversions: %w", err)
	}
	return nil
}

func SyncFunnels(ctx context.Context, store *db.Store, ids ...int64) error {
	if err := leads.SyncFunnels(ctx, store, ids...); err != nil {
		return fmt.Errorf("failed to sync funnels: %w", err)
	}
	return nil
}

func SyncLeadsByDateRange(ctx context.Context, store *db.Store, startDate, endDate time.Time) error {
	if err := leads.SyncLeadsByDateRange(ctx, store, startDate, endDate); err != nil {
		return fmt.Errorf("failed to sync leads: %w", err)
	}
	return nil
}

func SyncLeadsByIDs(ctx context.Context, store *db.Store, ids []int64) error {
	if err := leads.SyncLeadsByIDs(ctx, store, ids); err != nil {
		return fmt.Errorf("failed to sync leads by ID: %w", err)
	}
	return nil
}

func SyncLeadsByFunnelIDs(ctx context.Context, store *db.Store, funnelIDs []int64) error {
	if err := leads.SyncLeadsByFunnelIDs(ctx, store, funnelIDs); err != nil {
		return fmt.Errorf("failed to sync leads by funnel ID: %w", err)
	}
	return nil
}
