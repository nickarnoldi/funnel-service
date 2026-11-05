package webhook

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
)

func ImportAdvertiser(ctx context.Context, store *db.Store, efAdvertiser *models.Advertiser) error {
	if efAdvertiser.ID == 0 {
		return fmt.Errorf("skipping Advertiser with empty ID")
	}

	if efAdvertiser.Name == "" {
		return fmt.Errorf("skipping Advertiser with empty Name")
	}

	if efAdvertiser.NetworkID == 0 {
		return fmt.Errorf("skipping Advertiser %s with empty NetworkID", efAdvertiser.Name)
	}

	networkID := enum.NetworkEnum(efAdvertiser.NetworkID)
	platformID := networkID.GetPlatform()

	refID, err := store.GetReferenceID(ctx, platformID, "advertiser", int64(efAdvertiser.ID))
	if err != nil {
		return fmt.Errorf("error getting local ID for Advertiser %s with imported ID %d: %v", efAdvertiser.Name, efAdvertiser.ID, err)
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

	if advertiserRow, err := store.Queries.GetAdvertiserByReferenceID(ctx, db.BigInt(&refID)); err == nil {
		if _, err := store.Queries.UpdateAdvertiser(ctx, db.UpdateAdvertiserParams{
			ID:                    advertiserRow.Advertiser.ID,
			Name:                  efAdvertiser.Name,
			Status:                int32(statusInt),
			DefaultCurrency:       efAdvertiser.DefaultCurrencyID,
			AttributionMethod:     int32(attributionMethodInt),
			EmailAttibutionMethod: int32(emailAttributionMethodInt),
			AttributionPriority:   int32(attributionPriorityInt),
		}); err != nil {
			return fmt.Errorf("error updating Advertiser %s with imported ID %d: %v", efAdvertiser.Name, efAdvertiser.ID, err)
		}

		return nil
	} else if err != pgx.ErrNoRows {
		return fmt.Errorf("error fetching Advertiser for %s with imported ID %d: %v", efAdvertiser.Name, efAdvertiser.ID, err)
	}

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
		return fmt.Errorf("error creating Advertiser for %s with imported ID %d: %v", efAdvertiser.Name, efAdvertiser.ID, err)
	}

	return nil
}
