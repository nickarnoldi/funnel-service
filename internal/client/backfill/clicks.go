package backfill

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
)

/*
Scans all RawClicks in the database to identify and correct missing or invalid data.
Repairs missing OfferLinkID fields by attempting to match URLs or using default offer URLs.
Backfills new aggregate fields such as TrafficSource, DeviceType, LanguageCode, CountryCode, and RegionCode based on existing click data.
Processes clicks in batches to efficiently handle large datasets while providing progress updates.
*/
func Clicks(ctx context.Context, store *db.Store, initialID ...int64) error {
	var afterID pgtype.Int8
	if len(initialID) > 0 && initialID[0] > 0 {
		afterID.Scan(initialID[0])
		fmt.Printf("Using starting ID %d\n", initialID[0])
	}

	var count int
	var countOfferLinkInvalid int
	var countOfferLinkNoMatch int
	var countOfferLinkCorrected int
	var countStatsAggregatesCorrected int

	var limit int32 = BackfillBatchSize

	startTime := time.Now().UTC()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Starting backfill of RawClicks...")

	for {
		if clicks, err := store.Queries.GetPaginatedRawClicks(ctx, db.GetPaginatedRawClicksParams{
			AfterID: afterID,
			Limit:   db.Int(&limit),
		}); err == nil {
			for _, click := range clicks {
				select {
				case <-ctx.Done():
					if afterID.Valid {
						fmt.Printf("\rContext cancelled, stopping backfill. Last ID: %d\n", afterID.Int64)
					}
					return nil
				case <-ticker.C:
					elapsedTime := time.Since(startTime)
					fmt.Printf("Backfilled %d clicks (OfferLinks: %d Invalid | %d No Match | %d Corrected | Stats Aggregates Corrected: %d) so far in %s\n", count, countOfferLinkInvalid, countOfferLinkNoMatch, countOfferLinkCorrected, countStatsAggregatesCorrected, elapsedTime)
				default:
				}

				// Repair OfferLinkID if missing
				if !click.OfferURLID.Valid {
					countOfferLinkInvalid++

					var offerLinkID pgtype.Int8
					// No OfferURLID; try to get the default offer URL for this offer
					if click.URL != "" {
						if urlID, err := store.Queries.GetOfferLinkIDMatchingURL(ctx, db.GetOfferLinkIDMatchingURLParams{URL: click.URL, OfferID: db.BigInt(&click.OfferID)}); err == nil {
							offerLinkID.Scan(urlID)
						} else {
							countOfferLinkNoMatch++
						}
					} else {
						if defaultLinkID, err := store.Queries.GetDefaultOfferURLIDByOfferID(ctx, click.OfferID); err == nil {
							offerLinkID.Scan(defaultLinkID)
						}
					}
					if offerLinkID.Valid {
						if _, err := store.Queries.UpdateRawClickOfferURLID(ctx, db.UpdateRawClickOfferURLIDParams{
							ID:         click.ID,
							OfferURLID: offerLinkID,
						}); err == nil {
							countOfferLinkCorrected++
						}
					}
				}

				// Backfill the new aggregate fields
				if !click.LanguageCode.Valid && !click.CountryCode.Valid && !click.RegionCode.Valid {
					countStatsAggregatesCorrected++

					var trafficSource enum.TrafficSourceEnum
					trafficSource.FromString(click.SourceID)
					if trafficSource == enum.TrafficSourceOther {
						trafficSource.FromString(click.Sub5)
						if trafficSource == enum.TrafficSourceOther {
							trafficSource.FromString(click.Referer)
						}
					}

					var deviceInfo db.DeviceInfo
					if err := json.Unmarshal(click.DeviceInfo, &deviceInfo); err != nil {
						fmt.Printf("Failed to unmarshal DeviceInfo for Click ID %d: %v\n", click.ID, err)
					}

					var deviceType enum.DeviceTypeEnum
					deviceType.FromString(click.UserAgent)
					if deviceType == enum.DeviceTypeOther {
						deviceType.FromString(deviceInfo.PlatformName)
						if deviceType == enum.DeviceTypeOther {
							deviceType.FromString(deviceInfo.Brand)
						}
					}

					var geolocation db.Geolocation
					if err := json.Unmarshal(click.Geolocation, &geolocation); err != nil {
						fmt.Printf("Failed to unmarshal Geolocation for Click ID %d: %v\n", click.ID, err)
					}

					if _, err := store.Queries.UpdateRawClickStatsAggregates(ctx, db.UpdateRawClickStatsAggregatesParams{
						ID:            click.ID,
						TrafficSource: int32(trafficSource),
						IsBot:         deviceInfo.IsRobot,
						IsMobile:      deviceInfo.IsMobile,
						IsTablet:      deviceInfo.IsTablet,
						DeviceType:    int32(deviceType),
						LanguageCode:  db.String(&deviceInfo.Language),
						CountryCode:   db.String(&geolocation.CountryCode),
						RegionCode:    db.String(&geolocation.RegionCode),
					}); err != nil {
						fmt.Printf("Failed to update RawClick stats aggregates for Click ID %d: %v\n", click.ID, err)
					}
				}

				count++
				afterID.Scan(click.ID)
			}

			if len(clicks) < int(limit) {
				break
			}
		} else {
			if err == pgx.ErrNoRows {
				break
			} else {
				return fmt.Errorf("failed to get paginated raw clicks: %w", err)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Backfilled %d clicks (OfferLinks: %d Invalid | %d No Match | %d Corrected | Stats Aggregates Corrected: %d) so far in %s\n", count, countOfferLinkInvalid, countOfferLinkNoMatch, countOfferLinkCorrected, countStatsAggregatesCorrected, elapsedTime)
	log.Println("Backfill completed")

	return nil
}
