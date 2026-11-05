package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/constants"
	"github.com/nickarnoldi/funnel-service/internal/db"
)

type HourlyStatsParams struct {
	From     time.Time
	Backfill bool
}

type ProcessStatsResult struct {
	CountInserts      int
	CountUpdates      int
	CountMissingOffer int
}

func ProcessHourlyStats(ctx context.Context, store *db.Store, params ...HourlyStatsParams) error {
	if len(params) > 0 && params[0].Backfill {
		return backfillHourlyStats(ctx, store, params[0])
	}

	reportDate, err := prepareDate(ctx, store, params...)
	if err != nil {
		return err
	}

	result, err := processStats(ctx, store, reportDate)
	if err != nil {
		return fmt.Errorf("failed to backfill click stats: %w", err)
	}

	fmt.Printf("Processed %d (%d new) Stats for %s\n", result.CountInserts+result.CountUpdates, result.CountInserts, reportDate.Time.Format("2006-01-02 15:00"))
	if result.CountMissingOffer > 0 {
		fmt.Printf("  - Skipped %d Stats missing Offer\n", result.CountMissingOffer)
	}

	return nil
}

func ProcessHourlyStatsByFunnelIDs(ctx context.Context, store *db.Store, funnelIDs []int64) (ProcessStatsResult, error) {
	if len(funnelIDs) == 0 {
		return ProcessStatsResult{}, fmt.Errorf("no funnel IDs provided for processing hourly stats")
	}

	var queryDate pgtype.Date
	var results ProcessStatsResult

	if first, err := store.Queries.GetFirstRawClickTimestamp(ctx); err == nil && first.Valid {
		queryDate.Scan(first.Time)
	} else {
		queryDate.Scan(constants.FunnelLoggingStartDate)
	}

	fmt.Printf("Starting stats sync for funnels: %v\n", funnelIDs)

	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-ticker.C:
			fmt.Printf(
				"Processed %d (%d new) Stats. Working %s (duration: %s)\n",
				results.CountInserts+results.CountUpdates,
				results.CountInserts,
				queryDate.Time.Format("2006-01-02"),
				time.Since(startTime).Truncate(time.Second),
			)
		default:
		}

		if stats, err := store.Queries.GenerateHourlyStatsByFunnelIDs(ctx, db.GenerateHourlyStatsByFunnelIDsParams{
			FunnelIDs: funnelIDs,
			Date:      queryDate,
		}); err == nil {
			for _, stat := range stats {
				select {
				case <-ctx.Done():
					return results, ctx.Err()
				case <-ticker.C:
					fmt.Printf(
						"Processed %d (%d new) Stats. Working %s (duration: %s)\n",
						results.CountInserts+results.CountUpdates,
						results.CountInserts,
						queryDate.Time.Format("2006-01-02"),
						time.Since(startTime).Truncate(time.Second),
					)
				default:
				}

				if !stat.OfferURLID.Valid {
					results.CountMissingOffer++
					continue
				}

				if statID, err := store.Queries.GetStatsHourlyID(ctx, db.GetStatsHourlyIDParams{
					SliceHour:     stat.SliceHour,
					OfferID:       stat.OfferID,
					OfferURLID:    stat.OfferURLID.Int64,
					CategoryID:    stat.CategoryID,
					AdvertiserID:  stat.AdvertiserID,
					AffiliateID:   stat.AffiliateID,
					TrafficSource: stat.TrafficSource,
					IsUnique:      stat.IsUnique,
					IsBot:         stat.IsBot,
					IsMobile:      stat.IsMobile,
					IsTablet:      stat.IsTablet,
					DeviceType:    stat.DeviceType,
					LanguageCode:  stat.LanguageCode,
					CountryCode:   stat.CountryCode,
					RegionCode:    stat.RegionCode,
				}); err == nil {
					if _, err := store.Queries.UpdateStatsHourly(ctx, db.UpdateStatsHourlyParams{
						ID:          statID,
						Clicks:      stat.Clicks,
						Conversions: stat.Conversions,
					}); err != nil {
						return results, fmt.Errorf("failed to update stats hourly for hour %v and offer %v: %w", stat.SliceHour, stat.OfferID, err)
					}

					results.CountUpdates++
					continue
				} else if err != pgx.ErrNoRows {
					return results, fmt.Errorf("failed to get stats hourly ID for hour %v and offer %v: %w", stat.SliceHour, stat.OfferID, err)
				}

				if _, err := store.Queries.CreateStatsHourly(ctx, db.CreateStatsHourlyParams{
					SliceHour:     stat.SliceHour,
					OfferID:       stat.OfferID,
					OfferURLID:    stat.OfferURLID.Int64,
					CategoryID:    stat.CategoryID,
					AdvertiserID:  stat.AdvertiserID,
					AffiliateID:   stat.AffiliateID,
					TrafficSource: stat.TrafficSource,
					IsUnique:      stat.IsUnique,
					IsBot:         stat.IsBot,
					IsMobile:      stat.IsMobile,
					IsTablet:      stat.IsTablet,
					DeviceType:    stat.DeviceType,
					LanguageCode:  stat.LanguageCode,
					CountryCode:   stat.CountryCode,
					RegionCode:    stat.RegionCode,
					Clicks:        stat.Clicks,
					Conversions:   stat.Conversions,
				}); err != nil {
					return results, fmt.Errorf("failed to create stats hourly for hour %v and offer %v: %w", stat.SliceHour, stat.OfferID, err)
				}
				results.CountInserts++
			}
		}

		nextDate := queryDate.Time.AddDate(0, 0, 1)
		if nextDate.After(time.Now()) {
			break
		}
		queryDate.Scan(nextDate)
	}

	return results, nil
}

