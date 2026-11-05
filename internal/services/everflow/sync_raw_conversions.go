package everflow

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/constants"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
)

func SyncRawConversions(ctx context.Context, store *db.Store, requestedStartDate, requestedEndDate time.Time, requestedNetworks ...enum.NetworkEnum) error {
	availableNetworks := enum.FilterNetwork(requestedNetworks)

	for network, networkName := range availableNetworks {
		startDate := requestedStartDate
		endDate := requestedEndDate

		if startDate.IsZero() {
			lastConversionUpdate, err := store.Queries.GetLastRawConversionTimestampByNetwork(ctx, int32(network))
			if err != nil && err != pgx.ErrNoRows {
				return fmt.Errorf("failed to get last conversion timestamp: %w", err)
			}

			if !lastConversionUpdate.Valid {
				startDate = constants.FunnelLoggingStartDate
			} else {
				fmt.Printf("Last %s Conversion Timestamp: %s\n", networkName, lastConversionUpdate.Time)
				startDate = lastConversionUpdate.Time
			}
		}

		syncRangeStartDate, syncRangeEndDate, err := store.GenerateSafeDateRange(startDate, endDate)
		if err != nil {
			return fmt.Errorf("failed to get date range: %w", err)
		}

		fmt.Printf("Syncing %s Conversions from %s to %s...\n", networkName, syncRangeStartDate.Format("2006-01-02 15:04:05"), syncRangeEndDate.Format("2006-01-02 15:04:05"))

		conversions, err := GetRawConversions(ctx, network, syncRangeStartDate, syncRangeEndDate)
		if err != nil {
			return fmt.Errorf("failed to get raw conversions: %w", err)
		}
		count := len(*conversions)

		fmt.Printf("Downloaded %d %s Conversions from %s to %s.\n", count, networkName, syncRangeStartDate.Format("2006-01-02 15:04:05"), syncRangeEndDate.Format("2006-01-02 15:04:05"))

		var failures int
		var clickMisses int

		for _, efConversion := range *conversions {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			result, err := ImportConversion(ctx, store, &efConversion)
			if err != nil {
				failures++
				continue
			}
			if result.ClickMissed {
				clickMisses++
			}
		}

		if failures > 0 {
			return fmt.Errorf("imported clicks with %d failures out of %d total", failures, len(*conversions))
		}

		fmt.Printf("Synced %d %s Conversions (%d missing Clicks) from %s to %s.\n", count, networkName, clickMisses, syncRangeStartDate.Format("2006-01-02 15:04:05"), syncRangeEndDate.Format("2006-01-02 15:04:05"))
	}

	return nil
}

type ImportConversionResult struct {
	Conversion  *db.RawConversion
	Existing    bool
	ClickMissed bool
}

