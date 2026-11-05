package everflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/constants"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
)

func SyncRawClicks(ctx context.Context, store *db.Store, requestedStartDate time.Time, requestedEndDate time.Time, requestedNetworkID ...enum.NetworkEnum) error {
	availableNetworks := enum.Network
	if len(requestedNetworkID) > 0 {
		availableNetworks = make(map[enum.NetworkEnum]string)
		for _, networkID := range requestedNetworkID {
			if networkName, exists := enum.Network[networkID]; exists {
				availableNetworks[networkID] = networkName
			}
		}
	}

	for networkID, networkName := range availableNetworks {
		startDate := requestedStartDate
		endDate := requestedEndDate

		if startDate.IsZero() {
			lastClickUpdate, err := store.Queries.GetLastRawClickTimestampByNetwork(ctx, int32(networkID))
			if err != nil && err != pgx.ErrNoRows {
				return fmt.Errorf("failed to get last click timestamp: %w", err)
			}

			if !lastClickUpdate.Valid {
				startDate = constants.FunnelLoggingStartDate
			} else {
				fmt.Printf("Last %s Click Timestamp: %s\n", networkName, lastClickUpdate.Time)
				startDate = lastClickUpdate.Time
			}
		}

		syncRangeStartDate, syncRangeEndDate, err := store.GenerateSafeDateRange(startDate, endDate)
		if err != nil {
			return fmt.Errorf("failed to get date range: %w", err)
		}

		var totalNetworkClicks int
		var totalNetworkConversions int
		fmt.Printf("Syncing hourly %s Clicks from %s to %s\n", networkName, syncRangeStartDate.Format("2006-01-02 15:04:05"), syncRangeEndDate.Format("2006-01-02 15:04:05"))

		syncDate := syncRangeStartDate

		for syncDate.Before(syncRangeEndDate) {
			rangeFrom := syncDate
			rangeTo := time.Date(syncDate.Year(), syncDate.Month(), syncDate.Day(), syncDate.Hour(), 59, 59, 0, time.UTC)

			if rangeTo.After(syncRangeEndDate) {
				rangeTo = syncRangeEndDate
			}

			clicks, err := GetRawClicks(ctx, networkID, rangeFrom, rangeTo)
			if err != nil {
				return fmt.Errorf("failed to get raw clicks: %w", err)
			}

			count := len(*clicks)
			totalNetworkClicks += count

			fmt.Printf("Downloaded %d %s Clicks from %s to %s.\n", count, networkName, rangeFrom.Format("2006-01-02 15:04:05"), rangeTo.Format("2006-01-02 15:04:05"))

			conversionsLinked, err := importRawClicks(ctx, store, clicks)
			if err != nil {
				return fmt.Errorf("failed to import raw clicks: %w", err)
			}

			totalNetworkConversions += conversionsLinked

			fmt.Printf("Synced %d %s Clicks (%d Conversions linked) from %s to %s.\n", count, networkName, conversionsLinked, rangeFrom.Format("2006-01-02 15:04:05"), rangeTo.Format("2006-01-02 15:04:05"))

			syncDate = time.Date(syncDate.Year(), syncDate.Month(), syncDate.Day(), syncDate.Hour()+1, 0, 0, 0, time.UTC)
		}

		fmt.Printf("%s - Total Clicks: %d, Conversions linked: %d\n", networkName, totalNetworkClicks, totalNetworkConversions)
	}

	return nil
}

func importRawClicks(ctx context.Context, store *db.Store, clicks *[]everflow.ClickData) (int, error) {
	var failures int
	var conversionsLinked int

	for _, efClick := range *clicks {
		select {
		case <-ctx.Done():
			return conversionsLinked, ctx.Err()
		default:
		}

		result, err := ImportClick(ctx, store, &efClick)
		if err != nil {
			failures++
			fmt.Printf("Error importing Click with transaction ID %s: %v\n", efClick.TransactionID, err)
		}

		conversionsLinked += result.ConversionsLinked
	}

	if failures > 0 {
		return conversionsLinked, fmt.Errorf("imported clicks with %d failures out of %d total", failures, len(*clicks))
	}

	return conversionsLinked, nil
}

type ImportEverflowClickResult struct {
	Click             *db.RawClick
	ConversionsLinked int
}