func backfillHourlyStats(ctx context.Context, store *db.Store, params HourlyStatsParams) error {
	reportDate, err := prepareDate(ctx, store, params)
	if err != nil {
		return err
	}

	fmt.Printf("Backfilling Stats for: %s\n", reportDate.Time)

	var statsResults ProcessStatsResult

	startTime := time.Now()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			fmt.Printf(
				"Processed %d (%d new) Stats. Working %s (duration: %s)\n",
				statsResults.CountInserts+statsResults.CountUpdates,
				statsResults.CountInserts,
				reportDate.Time.Format("2006-01-02"),
				time.Since(startTime).Truncate(time.Second),
			)
		default:
		}

		result, err := processStats(ctx, store, reportDate)
		if err != nil {
			return fmt.Errorf("failed to backfill click stats: %w", err)
		}

		statsResults.CountInserts += result.CountInserts
		statsResults.CountUpdates += result.CountUpdates
		statsResults.CountMissingOffer += result.CountMissingOffer

		reportDate.Time = reportDate.Time.Add(24 * time.Hour)
		if reportDate.Time.After(time.Now()) {
			break
		}
	}

	fmt.Printf(
		"Finished Backfill %d (%d new) Stats in %s.\n",
		statsResults.CountInserts+statsResults.CountUpdates,
		statsResults.CountInserts,
		time.Since(startTime).Truncate(time.Second),
	)

	if statsResults.CountMissingOffer > 0 {
		fmt.Printf("  - Skipped %d Stats missing Offer\n", statsResults.CountMissingOffer)
	}

	return nil
}

