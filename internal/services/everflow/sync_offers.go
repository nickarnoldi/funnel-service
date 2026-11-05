package everflow

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
)

func ImportOffer(ctx context.Context, store *db.Store, efOffer *everflow.OfferData) error {
	if efOffer.ID == 0 {
		return fmt.Errorf("skipping Offer with empty ID")
	}

	if efOffer.Name == "" {
		return fmt.Errorf("skipping Offer with empty Name")
	}

	networkID := enum.NetworkEnum(efOffer.NetworkID)
	platform := networkID.GetPlatform()

	if platform == enum.PlatformUnknown {
		fmt.Printf("Unknown platform for Offer %s with NetworkID %d, skipping\n", efOffer.Name, efOffer.NetworkID)
		return nil
	}

	statusInt, exists := enum.Index(enum.OfferStatus, efOffer.Status)
	if !exists {
		return fmt.Errorf("unknown status %s for Offer %s, skipping", efOffer.Status, efOffer.Name)
	}

	refID, err := store.GetReferenceID(ctx, platform, "offer", int64(efOffer.ID))
	if err != nil {
		return fmt.Errorf("error getting local ID for Offer %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
	}

	categoryRow, err := store.Queries.GetCategoryByNetworkAndName(ctx, db.GetCategoryByNetworkAndNameParams{
		NetworkID: int32(efOffer.NetworkID),
		Name:      strings.TrimSpace(efOffer.Category),
	})
	if err != nil {
		return fmt.Errorf("error fetching Category '%s' for Offer %s: %v", efOffer.Category, efOffer.Name, err)
	}
	category := categoryRow.Category

	refAdvertiserID, err := store.GetReferenceID(ctx, platform, "advertiser", int64(efOffer.NetworkAdvertiserID))
	if err != nil {
		return fmt.Errorf("error getting local Advertiser ID for Offer %s with imported Advertiser ID %d: %v", efOffer.Name, efOffer.NetworkAdvertiserID, err)
	}

	advertiser, err := store.Queries.GetAdvertiserByReferenceID(ctx, db.BigInt(&refAdvertiserID))
	if err != nil {
		return fmt.Errorf("error fetching Advertiser for Offer %s with imported Advertiser ID %d: %v", efOffer.Name, efOffer.NetworkAdvertiserID, err)
	}

	visibilityInt, exists := enum.Index(enum.Visibility, efOffer.Visibility)
	if !exists {
		return fmt.Errorf("unknown visibility %s for Offer %s, skipping", efOffer.Visibility, efOffer.Name)
	}

	offerRow, err := store.Queries.GetOfferByReferenceID(ctx, db.BigInt(&refID))
	if err == pgx.ErrNoRows {
		offer, err := store.Queries.CreateOffer(ctx, db.CreateOfferParams{
			ReferenceID:  db.BigInt(&refID),
			NetworkID:    int32(efOffer.NetworkID),
			CategoryID:   category.ID,
			AdvertiserID: advertiser.Advertiser.ID,
			Name:         strings.TrimSpace(efOffer.Name),
			Status:       int32(statusInt),
			ThumbnailURL: db.String(&efOffer.ThumbnailURL),
			Visibility:   int32(visibilityInt),
		})
		if err != nil {
			fmt.Printf("Error creating Offer for %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
			return nil
		}

		if _, err = store.Queries.CreateOfferURL(ctx, db.CreateOfferURLParams{
			ReferenceID:    db.BigInt(&refID),
			NetworkID:      int32(efOffer.NetworkID),
			OfferID:        offer.ID,
			Name:           "Default",
			Status:         int32(enum.OfferURLStatusActive),
			DestinationURL: efOffer.DestinationURL,
		}); err != nil {
			fmt.Printf("Error creating Offer URL for %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
		}

		return nil
	} else if err != nil {
		fmt.Printf("Error fetching Offer for %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
		return nil
	}
	offer := offerRow.Offer

	if _, err := store.Queries.UpdateOffer(ctx, db.UpdateOfferParams{
		ID:           offer.ID,
		CategoryID:   category.ID,
		AdvertiserID: advertiser.Advertiser.ID,
		Name:         strings.TrimSpace(efOffer.Name),
		Status:       int32(statusInt),
		ThumbnailURL: db.String(&efOffer.ThumbnailURL),
		Visibility:   int32(visibilityInt),
	}); err != nil {
		fmt.Printf("Error updating Offer %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
		return nil
	}

	url, err := store.Queries.GetOfferURLByReferenceID(ctx, db.BigInt(&refID))
	if err != nil {
		fmt.Printf("Error fetching Offer URL for %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
		return nil
	}

	if _, err := store.Queries.UpdateOfferURL(ctx, db.UpdateOfferURLParams{
		ID:             url.ID,
		OfferID:        offer.ID,
		FunnelID:       url.FunnelID,
		Name:           "Default",
		Status:         int32(enum.OfferURLStatusActive),
		DestinationURL: efOffer.DestinationURL,
		// No PreviewURL on the default link
	}); err != nil {
		fmt.Printf("Error updating Offer URL for %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
	}

	return nil
}

func SyncOffers(ctx context.Context, store *db.Store, requestedNetworks ...enum.NetworkEnum) error {
	availableNetworks := enum.FilterNetwork(requestedNetworks)

	for network := range availableNetworks {
		offers, err := GetOffers(ctx, network)
		if err != nil {
			return err
		}

		fmt.Printf("%d Offers pulled from %s\n", len(*offers), network)

		var failures int

		for _, efOffer := range *offers {
			if err := ImportOffer(ctx, store, &efOffer); err != nil {
				log.Print(err)
				failures++
			}
		}

		if failures > 0 {
			fmt.Printf("imported offers with %d failures out of %d total\n", failures, len(*offers))
		}
	}

	return nil
}