func ImportClick(ctx context.Context, store *db.Store, efClick *everflow.ClickData) (ImportEverflowClickResult, error) {
	var result ImportEverflowClickResult

	if efClick.TransactionID == "" {
		return result, fmt.Errorf("skipping Click with empty TransactionID")
	}

	networkID := enum.NetworkEnum(efClick.Relationship.Offer.NetworkID)
	platform := networkID.GetPlatform()
	if platform == enum.PlatformUnknown {
		return result, fmt.Errorf("unknown platform for Click with transaction ID %s and NetworkID %d, skipping", efClick.TransactionID, efClick.Relationship.Offer.NetworkID)
	}

	categoryID, err := store.Queries.GetCategoryIDByExternalReferenceID(ctx, db.GetCategoryIDByExternalReferenceIDParams{
		PlatformID: int32(platform),
		ExternalID: int64(efClick.CategoryID),
	})
	if err != nil {
		return result, fmt.Errorf("error getting Category %d for Click with transaction ID %s: %v", efClick.CategoryID, efClick.TransactionID, err)
	}

	advertiserID, err := store.Queries.GetAdvertiserIDByExternalReferenceID(ctx, db.GetAdvertiserIDByExternalReferenceIDParams{
		PlatformID: int32(platform),
		ExternalID: int64(efClick.Relationship.Advertiser.NetworkAdvertiserID),
	})
	if err != nil {
		return result, fmt.Errorf("error getting Advertiser %s with imported ID %d: %v", efClick.Relationship.Advertiser.Name, efClick.Relationship.Advertiser.NetworkAdvertiserID, err)
	}

	var affiliateID pgtype.Int8
	if efClick.Relationship.Affiliate.NetworkAffiliateID != 0 {
		if fetchedAffiliateID, err := store.Queries.GetAffiliateIDByExternalReferenceID(ctx, db.GetAffiliateIDByExternalReferenceIDParams{
			PlatformID: int32(platform),
			ExternalID: int64(efClick.Relationship.Affiliate.NetworkAffiliateID),
		}); err == nil {
			affiliateID.Scan(fetchedAffiliateID)
		} else if err != pgx.ErrNoRows {
			return result, fmt.Errorf("error getting Affiliate %s with imported ID %d: %v", efClick.Relationship.Affiliate.Name, efClick.Relationship.Affiliate.NetworkAffiliateID, err)
		}
	}

	payoutAmount := fmt.Sprintf("%.2f", efClick.Payout)
	revenueAmount := fmt.Sprintf("%.2f", efClick.Revenue)

	redirectMethod, found := enum.Index(enum.RedirectMethod, efClick.RedirectMethod)
	if !found {
		redirectMethod = int(enum.RedirectMethodStandard)
	}

	var createdAt pgtype.Timestamp
	if err := createdAt.Scan(time.Unix(efClick.UnixTimestamp, 0).In(time.UTC)); err != nil {
		return result, fmt.Errorf("error scanning CreatedAt for Click with transaction ID %s: %v", efClick.TransactionID, err)
	}

	offerRow, err := store.Queries.GetOfferByExternalReferenceID(ctx, db.GetOfferByExternalReferenceIDParams{
		PlatformID: int32(platform),
		ExternalID: int64(efClick.Relationship.Offer.NetworkOfferID),
	})
	if err != nil {
		return result, fmt.Errorf("error getting Offer %s with imported ID %d: %v", efClick.Relationship.Offer.Name, efClick.Relationship.Offer.NetworkOfferID, err)
	}
	offer := offerRow.Offer

	// OfferURL is not present in the clicks stream model, so we try to look it up by URL
	var offerLinkID pgtype.Int8
	if efClick.URL == "" {
		if defaultLinkID, err := store.Queries.GetDefaultOfferURLIDByOfferID(ctx, offer.ID); err == nil {
			offerLinkID.Scan(defaultLinkID)
		}
	} else {
		if urlID, err := store.Queries.GetOfferURLIDByURL(ctx, efClick.URL); err == nil {
			offerLinkID.Scan(urlID)
		}
	}

	var geo db.Geolocation
	geo.CountryCode = efClick.Relationship.Geolocation.CountryCode
	geo.CountryName = efClick.Relationship.Geolocation.CountryName
	geo.RegionCode = efClick.Relationship.Geolocation.RegionCode
	geo.RegionName = efClick.Relationship.Geolocation.RegionName
	geo.CityName = efClick.Relationship.Geolocation.CityName
	geo.ISPName = efClick.Relationship.Geolocation.IspName
	geo.DMA = efClick.Relationship.Geolocation.Dma
	geo.PostalCode = efClick.Relationship.Geolocation.PostalCode
	geo.Timezone = efClick.Relationship.Geolocation.Timezone
	geo.CarrierName = efClick.Relationship.Geolocation.CarrierName
	geo.CarrierCode = efClick.Relationship.Geolocation.CarrierCode
	geo.IsMobile = efClick.Relationship.Geolocation.IsMobile
	geo.IsProxy = efClick.Relationship.Geolocation.IsProxy
	geo.Normalize()

	geolocation, err := json.Marshal(geo)
	if err != nil {
		return result, fmt.Errorf("error marshalling Geolocation for Click with transaction ID %s: %v", efClick.TransactionID, err)
	}
	deviceInfo, err := json.Marshal(efClick.Relationship.DeviceInformation)
	if err != nil {
		return result, fmt.Errorf("error marshalling DeviceInfo for Click with transaction ID %s: %v", efClick.TransactionID, err)
	}

	var languageCode, countryCode, regionCode pgtype.Text
	languageCode.Scan(efClick.Relationship.DeviceInformation.Language)
	countryCode.Scan(geo.CountryCode)
	regionCode.Scan(geo.RegionCode)

	var trafficSource enum.TrafficSourceEnum
	trafficSource.FromString(efClick.Sub5)
	if trafficSource == enum.TrafficSourceOther {
		trafficSource.FromString(efClick.Referer)
	}

	var deviceType enum.DeviceTypeEnum
	deviceType.FromString(efClick.Relationship.HTTPUserAgent)
	if deviceType == enum.DeviceTypeOther {
		deviceType.FromString(efClick.Relationship.DeviceInformation.PlatformName)
		if deviceType == enum.DeviceTypeOther {
			deviceType.FromString(efClick.Relationship.DeviceInformation.Brand)
		}
	}

	// Check if this click already exists
	if click, err := store.Queries.GetRawClickByNetworkAndTransactionID(ctx, db.GetRawClickByNetworkAndTransactionIDParams{
		NetworkID:     int32(networkID),
		TransactionID: efClick.TransactionID,
	}); err == nil {
		result.Click = &click

		if offerLinkID.Valid && !click.OfferURLID.Valid {
			if _, err := store.Queries.UpdateRawClickOfferURLID(ctx, db.UpdateRawClickOfferURLIDParams{
				ID:         click.ID,
				OfferURLID: offerLinkID,
			}); err != nil {
				log.Printf("error updating OfferURLID for RawClick %d: %v\n", click.ID, err)
			}
		}

		// Update the new aggregate fields
		if _, err := store.Queries.UpdateRawClickStatsAggregates(ctx, db.UpdateRawClickStatsAggregatesParams{
			ID:            click.ID,
			TrafficSource: int32(trafficSource),
			IsBot:         efClick.Relationship.DeviceInformation.IsRobot,
			IsMobile:      efClick.Relationship.DeviceInformation.IsMobile,
			IsTablet:      efClick.Relationship.DeviceInformation.IsTablet,
			DeviceType:    int32(deviceType),
			LanguageCode:  languageCode,
			CountryCode:   countryCode,
			RegionCode:    regionCode,
		}); err != nil {
			log.Printf("Error updating RawClick stats aggregates for RawClick %d: %v\n", click.ID, err)
		}

		if (click.Geolocation == nil || string(click.Geolocation) == "{}") || (click.DeviceInfo == nil || string(click.DeviceInfo) == "{}") {
			if _, err := store.Queries.UpdateRawClickMetadata(ctx, db.UpdateRawClickMetadataParams{
				ID:          click.ID,
				Geolocation: geolocation,
				DeviceInfo:  deviceInfo,
			}); err != nil {
				log.Printf("error updating Geolocation for RawClick %d: %v\n", click.ID, err)
			}
		}

		if conversions, err := store.Queries.GetRawConversionsMissingClickIDByNetworkAndTransactionID(ctx, db.GetRawConversionsMissingClickIDByNetworkAndTransactionIDParams{
			NetworkID:     int32(networkID),
			TransactionID: efClick.TransactionID,
		}); err == nil {
			for _, conversion := range conversions {
				if offerLinkID.Valid && !conversion.OfferURLID.Valid {
					if _, err := store.Queries.UpdateRawConversionOfferURLID(ctx, db.UpdateRawConversionOfferURLIDParams{
						ID:         conversion.ID,
						OfferURLID: offerLinkID,
					}); err != nil {
						log.Printf("error updating OfferURLID for RawConversion %d: %v\n", conversion.ID, err)
					}
				}

				if conversion.ClickID.Valid && conversion.ClickID.Int64 == click.ID {
					return result, nil
				}

				if _, err := store.Queries.UpdateRawConversionClickID(ctx, db.UpdateRawConversionClickIDParams{
					ID:      conversion.ID,
					ClickID: db.BigInt(&click.ID),
				}); err != nil {
					return result, fmt.Errorf("error updating Click ID for RawConversion %d: %v", conversion.ID, err)
				}

				result.ConversionsLinked++
			}
		}

		return result, nil
	} else if err != pgx.ErrNoRows {
		return result, fmt.Errorf("error getting RawClick for Click with transaction ID %s: %v", efClick.TransactionID, err)
	}

	// insert this as a new click
	click, err := store.Queries.CreateRawClick(ctx, db.CreateRawClickParams{
		OfferID:        offer.ID,
		OfferURLID:     offerLinkID,
		AdvertiserID:   advertiserID,
		AffiliateID:    affiliateID,
		CategoryID:     categoryID,
		NetworkID:      int32(networkID),
		TransactionID:  efClick.TransactionID,
		IsUnique:       efClick.IsUnique != 0,
		TrackingURL:    efClick.TrackingURL,
		SourceID:       efClick.SourceID,
		Sub1:           efClick.Sub1,
		Sub2:           efClick.Sub2,
		Sub3:           efClick.Sub3,
		Sub4:           efClick.Sub4,
		Sub5:           efClick.Sub5,
		PayoutAmount:   db.Numeric(&payoutAmount),
		RevenueAmount:  db.Numeric(&revenueAmount),
		Referer:        efClick.Referer,
		ErrorCode:      int32(efClick.ErrorCode),
		ErrorFilterID:  efClick.ErrorFilterID,
		URL:            efClick.URL,
		IsViewThrough:  efClick.IsViewThrough,
		IsAsync:        efClick.IsAsync,
		ServerSideURL:  efClick.ServerSideURL,
		IsTestMode:     efClick.IsTestMode,
		Idfa:           efClick.Idfa,
		GoogleAdID:     efClick.GoogleAdID,
		CreativeID:     fmt.Sprintf("%d", efClick.CreativeID),
		CouponCode:     efClick.CouponCode,
		RedirectMethod: int32(redirectMethod),
		IsSdkClick:     efClick.IsSdkClick,
		IPAddress:      efClick.UserIP,
		UserAgent:      efClick.Relationship.HTTPUserAgent,
		Geolocation:    geolocation,
		DeviceInfo:     deviceInfo,
		TrafficSource:  int32(trafficSource),
		IsBot:          efClick.Relationship.DeviceInformation.IsRobot,
		IsMobile:       efClick.Relationship.DeviceInformation.IsMobile,
		IsTablet:       efClick.Relationship.DeviceInformation.IsTablet,
		DeviceType:     int32(deviceType),
		LanguageCode:   languageCode,
		CountryCode:    countryCode,
		RegionCode:     regionCode,
		CreatedAt:      createdAt,
	})
	if err != nil {
		return result, fmt.Errorf("error creating RawClick %s with transaction ID %s: %v", offer.Name, efClick.TransactionID, err)
	}

	result.Click = &click

	if conversions, err := store.Queries.GetRawConversionsByNetworkAndTransactionID(ctx, db.GetRawConversionsByNetworkAndTransactionIDParams{
		NetworkID:     int32(networkID),
		TransactionID: efClick.TransactionID,
	}); err == nil {
		for _, conversion := range conversions {
			if conversion.ClickID.Valid && conversion.ClickID.Int64 == click.ID {
				continue
			}

			if _, err := store.Queries.UpdateRawConversionClickID(ctx, db.UpdateRawConversionClickIDParams{
				ID:      conversion.ID,
				ClickID: db.BigInt(&click.ID),
			}); err != nil {
				fmt.Printf("Error updating Click ID for RawConversion %d: %v\n", conversion.ID, err)
				continue
			}

			result.ConversionsLinked++
		}
	}

	return result, nil
}
