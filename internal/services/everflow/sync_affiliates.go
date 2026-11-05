package everflow

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
)

func ImportAffiliate(ctx context.Context, store *db.Store, efAffiliate *everflow.AffiliateData) error {
	if efAffiliate.ID == 0 {
		return fmt.Errorf("skipping Affiliate with empty ID")
	}

	if efAffiliate.Name == "" {
		return fmt.Errorf("skipping Affiliate with empty Name")
	}

	networkID := enum.NetworkEnum(efAffiliate.NetworkID)
	platform := networkID.GetPlatform()
	if platform == enum.PlatformUnknown {
		return fmt.Errorf("unknown platform for Affiliate %s with NetworkID %d", efAffiliate.Name, efAffiliate.NetworkID)
	}

	refID, err := store.GetReferenceID(ctx, platform, "affiliate", int64(efAffiliate.ID))
	if err != nil {
		return fmt.Errorf("error getting local ID for Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
	}

	status, found := enum.Index(enum.Status, efAffiliate.AccountStatus)
	if !found {
		return fmt.Errorf("unknown status %s for Affiliate %s", efAffiliate.AccountStatus, efAffiliate.Name)
	}

	affiliateRow, err := store.Queries.GetAffiliateByReferenceID(ctx, db.BigInt(&refID))
	if err == pgx.ErrNoRows {
		if _, err := store.Queries.CreateAffiliate(ctx, db.CreateAffiliateParams{
			ReferenceID: db.BigInt(&refID),
			NetworkID:   int32(networkID),
			Name:        efAffiliate.Name,
			Status:      int32(status),
		}); err != nil {
			return fmt.Errorf("error creating Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("error fetching Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
	}
	affiliate := affiliateRow.Affiliate

	if _, err := store.Queries.UpdateAffiliate(ctx, db.UpdateAffiliateParams{
		ID:     affiliate.ID,
		Name:   efAffiliate.Name,
		Status: int32(status),
	}); err != nil {
		return fmt.Errorf("error updating Affiliate %s with imported ID %d: %w", efAffiliate.Name, efAffiliate.ID, err)
	}

	return nil
}

func SyncAffiliates(ctx context.Context, store *db.Store, requestedNetworks ...enum.NetworkEnum) error {
	availableNetworks := enum.FilterNetwork(requestedNetworks)

	for network := range availableNetworks {
		affiliates, err := GetAffiliates(ctx, network)
		if err != nil {
			return err
		}

		fmt.Printf("%d Affiliates pulled from %s\n", len(*affiliates), network)

		var failures int

		for _, efAffiliate := range *affiliates {
			if err := ImportAffiliate(ctx, store, &efAffiliate); err != nil {
				log.Print(err)
				failures++
			}
		}

		if failures > 0 {
			fmt.Printf("imported affiliates with %d failures out of %d total\n", failures, len(*affiliates))
		}
	}

	return nil
}
