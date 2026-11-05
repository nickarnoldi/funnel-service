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
Scans all RawConversions in the database to identify and correct missing or invalid data.
Repairs missing ClickID fields by attempting to import the associated click from Everflow.
Repairs missing OfferLinkID fields by attempting to match URLs or using default offer URLs.
Processes conversions in batches to efficiently handle large datasets while providing progress updates.
*/
func Conversions(ctx context.Context, store *db.Store, initialID ...int64) error {
	var afterID pgtype.Int8
	if len(initialID) > 0 && initialID[0] > 0 {
		afterID.Scan(initialID[0])
		fmt.Printf("Using starting ID %d\n", initialID[0])
	}

	var count int
	var countClickInvalid int
	var countClickCorrected int
	var countOfferInvalid int
	var countOfferLinkInvalid int
	var countOfferLinkNoMatch int
	var countOfferLinkCorrected int

	var limit int32 = BackfillBatchSize

	startTime := time.Now().UTC()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Starting backfill of RawConversions...")

	for {
		if conversions, err := store.Queries.GetPaginatedRawConversions(ctx, db.GetPaginatedRawConversionsParams{
			AfterID: afterID,
			Limit:   db.Int(&limit),
		}); err == nil {
			for _, conversion := range conversions {
				select {
				case <-ctx.Done():
					if afterID.Valid {
						fmt.Printf("\rContext cancelled, stopping backfill. Last ID: %d\n", afterID.Int64)
					}
					return nil
				case <-ticker.C:
					elapsedTime := time.Since(startTime)
					fmt.Printf("Backfilled %d clicks (Offer: %d Invalid | OfferLinks: %d Invalid, %d No Match, %d Corrected | Clicks: %d Invalid, %d Corrected) so far in %s\n", count, countOfferInvalid, countOfferLinkInvalid, countOfferLinkNoMatch, countOfferLinkCorrected, countClickInvalid, countClickCorrected, elapsedTime)
				default:
				}

				// Repair ClickID if missing
				if !conversion.ClickID.Valid {
					countClickInvalid++

					time.Sleep(50 * time.Millisecond)
					if click, err := everflow.ImportRawClickByID(ctx, store, enum.NetworkEnum(conversion.NetworkID), conversion.TransactionID); err == nil {
						if click.ID == 0 {
							fmt.Printf("Imported Click has invalid ID for Conversion Transaction %s\n", conversion.TransactionID)
							continue
						}

						conversion.ClickID = db.BigInt(&click.ID)
						conversion.URL = db.String(&click.URL)
						if _, err := store.Queries.UpdateRawConversionClickID(ctx, db.UpdateRawConversionClickIDParams{
							ID:      conversion.ID,
							ClickID: conversion.ClickID,
						}); err == nil {
							countClickCorrected++
							if !conversion.OfferID.Valid {
								conversion.OfferID = db.BigInt(&click.OfferID)
								if _, err := store.Queries.UpdateRawConversionOfferID(ctx, db.UpdateRawConversionOfferIDParams{
									ID:      conversion.ID,
									OfferID: conversion.OfferID,
								}); err != nil {
									fmt.Printf("Failed to update OfferID for Conversion ID %d: %v\n", conversion.ID, err)
								}

								if !conversion.OfferURLID.Valid && click.OfferURLID.Valid {
									conversion.OfferURLID = click.OfferURLID
									if _, err := store.Queries.UpdateRawConversionOfferURLID(ctx, db.UpdateRawConversionOfferURLIDParams{
										ID:         conversion.ID,
										OfferURLID: conversion.OfferURLID,
									}); err != nil {
										fmt.Printf("Failed to update OfferURLID for Conversion ID %d: %v\n", conversion.ID, err)
									}
								}
							}
						}
					}
				}

				if !conversion.OfferID.Valid {
					countOfferInvalid++

					fmt.Printf("Invalid Offer ID for Transaction %s\n", conversion.TransactionID)
					continue
				}

				// Repair OfferLinkID if missing
				if !conversion.OfferURLID.Valid {
					countOfferLinkInvalid++

					// No OfferURLID; try to get the ID from the Click or fallback to the pattern matching/default offer URL for this offer
					var offerURLID pgtype.Int8
					if conversion.ClickOfferURLID.Valid {
						offerURLID = conversion.ClickOfferURLID
					} else if conversion.URL.Valid && conversion.URL.String != "" {
						if urlID, err := store.Queries.GetOfferLinkIDMatchingURL(ctx, db.GetOfferLinkIDMatchingURLParams{URL: conversion.URL.String, OfferID: db.BigInt(&conversion.OfferID.Int64)}); err == nil {
							offerURLID.Scan(urlID)
						} else {
							countOfferLinkNoMatch++
						}
					} else {
						if defaultLinkID, err := store.Queries.GetDefaultOfferURLIDByOfferID(ctx, conversion.OfferID.Int64); err == nil {
							offerURLID.Scan(defaultLinkID)
						}
					}
					if offerURLID.Valid {
						if _, err := store.Queries.UpdateRawConversionOfferURLID(ctx, db.UpdateRawConversionOfferURLIDParams{
							ID:         conversion.ID,
							OfferURLID: offerURLID,
						}); err == nil {
							countOfferLinkCorrected++
						}
					}
				}

				count++
				afterID.Scan(conversion.ID)
			}

			if len(conversions) < int(limit) {
				break
			}
		} else {
			if err == pgx.ErrNoRows {
				break
			} else {
				return fmt.Errorf("failed to get paginated raw conversions: %w", err)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Backfilled %d clicks (Offer: %d Invalid | OfferLinks: %d Invalid, %d No Match, %d Corrected | Clicks: %d Invalid, %d Corrected) in %s\n", count, countOfferInvalid, countOfferLinkInvalid, countOfferLinkNoMatch, countOfferLinkCorrected, countClickInvalid, countClickCorrected, elapsedTime)
	log.Println("Backfill completed")

	return nil
}