func processStats(ctx context.Context, store *db.Store, date pgtype.Date) (ProcessStatsResult, error) {
	var result ProcessStatsResult

	generatedStats, err := store.Queries.GenerateHourlyStats(ctx, date)
	if err == nil {
		for _, stat := range generatedStats {
			if !stat.OfferURLID.Valid {
				result.CountMissingOffer++
				continue
			}

			if statID, err := store.Queries.GetStatsHourlyID(ctx, db.GetStatsHourlyIDParams{
				SliceHour:     stat.SliceHour,
				OfferID:       stat.OfferID,
				OfferURLID:    stat.OfferURLID.Int64,
				CategoryID:    stat.CategoryID,
				AdvertiserID:  stat.AdvertiserID,
				AffiliateID:   stat.AffiliateID,
				TrafficSource: stat.TrafficSource,
				IsUnique:      stat.IsUnique,
				IsBot:         stat.IsBot,
				IsMobile:      stat.IsMobile,
				IsTablet:      stat.IsTablet,
				DeviceType:    stat.DeviceType,
				LanguageCode:  stat.LanguageCode,
				CountryCode:   stat.CountryCode,
				RegionCode:    stat.RegionCode,
			}); err == nil {
				if _, err := store.Queries.UpdateStatsHourly(ctx, db.UpdateStatsHourlyParams{
					ID:          statID,
					Clicks:      stat.Clicks,
					Conversions: stat.Conversions,
				}); err != nil {
					return result, fmt.Errorf("failed to update stats hourly for hour %v and offer %v: %w", stat.SliceHour, stat.OfferID, err)
				}

				result.CountUpdates++
				continue
			} else if err != pgx.ErrNoRows {
				return result, fmt.Errorf("failed to get stats hourly ID for hour %v and offer %v: %w", stat.SliceHour, stat.OfferID, err)
			}

			if _, err := store.Queries.CreateStatsHourly(ctx, db.CreateStatsHourlyParams{
				SliceHour:     stat.SliceHour,
				OfferID:       stat.OfferID,
				OfferURLID:    stat.OfferURLID.Int64,
				CategoryID:    stat.CategoryID,
				AdvertiserID:  stat.AdvertiserID,
				AffiliateID:   stat.AffiliateID,
				TrafficSource: stat.TrafficSource,
				IsUnique:      stat.IsUnique,
				IsBot:         stat.IsBot,
				IsMobile:      stat.IsMobile,
				IsTablet:      stat.IsTablet,
				DeviceType:    stat.DeviceType,
				LanguageCode:  stat.LanguageCode,
				CountryCode:   stat.CountryCode,
				RegionCode:    stat.RegionCode,
				Clicks:        stat.Clicks,
				Conversions:   stat.Conversions,
			}); err != nil {
				return result, fmt.Errorf("failed to create stats hourly for hour %v and offer %v: %w", stat.SliceHour, stat.OfferID, err)
			}
			result.CountInserts++
		}

		return result, nil
	}

	return result, err
}

func prepareDate(ctx context.Context, store *db.Store, params ...HourlyStatsParams) (pgtype.Date, error) {
	var fromDate pgtype.Date

	if paramsCount := len(params); paramsCount > 0 {
		if paramsCount > 1 {
			return pgtype.Date{}, fmt.Errorf("only one HourlyStatsParams is supported")
		}
		fromDate.Scan(params[0].From)
	}

	if !fromDate.Valid || fromDate.Time.IsZero() {
		var from time.Time
		if last, err := store.Queries.GetLastStatsHourlySliceHour(ctx); err == nil && last.Valid {
			from = last.Time.Add(time.Hour)
			if from.After(time.Now()) {
				from = last.Time
			}

			if next, err := store.Queries.GetNextRawClickTimestamp(ctx, pgtype.Timestamp{Time: from, Valid: true}); err == nil && next.Valid {
				from = next.Time.Truncate(time.Hour)
				fromDate.Scan(from)
			} else {
				// If there is no future timestamp, we are up to date - set from to last hour
				from = time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Hour)
				fromDate.Scan(from)
			}
		} else {
			fmt.Printf("Unable to locate previous progress.\n")
			if first, err := store.Queries.GetFirstRawClickTimestamp(ctx); err == nil && first.Valid {
				from = first.Time.Truncate(time.Hour)
				fromDate.Scan(from)
			} else {
				return pgtype.Date{}, fmt.Errorf("failed to get last stats hourly clicks slice hour: %v", first)
			}
		}
	}

	if !fromDate.Valid || fromDate.Time.IsZero() {
		return pgtype.Date{}, fmt.Errorf("invalid 'from' date for processing click stats: %v", fromDate)
	}

	return fromDate, nil
}
