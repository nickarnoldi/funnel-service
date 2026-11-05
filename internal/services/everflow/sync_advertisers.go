package everflow

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
)

func ImportAdvertiser(ctx context.Context, store *db.Store, efAdvertiser *everflow.AdvertiserData) error {
	if efAdvertiser.ID == 0 {
		return fmt.Errorf("skipping Advertiser with empty ID")
	}

	if efAdvertiser.Name == "" {
		return fmt.Errorf("skipping Advertiser with empty Name")
	}

	networkID := enum.NetworkEnum(efAdvertiser.NetworkID)
	platform := networkID.GetPlatform()

	if platform == enum.PlatformUnknown {
		return fmt.Errorf("unknown platform for Advertiser %s with NetworkID %d, skipping", efAdvertiser.Name, efAdvertiser.NetworkID)
	}

	refID, err := store.GetReferenceID(ctx, platform, "advertiser", int64(efAdvertiser.ID))
	if err != nil {
		return fmt.Errorf("error getting local ID for Advertiser %s with imported ID %d: %w", efAdvertiser.Name, efAdvertiser.ID, err)
	}

	statusInt, exists := enum.Index(enum.Status, efAdvertiser.Status)
	if !exists {
		return fmt.Errorf("unknown status %s for Advertiser %s, skipping", efAdvertiser.Status, efAdvertiser.Name)
	}

	attributionMethodInt, found := enum.Index(enum.AttributionMethod, efAdvertiser.AttributionMethod)
	if !found {
		return fmt.Errorf("unknown attribution method %s for Advertiser %s, skipping", efAdvertiser.AttributionMethod, efAdvertiser.Name)
	}

	emailAttributionMethodInt, found := enum.Index(enum.EmailAttributionMethod, efAdvertiser.EmailAttributionMethod)
	if !found {
		return fmt.Errorf("unknown email attribution method %s for Advertiser %s, skipping", efAdvertiser.EmailAttributionMethod, efAdvertiser.Name)
	}

	attributionPriorityInt, found := enum.Index(enum.AttributionPriority, efAdvertiser.AttributionPriority)
	if !found {
		return fmt.Errorf("unknown attribution priority %s for Advertiser %s, skipping", efAdvertiser.AttributionPriority, efAdvertiser.Name)
	}

	advertiserRow, err := store.Queries.GetAdvertiserByReferenceID(ctx, db.BigInt(&refID))
	if err == pgx.ErrNoRows {
		if _, err := store.Queries.CreateAdvertiser(ctx, db.CreateAdvertiserParams{
			ReferenceID:           db.BigInt(&refID),
			NetworkID:             int32(efAdvertiser.NetworkID),
			Name:                  efAdvertiser.Name,
			Status:                int32(statusInt),
			DefaultCurrency:       efAdvertiser.DefaultCurrencyID,
			AttributionMethod:     int32(attributionMethodInt),
			EmailAttibutionMethod: int32(emailAttributionMethodInt),
			AttributionPriority:   int32(attributionPriorityInt),
		}); err != nil {
			return fmt.Errorf("error creating Advertiser for %s with imported ID %d: %w", efAdvertiser.Name, efAdvertiser.ID, err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("error fetching Advertiser for %s with imported ID %d: %w", efAdvertiser.Name, efAdvertiser.ID, err)
	}

	if _, err := store.Queries.UpdateAdvertiser(ctx, db.UpdateAdvertiserParams{
		ID:                    advertiserRow.Advertiser.ID,
		Name:                  efAdvertiser.Name,
		Status:                int32(statusInt),
		DefaultCurrency:       efAdvertiser.DefaultCurrencyID,
		AttributionMethod:     int32(attributionMethodInt),
		EmailAttibutionMethod: int32(emailAttributionMethodInt),
		AttributionPriority:   int32(attributionPriorityInt),
	}); err != nil {
		return fmt.Errorf("error updating Advertiser %s with imported ID %d: %w", efAdvertiser.Name, efAdvertiser.ID, err)
	}

	return nil
}

func SyncAdvertisers(ctx context.Context, store *db.Store, requestedNetworks ...enum.NetworkEnum) error {
	availableNetworks := enum.FilterNetwork(requestedNetworks)

	for network := range availableNetworks {
		advertisers, err := GetAdvertisers(ctx, network)
		if err != nil {
			fmt.Print(err)
			continue
		}

		fmt.Printf("%d Advertisers pulled from %s\n", len(*advertisers), network)

		var failures int

		for _, efAdvertiser := range *advertisers {
			if err := ImportAdvertiser(ctx, store, &efAdvertiser); err != nil {
				fmt.Print(err)
				failures++
			}
		}

		if failures > 0 {
			fmt.Printf("imported advertisers with %d failures out of %d total\n", failures, len(*advertisers))
		}
	}

	return nil
}