func ImportConversion(ctx context.Context, store *db.Store, conversionData *everflow.ConversionData) (ImportConversionResult, error) {
	var result ImportConversionResult

	if conversionData.TransactionID == "" {
		return result, fmt.Errorf("skipping Conversion with empty TransactionID")
	}

	networkID := enum.NetworkEnum(conversionData.Relationship.Offer.NetworkID)
	platform := networkID.GetPlatform()

	if platform == enum.PlatformUnknown {
		return result, fmt.Errorf("error getting Platform for Conversion with transaction ID %s and NetworkID %d", conversionData.TransactionID, conversionData.Relationship.Offer.NetworkID)
	}

	status, found := enum.Index(enum.ConversionStatus, conversionData.Status)
	if !found {
		return result, fmt.Errorf("error getting Status for Conversion with transaction ID %s", conversionData.TransactionID)
	}

	var createdAt pgtype.Timestamp
	if err := createdAt.Scan(time.Unix(conversionData.ConversionUnixTimestamp, 0).In(time.UTC)); err != nil {
		return result, fmt.Errorf("error scanning CreatedAt %d for Conversion with transaction ID %s: %v", conversionData.ConversionUnixTimestamp, conversionData.TransactionID, err)
	}

	offerRow, err := store.Queries.GetOfferByExternalReferenceID(ctx, db.GetOfferByExternalReferenceIDParams{
		PlatformID: int32(platform),
		ExternalID: int64(conversionData.Relationship.Offer.NetworkOfferID),
	})
	if err != nil {
		return result, fmt.Errorf("error getting Offer %s with imported ID %d: %v", conversionData.Relationship.Offer.Name, conversionData.Relationship.Offer.NetworkOfferID, err)
	}

	offer := offerRow.Offer

	var offerID pgtype.Int8
	offerID.Scan(offer.ID)

	var offerLinkID pgtype.Int8
	if conversionData.URL == "" {
		if defaultLinkID, err := store.Queries.GetDefaultOfferURLIDByOfferID(ctx, offer.ID); err == nil {
			offerLinkID.Scan(defaultLinkID)
		}
	} else {
		// OfferURL is not present in the conversions stream model, so we try to look it up by URL
		if urlID, err := store.Queries.GetOfferURLIDByURL(ctx, conversionData.URL); err == nil {
			offerLinkID.Scan(urlID)
		}
	}

	payoutModel, _ := enum.Index(enum.PayoutModel, strings.ToUpper(conversionData.PayoutType))
	revenueModel, _ := enum.Index(enum.RevenueModel, strings.ToUpper(conversionData.RevenueType))

	payoutAmount := fmt.Sprintf("%.2f", conversionData.Payout)
	revenueAmount := fmt.Sprintf("%.2f", conversionData.Revenue)

	var clickID pgtype.Int8
	if click, err := store.Queries.GetRawClickByNetworkAndTransactionID(ctx, db.GetRawClickByNetworkAndTransactionIDParams{
		NetworkID:     int32(networkID),
		TransactionID: conversionData.TransactionID,
	}); err == nil {
		clickID.Scan(click.ID)
	} else {
		result.ClickMissed = true
		if err != pgx.ErrNoRows {
			return result, fmt.Errorf("error getting RawClick for Conversion with transaction ID %s: %v", conversionData.TransactionID, err)
		}
	}

	if conversion, err := store.Queries.GetRawConversionByNetworkAndConversionID(ctx, db.GetRawConversionByNetworkAndConversionIDParams{
		NetworkID:    int32(networkID),
		ConversionID: conversionData.ConversionID,
	}); err == nil {
		if offerLinkID.Valid && !conversion.OfferURLID.Valid {
			log.Printf("Updating offerLinkID for conversion %s", conversion.TransactionID)
			if _, err := store.Queries.UpdateRawConversionOfferURLID(ctx, db.UpdateRawConversionOfferURLIDParams{
				ID:         conversion.ID,
				OfferURLID: offerLinkID,
			}); err != nil {
				log.Printf("error updating OfferURLID for RawConversion %d: %v\n", conversion.ID, err)
			}
		}

		// Conversion already exists, update it
		conversion, err = store.Queries.UpdateRawConversion(ctx, db.UpdateRawConversionParams{
			ID:            conversion.ID,
			NetworkID:     int32(networkID),
			TransactionID: conversionData.TransactionID,
			ClickID:       clickID,
			Status:        int32(status),
			PayoutModel:   db.Int(&payoutModel),
			PayoutAmount:  db.Numeric(&payoutAmount),
			RevenueModel:  db.Int(&revenueModel),
			RevenueAmount: db.Numeric(&revenueAmount),
			IPAddress:     conversionData.ConversionUserIP,
			UserAgent:     conversionData.HTTPUserAgent,
		})
		if err != nil {
			return result, fmt.Errorf("error updating RawConversion %s with transaction ID %s: %v", offer.Name, conversionData.TransactionID, err)
		}

		result.Conversion = &conversion
		result.Existing = true

		return result, nil
	}

	conversion, err := store.Queries.CreateRawConversion(ctx, db.CreateRawConversionParams{
		OfferID:       offerID,
		OfferURLID:    offerLinkID,
		ConversionID:  conversionData.ConversionID,
		NetworkID:     int32(networkID),
		TransactionID: conversionData.TransactionID,
		Status:        int32(status),
		PayoutModel:   db.Int(&payoutModel),
		PayoutAmount:  db.Numeric(&payoutAmount),
		RevenueModel:  db.Int(&revenueModel),
		RevenueAmount: db.Numeric(&revenueAmount),
		IPAddress:     conversionData.ConversionUserIP,
		UserAgent:     conversionData.HTTPUserAgent,
		CreatedAt:     createdAt,
	})
	if err != nil {
		return result, fmt.Errorf("error creating RawConversion %s with transaction ID %s: %v", offer.Name, conversionData.TransactionID, err)
	}

	result.Conversion = &conversion

	return result, nil
}
