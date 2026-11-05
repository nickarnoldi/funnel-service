package webhook

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
)

func ImportAffiliate(ctx context.Context, store *db.Store, efAffiliate *models.Affiliate) error {
	if efAffiliate.ID == 0 {
		return fmt.Errorf("skipping Affiliate with empty ID")
	}

	if efAffiliate.Name == "" {
		return fmt.Errorf("skipping Affiliate with empty Name")
	}

	if efAffiliate.NetworkID == 0 {
		return fmt.Errorf("skipping Affiliate %s with empty NetworkID", efAffiliate.Name)
	}

	networkID := enum.NetworkEnum(efAffiliate.NetworkID)
	platformID := networkID.GetPlatform()

	refID, err := store.GetReferenceID(ctx, platformID, "affiliate", int64(efAffiliate.ID))
	if err != nil {
		return fmt.Errorf("error getting local ID for Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
	}

	status, found := enum.Index(enum.Status, efAffiliate.Status)
	if !found {
		return fmt.Errorf("unknown status %s for Affiliate %s", efAffiliate.Status, efAffiliate.Name)
	}

	if affiliateRow, err := store.Queries.GetAffiliateByReferenceID(ctx, db.BigInt(&refID)); err == nil {
		if _, err := store.Queries.UpdateAffiliate(ctx, db.UpdateAffiliateParams{
			ID:     affiliateRow.Affiliate.ID,
			Name:   efAffiliate.Name,
			Status: int32(status),
		}); err != nil {
			return fmt.Errorf("error updating Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
		}

		return nil
	} else if err != pgx.ErrNoRows {
		return fmt.Errorf("error fetching Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
	}

	if _, err = store.Queries.CreateAffiliate(ctx, db.CreateAffiliateParams{
		ReferenceID: db.BigInt(&refID),
		NetworkID:   int32(networkID),
		Name:        efAffiliate.Name,
		Status:      int32(status),
	}); err != nil {
		return fmt.Errorf("error creating Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
	}

	return nil
}
