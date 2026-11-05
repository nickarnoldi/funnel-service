package backfill

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow"
)

/*
Locates abandoned leads missing click associations within a specified date range and attempts to find and link the corresponding clicks.
If a matching click is found based on the transaction ID and offer ID, the lead is updated accordingly.
If no matching click is found, an attempt is made to import the click from Everflow.
*/
func LeadsMissingClicks(ctx context.Context, store *db.Store, startDate, endDate time.Time) error {
	parsedStartDate, parsedEndDate, err := store.GenerateSafeDateRange(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to generate safe date range: %w", err)
	}

	var startRange, endRange pgtype.Timestamp
	startRange.Scan(parsedStartDate)
	endRange.Scan(parsedEndDate)

	count, err := store.Queries.CountAbandonedLeadsMissingClicksByDateRange(ctx, db.CountAbandonedLeadsMissingClicksByDateRangeParams{
		From: startRange,
		To:   endRange,
	})
	if err != nil {
		return fmt.Errorf("failed to count abandoned leads missing clicks: %w", err)
	}

	fmt.Printf("Syncing %d abandoned leads missing clicks from %s to %s\n", count, parsedStartDate, parsedEndDate)

	var offset int = 0
	var limit int = 100

	var numberTotal int
	var numberCorrected int

	for {
		if leads, err := store.Queries.GetAbandonedLeadsMissingClicksByDateRange(ctx, db.GetAbandonedLeadsMissingClicksByDateRangeParams{
			From:   startRange,
			To:     endRange,
			Offset: db.Int(&offset),
			Limit:  db.Int(&limit),
		}); err == nil {
			leadCount := len(leads)
			numberTotal += leadCount
			var correctedThisBatch int

			for _, lead := range leads {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// Process each lead
				if lead.TransactionID != "" && len(lead.TransactionID) >= 20 {
					for network := range enum.Network {
						if click, err := store.Queries.GetRawClickByNetworkAndTransactionID(ctx, db.GetRawClickByNetworkAndTransactionIDParams{
							NetworkID:     int32(network),
							TransactionID: lead.TransactionID,
						}); err == nil {
							if click.OfferID == lead.OfferID {
								fmt.Printf(" => Found matching click %d for lead %d\n", click.ID, lead.ID)
								if _, err := store.Queries.UpdateAbandonedLeadClickID(ctx, db.UpdateAbandonedLeadClickIDParams{
									ID:      lead.ID,
									ClickID: click.ID,
								}); err == nil {
									correctedThisBatch++
								}
							} else {
								if !click.OfferURLID.Valid {
									log.Printf("Click %s has an invalid OfferURLID", click.TransactionID)
									break
								}

								// Update Click and Offer ID
								if _, err := store.Queries.UpdateAbandonedLeadClickOfferAndOfferURL(ctx, db.UpdateAbandonedLeadClickOfferAndOfferURLParams{
									ID:         lead.ID,
									ClickID:    click.ID,
									OfferID:    click.OfferID,
									OfferURLID: click.OfferURLID.Int64,
								}); err == nil {
									correctedThisBatch++
								}
							}

							break
						}

						click, err := everflow.ImportRawClickByID(ctx, store, enum.NetworkEnum(lead.NetworkID.Int32), lead.TransactionID)
						if err == nil {
							fmt.Printf(" => Found imported click %d for lead %d\n", click.ID, lead.ID)
							if _, err := store.Queries.UpdateAbandonedLeadClickID(ctx, db.UpdateAbandonedLeadClickIDParams{
								ID:      lead.ID,
								ClickID: click.ID,
							}); err == nil {
								correctedThisBatch++
							}
						}

						time.Sleep(250 * time.Millisecond)
					}
				}
			}

			offset += leadCount
			numberCorrected += correctedThisBatch

			if leadCount != limit {
				break
			}

			time.Sleep(50 * time.Millisecond)
		} else if err == pgx.ErrNoRows {
			break
		} else {
			return fmt.Errorf("failed to get abandoned leads missing clicks: %w", err)
		}
	}

	if numberCorrected > 0 {
		fmt.Printf("Updated %d abandoned leads with missing clicks\n\n", numberCorrected)
	} else {
		fmt.Printf("%d abandoned leads with missing clicks found\n\n", numberCorrected)
	}

	return nil
}
