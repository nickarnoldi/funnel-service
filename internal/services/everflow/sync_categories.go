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

func ImportCategory(ctx context.Context, store *db.Store, efCategory *everflow.CategoryData) error {
	if efCategory.ID == 0 {
		return fmt.Errorf("skipping Category with empty ID")
	}

	if efCategory.Name == "" {
		return fmt.Errorf("skipping Category with empty Name")
	}

	networkID := enum.NetworkEnum(efCategory.NetworkID)
	platform := networkID.GetPlatform()
	if platform == enum.PlatformUnknown {
		return fmt.Errorf("unknown platform for Category %s with NetworkID %d, skipping", efCategory.Name, efCategory.NetworkID)
	}

	refID, err := store.GetReferenceID(ctx, platform, "category", int64(efCategory.ID))
	if err != nil {
		fmt.Printf("Error getting local ID for Category %s with imported ID %d: %v\n", efCategory.Name, efCategory.ID, err)
	}

	categoryName := strings.TrimSpace(efCategory.Name)

	statusInt, exists := enum.Index(enum.Status, efCategory.Status)
	if !exists {
		fmt.Printf("Unknown status %s for Category %s, skipping\n", efCategory.Status, categoryName)
	}

	categoryRow, err := store.Queries.GetCategoryByNetworkAndName(ctx, db.GetCategoryByNetworkAndNameParams{
		NetworkID: int32(networkID),
		Name:      categoryName,
	})
	if err == pgx.ErrNoRows {
		if _, err := store.Queries.CreateCategory(ctx, db.CreateCategoryParams{
			ReferenceID: db.BigInt(&refID),
			NetworkID:   int32(networkID),
			Name:        categoryName,
			Status:      int32(statusInt),
		}); err != nil {
			return fmt.Errorf("error creating Category for %s with imported ID %d: %w", categoryName, efCategory.ID, err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("error fetching Category for %s with imported ID %d: %w", categoryName, efCategory.ID, err)
	}
	category := categoryRow.Category

	if _, err := store.Queries.UpdateCategory(ctx, db.UpdateCategoryParams{
		ID:     category.ID,
		Name:   categoryName,
		Status: int32(statusInt),
	}); err != nil {
		return fmt.Errorf("error updating Category %s with imported ID %d: %w", categoryName, efCategory.ID, err)
	}

	return nil
}

func SyncCategories(ctx context.Context, store *db.Store, requestedNetworks ...enum.NetworkEnum) error {
	availableNetworks := enum.FilterNetwork(requestedNetworks)

	for network := range availableNetworks {
		categories, err := GetCategories(ctx, network)
		if err != nil {
			return err
		}

		fmt.Printf("%d Categories pulled from %s\n", len(*categories), network)

		var failures int

		for _, efCategory := range *categories {
			if err := ImportCategory(ctx, store, &efCategory); err != nil {
				fmt.Print(err)
				failures++
			}
		}

		if failures > 0 {
			fmt.Printf("imported categories with %d failures out of %d total\n", failures, len(*categories))
		}
	}

	return nil
}
