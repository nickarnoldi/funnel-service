package everflow

import (
	"context"
	"fmt"
	"time"

	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/env"
	"github.com/nickarnoldi/funnel-service/internal/everflow"
	"github.com/sirupsen/logrus"
)

const UTCTimeZoneID = 67 // UTC timezone ID for Everflow API

func GetOfferURLs(ctx context.Context, network enum.NetworkEnum) (*[]everflow.OfferURLData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	return client.GetOfferURLs(ctx, everflow.OfferURLsRequestParams{})
}

func GetOffers(ctx context.Context, network enum.NetworkEnum) (*[]everflow.OfferData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	return client.GetOffers(ctx, everflow.OffersRequestParams{})
}

func GetAdvertisers(ctx context.Context, network enum.NetworkEnum) (*[]everflow.AdvertiserData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	return client.GetAdvertisers(ctx, everflow.AdvertisersRequestParams{})
}

func GetCategories(ctx context.Context, network enum.NetworkEnum) (*[]everflow.CategoryData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	return client.GetCategories(ctx)
}

func GetAffiliates(ctx context.Context, network enum.NetworkEnum) (*[]everflow.AffiliateData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	affiliates, err := client.GetAffiliates(ctx, everflow.GetAffiliatesParams{})
	if err != nil {
		return nil, err
	}

	return affiliates, nil
}

func GetRawClickByID(ctx context.Context, network enum.NetworkEnum, transactionID string) (*everflow.ClickData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	return client.GetClickByID(ctx, transactionID)
}

func ImportRawClickByID(ctx context.Context, store *db.Store, network enum.NetworkEnum, transactionID string) (*db.RawClick, error) {
	click, err := GetRawClickByID(ctx, network, transactionID)
	if err != nil {
		return nil, err
	}

	result, err := ImportClick(ctx, store, click)
	if err != nil {
		return nil, err
	}

	return result.Click, nil
}

func GetRawClicks(ctx context.Context, network enum.NetworkEnum, from time.Time, to time.Time) (*[]everflow.ClickData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	clicks, err := client.GetClicksByDateTime(ctx, everflow.GetClicksByDateTimeParams{
		FromDateTime: from.Format("2006-01-02 15:04:05"),
		ToDateTime:   to.Format("2006-01-02 15:04:05"),
		TimezoneID:   UTCTimeZoneID,
	})
	if err != nil {
		return nil, err
	}

	return clicks, nil
}

func GetRawConversions(ctx context.Context, network enum.NetworkEnum, from time.Time, to time.Time) (*[]everflow.ConversionData, error) {
	client, err := newClient(network)
	if err != nil {
		return nil, err
	}

	conversions, err := client.GetConversionsByDateTime(ctx, everflow.GetConversionsByDateTimeParams{
		FromDate:   from.Format("2006-01-02 15:04:05"),
		ToDate:     to.Format("2006-01-02 15:04:05"),
		TimezoneID: UTCTimeZoneID,
	})
	if err != nil {
		return nil, err
	}

	return &conversions, nil
}

func newClient(network enum.NetworkEnum) (*everflow.Client, error) {
	var apiKey string

	switch network {
	case enum.NetworkSecondary:
		apiKey = env.Get("EVERFLOW_SECONDARY_API_KEY", "")
	default:
		apiKey = env.Get("EVERFLOW_API_KEY", "")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("missing Everflow API key for network %s", network)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	return everflow.NewClient(&everflow.ClientConfig{
		APIKey: apiKey,
		Logger: logger,
	})
}
