package everflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
)

func ImportOfferURL(ctx context.Context, store *db.Store, efOfferURL *everflow.OfferURLData) error {
	if efOfferURL.ID == 0 {
		return fmt.Errorf("skipping OfferURL with empty ID")
	}

	if efOfferURL.Name == "" {
		return fmt.Errorf("skipping OfferURL with empty Name")
	}

	networkID := enum.NetworkEnum(efOfferURL.NetworkID)
	platform := networkID.GetPlatform()

	if platform == enum.PlatformUnknown {
		return fmt.Errorf("unknown platform for OfferURL %s with NetworkID %d, skipping", efOfferURL.Name, efOfferURL.NetworkID)
	}

	offerName := strings.TrimSpace(efOfferURL.Name)

	statusInt, exists := enum.Index(enum.OfferURLStatus, efOfferURL.Status)
	if !exists {
		return fmt.Errorf("unknown status %s for OfferURL %s, skipping", efOfferURL.Status, offerName)
	}

	refID, err := store.GetReferenceID(ctx, platform, "offer_url", int64(efOfferURL.ID))
	if err != nil {
		return fmt.Errorf("error getting local ID for Offer %s with imported ID %d: %v", offerName, efOfferURL.ID, err)
	}

	offerRow, err := store.Queries.GetOfferByExternalReferenceID(ctx, db.GetOfferByExternalReferenceIDParams{
		PlatformID: int32(platform),
		ExternalID: int64(efOfferURL.OfferID),
	})
	if err != nil {
		return fmt.Errorf("error fetching Offer for %s with imported OfferID %d: %v", offerName, efOfferURL.OfferID, err)
	}
	offer := offerRow.Offer

	url, err := store.Queries.GetOfferURLByReferenceID(ctx, db.BigInt(&refID))
	if err == pgx.ErrNoRows {
		if _, err := store.Queries.CreateOfferURL(ctx, db.CreateOfferURLParams{
			ReferenceID:    db.BigInt(&refID),
			NetworkID:      int32(efOfferURL.NetworkID),
			OfferID:        offer.ID,
			Name:           offerName,
			Status:         int32(statusInt),
			DestinationURL: efOfferURL.DestinationURL,
			PreviewURL:     db.String(&efOfferURL.PreviewURL),
		}); err != nil {
			return fmt.Errorf("error creating Offer for %s with imported ID %d: %v", offerName, efOfferURL.ID, err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("error fetching Offer for %s with imported ID %d: %v", offerName, efOfferURL.ID, err)
	}

	if _, err := store.Queries.UpdateOfferURL(ctx, db.UpdateOfferURLParams{
		ID:             url.ID,
		OfferID:        offer.ID,
		FunnelID:       url.FunnelID,
		Name:           offerName,
		Status:         int32(statusInt),
		DestinationURL: efOfferURL.DestinationURL,
		PreviewURL:     db.String(&efOfferURL.PreviewURL),
	}); err != nil {
		fmt.Printf("Error updating Offer URL for %s with imported ID %d: %v\n", offerName, efOfferURL.ID, err)
	}

	return nil
}

func SyncOfferURLs(ctx context.Context, store *db.Store, requestedNetworks ...enum.NetworkEnum) error {
	availableNetworks := enum.FilterNetwork(requestedNetworks)

	for network := range availableNetworks {
		links, err := GetOfferURLs(ctx, network)
		if err != nil {
			return err
		}

		fmt.Printf("%d OfferURLs pulled from %s\n", len(*links), network)

		var failures int

		for _, efOfferURL := range *links {
			if err := ImportOfferURL(ctx, store, &efOfferURL); err != nil {
				fmt.Print(err)
				failures++
			}
		}

		if failures > 0 {
			fmt.Printf("imported OfferURLs with %d failures out of %d total\n", failures, len(*links))
		}
	}

	return nil
}
