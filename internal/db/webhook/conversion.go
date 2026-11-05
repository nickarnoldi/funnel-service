package webhook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
)

func ImportConversion(ctx context.Context, store *db.Store, firehoseConversion *models.Conversion) (*db.RawConversion, error) {
	if firehoseConversion.TransactionID == "" {
		return nil, fmt.Errorf("skipping Conversion with empty TransactionID")
	}

	if firehoseConversion.ConversionID == "" {
		return nil, fmt.Errorf("skipping Conversion with empty ConversionID")
	}

	if firehoseConversion.OfferID == 0 {
		return nil, fmt.Errorf("skipping Conversion with empty OfferID")
	}

	if firehoseConversion.NetworkID == 0 {
		return nil, fmt.Errorf("skipping Conversion with empty NetworkID")
	}

	networkID := enum.NetworkEnum(firehoseConversion.NetworkID)

	platformID := enum.PlatformEverflowPrimary
	if networkID == enum.NetworkSecondary {
		platformID = enum.PlatformEverflowSecondary
	}

	status, found := enum.Index(enum.ConversionStatus, firehoseConversion.Status)
	if !found {
		return nil, fmt.Errorf("error getting Status for Conversion with transaction ID %s", firehoseConversion.TransactionID)
	}

	var createdAt pgtype.Timestamp
	if err := createdAt.Scan(time.Unix(int64(firehoseConversion.ConversionTimestamp), 0).In(time.UTC)); err != nil {
		return nil, fmt.Errorf("error scanning CreatedAt %d for Conversion with transaction ID %s: %v", firehoseConversion.ConversionTimestamp, firehoseConversion.TransactionID, err)
	}

	var offerID pgtype.Int8
	var offer db.Offer
	if offerRow, err := store.Queries.GetOfferByExternalReferenceID(ctx, db.GetOfferByExternalReferenceIDParams{
		PlatformID: int32(platformID),
		ExternalID: int64(firehoseConversion.OfferID),
	}); err == nil {
		offer = offerRow.Offer
		offerID.Scan(offer.ID)
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("error getting Offer with imported ID %d: %v", firehoseConversion.OfferID, err)
	}

	var offerLinkID pgtype.Int8
	if firehoseConversion.OfferURLID == 0 {
		if defaultLinkID, err := store.Queries.GetDefaultOfferURLIDByOfferID(ctx, offer.ID); err == nil {
			offerLinkID.Scan(defaultLinkID)
		}
	} else {
		if linkID, err := store.Queries.GetOfferURLIDByExternalReferenceID(ctx, db.GetOfferURLIDByExternalReferenceIDParams{
			PlatformID: int32(platformID),
			ExternalID: int64(firehoseConversion.OfferURLID),
		}); err == nil {
			offerLinkID.Scan(linkID)
		}
	}

	payoutModel, _ := enum.Index(enum.PayoutModel, strings.ToUpper(firehoseConversion.PayoutType))
	revenueModel, _ := enum.Index(enum.RevenueModel, strings.ToUpper(firehoseConversion.RevenueType))

	var clickID pgtype.Int8
	if rawClickID, err := store.Queries.GetRawClickIDByNetworkAndTransactionID(ctx, db.GetRawClickIDByNetworkAndTransactionIDParams{
		NetworkID:     int32(networkID),
		TransactionID: firehoseConversion.TransactionID,
	}); err == nil {
		clickID.Scan(rawClickID)
	}

	if conversionID, err := store.Queries.GetRawConversionIDByNetworkAndConversionID(ctx, db.GetRawConversionIDByNetworkAndConversionIDParams{
		NetworkID:    int32(networkID),
		ConversionID: firehoseConversion.ConversionID,
	}); err == nil {
		conversion, err := store.Queries.UpdateRawConversion(ctx, db.UpdateRawConversionParams{
			ID:            conversionID,
			NetworkID:     int32(networkID),
			TransactionID: firehoseConversion.TransactionID,
			ClickID:       clickID,
			Status:        int32(status),
			PayoutModel:   db.Int(&payoutModel),
			PayoutAmount:  db.Numeric(&firehoseConversion.Payout),
			RevenueModel:  db.Int(&revenueModel),
			RevenueAmount: db.Numeric(&firehoseConversion.Revenue),
			IPAddress:     firehoseConversion.ConversionUserIP,
			UserAgent:     firehoseConversion.HTTPUserAgent,
		})
		if err != nil {
			return nil, fmt.Errorf("error updating %s RawConversion %s with transaction ID %s: %v", networkID.String(), offer.Name, firehoseConversion.TransactionID, err)
		}

		return &conversion, nil
	} else if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("error fetching %s RawConversion for transaction ID %s: %v", networkID.String(), firehoseConversion.TransactionID, err)
	}

	conversion, err := store.Queries.CreateRawConversion(ctx, db.CreateRawConversionParams{
		OfferID:       offerID,
		OfferURLID:    offerLinkID,
		ConversionID:  firehoseConversion.ConversionID,
		NetworkID:     int32(networkID),
		TransactionID: firehoseConversion.TransactionID,
		ClickID:       clickID,
		Status:        int32(status),
		PayoutModel:   db.Int(&payoutModel),
		PayoutAmount:  db.Numeric(&firehoseConversion.Payout),
		RevenueModel:  db.Int(&revenueModel),
		RevenueAmount: db.Numeric(&firehoseConversion.Revenue),
		IPAddress:     firehoseConversion.ConversionUserIP,
		UserAgent:     firehoseConversion.HTTPUserAgent,
		CreatedAt:     createdAt,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating %s RawConversion %s with transaction ID %s: %v", networkID.String(), offer.Name, firehoseConversion.TransactionID, err)
	}

	return &conversion, nil
}
