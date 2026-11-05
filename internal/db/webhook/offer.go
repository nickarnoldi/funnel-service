package webhook

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
)

func ImportOffer(ctx context.Context, store *db.Store, efOffer *models.Offer) (*db.Offer, error) {
	if efOffer.ID == 0 {
		return nil, fmt.Errorf("skipping Offer with empty ID")
	}

	if efOffer.Name == "" {
		return nil, fmt.Errorf("skipping Offer with empty Name")
	}

	if efOffer.NetworkID == 0 {
		return nil, fmt.Errorf("skipping Offer with empty NetworkID")
	}

	var importedOffer db.Offer

	networkID := enum.NetworkEnum(efOffer.NetworkID)
	platformID := networkID.GetPlatform()

	statusInt, exists := enum.Index(enum.OfferStatus, efOffer.Status)
	if !exists {
		return nil, fmt.Errorf("unknown status %s for Offer %s, skipping", efOffer.Status, efOffer.Name)
	}

	refOfferID, err := store.GetReferenceID(ctx, platformID, "offer", int64(efOffer.ID))
	if err != nil {
		return nil, fmt.Errorf("error getting local ID for Offer %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
	}

	refCategoryID, err := store.GetReferenceID(ctx, platformID, "category", int64(efOffer.CategoryID))
	if err != nil {
		return nil, fmt.Errorf("error getting local Category ID for Offer %s with imported Category ID %d: %v", efOffer.Name, efOffer.CategoryID, err)
	}

	categoryRow, err := store.Queries.GetCategoryByReferenceID(ctx, db.BigInt(&refCategoryID))
	if err != nil {
		return nil, fmt.Errorf("error fetching Category '%d' for Offer %s: %v", efOffer.CategoryID, efOffer.Name, err)
	}

	refAdvertiserID, err := store.GetReferenceID(ctx, platformID, "advertiser", int64(efOffer.AdvertiserID))
	if err != nil {
		return nil, fmt.Errorf("error getting local Advertiser ID for Offer %s with imported Advertiser ID %d: %v", efOffer.Name, efOffer.AdvertiserID, err)
	}

	advertiserRow, err := store.Queries.GetAdvertiserByReferenceID(ctx, db.BigInt(&refAdvertiserID))
	if err != nil {
		return nil, fmt.Errorf("error fetching Advertiser for Offer %s with imported Advertiser ID %d: %v", efOffer.Name, efOffer.AdvertiserID, err)
	}

	visibilityInt, exists := enum.Index(enum.Visibility, efOffer.Visibility)
	if !exists {
		return nil, fmt.Errorf("unknown visibility %s for Offer %s, skipping", efOffer.Visibility, efOffer.Name)
	}

	if err := importedOffer.ReferenceID.Scan(refOfferID); err != nil {
		return nil, fmt.Errorf("error scanning ReferenceID for Offer %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
	}

	importedOffer.NetworkID = int32(networkID)
	importedOffer.CategoryID = categoryRow.Category.ID
	importedOffer.AdvertiserID = advertiserRow.Advertiser.ID
	importedOffer.Name = strings.TrimSpace(efOffer.Name)
	importedOffer.Status = int32(statusInt)
	importedOffer.ThumbnailURL = db.String(&efOffer.ThumbnailURL)
	importedOffer.Visibility = int32(visibilityInt)

	if offerRow, err := store.Queries.GetOfferByReferenceID(ctx, importedOffer.ReferenceID); err == nil {
		importedOffer.ID = offerRow.Offer.ID

		offer, err := store.Queries.UpdateOffer(ctx, db.UpdateOfferParams{
			ID:           importedOffer.ID,
			CategoryID:   importedOffer.CategoryID,
			AdvertiserID: importedOffer.AdvertiserID,
			Name:         importedOffer.Name,
			Status:       importedOffer.Status,
			ThumbnailURL: importedOffer.ThumbnailURL,
			Visibility:   importedOffer.Visibility,
		})
		if err != nil {
			return nil, fmt.Errorf("error updating Offer %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
		}

		url, err := store.Queries.GetOfferURLByReferenceID(ctx, importedOffer.ReferenceID)
		if err != nil {
			return nil, fmt.Errorf("error fetching Offer URL for %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
		}

		if _, err := store.Queries.UpdateOfferURL(ctx, db.UpdateOfferURLParams{
			ID:             url.ID,
			OfferID:        offerRow.Offer.ID,
			Name:           "Default",
			Status:         int32(enum.OfferURLStatusActive),
			DestinationURL: efOffer.DestinationURL,
		}); err != nil {
			return nil, fmt.Errorf("error updating Offer URL for %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
		}

		return &offer, nil
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("error fetching Offer for %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
	}

	offer, err := store.Queries.CreateOffer(ctx, db.CreateOfferParams{
		ReferenceID:  importedOffer.ReferenceID,
		NetworkID:    importedOffer.NetworkID,
		CategoryID:   importedOffer.CategoryID,
		AdvertiserID: importedOffer.AdvertiserID,
		Name:         importedOffer.Name,
		Status:       importedOffer.Status,
		ThumbnailURL: importedOffer.ThumbnailURL,
		Visibility:   importedOffer.Visibility,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Offer for %s with imported ID %d: %v", efOffer.Name, efOffer.ID, err)
	}

	if _, err = store.Queries.CreateOfferURL(ctx, db.CreateOfferURLParams{
		ReferenceID:    importedOffer.ReferenceID,
		NetworkID:      importedOffer.NetworkID,
		OfferID:        offer.ID,
		Name:           "Default",
		Status:         int32(enum.OfferURLStatusActive),
		DestinationURL: efOffer.DestinationURL,
	}); err != nil {
		log.Printf("Error creating Offer URL for %s with imported ID %d: %v\n", efOffer.Name, efOffer.ID, err)
	}

	return &offer, nil
}
