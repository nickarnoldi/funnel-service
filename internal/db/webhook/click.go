package webhook

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
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
)

func ImportClick(ctx context.Context, store *db.Store, firehoseClick *models.Click) (*db.RawClick, error) {
	if firehoseClick.TransactionID == "" {
		return nil, fmt.Errorf("skipping Click with empty TransactionID")
	}

	networkID := enum.NetworkEnum(firehoseClick.NetworkID)
	platformID := networkID.GetPlatform()

	categoryID, err := store.Queries.GetCategoryIDByExternalReferenceID(ctx, db.GetCategoryIDByExternalReferenceIDParams{
		PlatformID: int32(platformID),
		ExternalID: int64(firehoseClick.CategoryID),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting Category %d for Click with transaction ID %s: %v", firehoseClick.CategoryID, firehoseClick.TransactionID, err)
	}

	advertiserID, err := store.Queries.GetAdvertiserIDByExternalReferenceID(ctx, db.GetAdvertiserIDByExternalReferenceIDParams{
		PlatformID: int32(platformID),
		ExternalID: int64(firehoseClick.AdvertiserID),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting Advertiser with imported ID %d: %v", firehoseClick.AdvertiserID, err)
	}

	var affiliateID pgtype.Int8
	if firehoseClick.AffiliateID != 0 {
		if fetchedAffiliateID, err := store.Queries.GetAffiliateIDByExternalReferenceID(ctx, db.GetAffiliateIDByExternalReferenceIDParams{
			PlatformID: int32(platformID),
			ExternalID: int64(firehoseClick.AffiliateID),
		}); err == nil {
			affiliateID.Scan(fetchedAffiliateID)
		} else if err != pgx.ErrNoRows {
			return nil, fmt.Errorf("error getting Affiliate with imported ID %d: %v", firehoseClick.AffiliateID, err)
		}
	}

	redirectMethod, found := enum.Index(enum.RedirectMethod, firehoseClick.RedirectMethod)
	if !found {
		return nil, fmt.Errorf("unknown redirect method %s for Click with transaction ID %s, skipping", firehoseClick.RedirectMethod, firehoseClick.TransactionID)
	}

	var createdAt pgtype.Timestamp
	if err := createdAt.Scan(time.Unix(int64(firehoseClick.UnixTimestamp), 0).In(time.UTC)); err != nil {
		return nil, fmt.Errorf("error scanning CreatedAt for Click with transaction ID %s: %v", firehoseClick.TransactionID, err)
	}

	offerID, err := store.Queries.GetOfferIDByExternalReferenceID(ctx, db.GetOfferIDByExternalReferenceIDParams{
		PlatformID: int32(platformID),
		ExternalID: int64(firehoseClick.OfferID),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting Offer with imported ID %d: %v", firehoseClick.OfferID, err)
	}

	var offerLinkID pgtype.Int8
	if firehoseClick.OfferURLID == 0 {
		// No OfferURLID; try to get the default offer URL for this offer
		if defaultOfferURLID, err := store.Queries.GetDefaultOfferURLIDByOfferID(ctx, offerID); err == nil {
			offerLinkID.Scan(defaultOfferURLID)
		}
	} else {
		if linkID, err := store.Queries.GetOfferURLIDByExternalReferenceID(ctx, db.GetOfferURLIDByExternalReferenceIDParams{
			PlatformID: int32(platformID),
			ExternalID: int64(firehoseClick.OfferURLID),
		}); err == nil {
			offerLinkID.Scan(linkID)
		}
	}

	var isBot, isMobile, isTablet bool
	var languageCode, countryCode, regionCode pgtype.Text

	var trafficSource enum.TrafficSourceEnum
	trafficSource.FromString(firehoseClick.Sub5)
	if trafficSource == enum.TrafficSourceOther {
		trafficSource.FromString(firehoseClick.Referer)
	}

	var deviceType enum.DeviceTypeEnum
	deviceType.FromString(firehoseClick.HTTPUserAgent)

	var geolocation, deviceInfo []byte
	if firehoseClick.Geolocation != nil {
		var nativeGeo db.Geolocation
		nativeGeo.FromFirehose(*firehoseClick.Geolocation)
		geolocation, _ = json.Marshal(nativeGeo)

		countryCode.Scan(nativeGeo.CountryCode)
		regionCode.Scan(nativeGeo.RegionCode)
	}
	if firehoseClick.DeviceInfo != nil {
		var nativeDeviceInfo db.DeviceInfo
		nativeDeviceInfo.FromFirehose(*firehoseClick.DeviceInfo)
		deviceInfo, _ = json.Marshal(nativeDeviceInfo)

		isBot = nativeDeviceInfo.IsRobot
		isMobile = nativeDeviceInfo.IsMobile
		isTablet = nativeDeviceInfo.IsTablet
		languageCode.Scan(nativeDeviceInfo.Language)

		if deviceType == enum.DeviceTypeOther {
			deviceType.FromString(firehoseClick.DeviceInfo.PlatformName)
			if deviceType == enum.DeviceTypeOther {
				deviceType.FromString(firehoseClick.DeviceInfo.Brand)
			}
		}
	}

	revenue := fmt.Sprintf("%.2f", firehoseClick.Revenue)
	payout := fmt.Sprintf("%.2f", firehoseClick.Payout)

	// Check if this click already exists
	if click, err := store.Queries.GetRawClickByNetworkAndTransactionID(ctx, db.GetRawClickByNetworkAndTransactionIDParams{
		NetworkID:     int32(networkID),
		TransactionID: firehoseClick.TransactionID,
	}); err == nil {
		click.Geolocation = geolocation
		click.DeviceInfo = deviceInfo

		if _, err := store.Queries.UpdateRawClickStatsAggregates(ctx, db.UpdateRawClickStatsAggregatesParams{
			ID:            click.ID,
			TrafficSource: int32(trafficSource),
			IsBot:         isBot,
			IsMobile:      isMobile,
			IsTablet:      isTablet,
			DeviceType:    int32(deviceType),
			LanguageCode:  languageCode,
			CountryCode:   countryCode,
			RegionCode:    regionCode,
		}); err != nil {
			log.Printf("Error updating RawClick stats aggregates for RawClick %d: %v\n", click.ID, err)
		}

		if click.Geolocation == nil || string(click.Geolocation) == "{}" || click.DeviceInfo == nil || string(click.DeviceInfo) == "{}" {
			if _, err := store.Queries.UpdateRawClickMetadata(ctx, db.UpdateRawClickMetadataParams{
				ID:          click.ID,
				Geolocation: geolocation,
				DeviceInfo:  deviceInfo,
			}); err != nil {
				log.Printf("Error updating Geolocation for RawClick %d: %v\n", click.ID, err)
			}
		}

		// Click already exists, skip it
		return &click, nil
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("error getting RawClick with transaction ID %s: %v", firehoseClick.TransactionID, err)
	}

	// insert this as a new click
	click, err := store.Queries.CreateRawClick(ctx, db.CreateRawClickParams{
		OfferID:        offerID,
		OfferURLID:     offerLinkID,
		AdvertiserID:   advertiserID,
		AffiliateID:    affiliateID,
		CategoryID:     categoryID,
		NetworkID:      int32(networkID),
		TransactionID:  firehoseClick.TransactionID,
		IsUnique:       firehoseClick.IsUnique,
		TrackingURL:    firehoseClick.TrackingURL,
		SourceID:       firehoseClick.SourceID,
		Sub1:           firehoseClick.Sub1,
		Sub2:           firehoseClick.Sub2,
		Sub3:           firehoseClick.Sub3,
		Sub4:           firehoseClick.Sub4,
		Sub5:           firehoseClick.Sub5,
		RevenueAmount:  db.Numeric(&revenue),
		PayoutAmount:   db.Numeric(&payout),
		Referer:        firehoseClick.Referer,
		ErrorCode:      int32(firehoseClick.ErrorCode),
		ErrorFilterID:  firehoseClick.ErrorFilterID,
		URL:            firehoseClick.RedirectURL,
		IsViewThrough:  firehoseClick.IsViewThrough,
		IsAsync:        firehoseClick.IsAsync,
		ServerSideURL:  firehoseClick.ServerSideURL,
		IsTestMode:     firehoseClick.IsTestMode,
		Idfa:           firehoseClick.IDFA,
		GoogleAdID:     firehoseClick.GoogleAdID,
		CreativeID:     fmt.Sprintf("%d", firehoseClick.CreativeID),
		CouponCode:     firehoseClick.CouponCode,
		RedirectMethod: int32(redirectMethod),
		IsSdkClick:     firehoseClick.IsSDKClick,
		IPAddress:      firehoseClick.UserIP,
		UserAgent:      firehoseClick.HTTPUserAgent,
		Geolocation:    geolocation,
		DeviceInfo:     deviceInfo,
		TrafficSource:  int32(trafficSource),
		IsBot:          isBot,
		IsMobile:       isMobile,
		IsTablet:       isTablet,
		DeviceType:     int32(deviceType),
		LanguageCode:   languageCode,
		CountryCode:    countryCode,
		RegionCode:     regionCode,
		CreatedAt:      createdAt,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating RawClick with transaction ID %s: %v", firehoseClick.TransactionID, err)
	}

	return &click, nil
}
