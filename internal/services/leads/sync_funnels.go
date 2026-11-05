package leads

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nickarnoldi/funnel-service/internal/db"
)

/*
SyncFunnels synchronizes funnels from the database, processing their abandoned leads
to ensure all funnel fields are up to date based on the lead data.
If specific IDs are provided, only those funnels are processed.
Otherwise, all funnels in the database are processed.
*/
func SyncFunnels(ctx context.Context, store *db.Store, IDs ...int64) error {
	var funnels []db.Funnel
	var err error

	if len(IDs) > 0 {
		funnels, err = store.Queries.GetFunnelsByID(ctx, IDs)
		if err != nil {
			return err
		}
	} else {
		funnels, err = store.Queries.GetAllFunnels(ctx)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Syncing %d funnels\n", len(funnels))

	for _, funnel := range funnels {
		if !funnel.AutoImport {
			continue
		}

		fields, err := store.Queries.GetFunnelFieldsByFunnelID(ctx, funnel.ID)
		if err != nil {
			return err
		}

		leads, err := store.Queries.GetRecentAbandonedLeadBodysByFunnelID(ctx, db.GetRecentAbandonedLeadBodysByFunnelIDParams{
			ID: funnel.ID,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Funnel %d (%s) has %d leads\n", funnel.ID, funnel.Name.String, len(leads))
		for _, body := range leads {
			var bodyMap map[string]any
			if err := json.Unmarshal(body, &bodyMap); err != nil {
				return err
			}

			// Locate any fields in the body that are not already in the funnel fields
			for bodyField := range bodyMap {
				found := false
				for _, field := range fields {
					if field.Name == bodyField {
						found = true
						break
					}
				}

				if !found {
					field, err := store.Queries.CreateFunnelField(ctx, db.CreateFunnelFieldParams{
						FunnelID: funnel.ID,
						Name:     bodyField,
					})
					if err != nil {
						return err
					}
					fmt.Printf("Created funnel field %d (%s) for funnel %d\n", field.ID, field.Name, funnel.ID)

					fields = append(fields, field)
				}
			}
		}
	}

	return nil
}
