package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/nickarnoldi/funnel-service/internal/db/webhook"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/env"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow/models"
	"github.com/nickarnoldi/funnel-service/internal/utils"
)

func (handler *Handler) isFromEverflow(r *http.Request) bool {
	// Production checks a shared key and a source IP. Both come from the environment.
	if env.Get("ENVIRONMENT", "local") != "production" {
		return true
	}

	key := env.Get("EVERFLOW_WEBHOOK_KEY", "")
	allowedIP := env.Get("EVERFLOW_WEBHOOK_IP", "")
	if key == "" || allowedIP == "" {
		log.Println("Rejecting webhook: EVERFLOW_WEBHOOK_KEY and EVERFLOW_WEBHOOK_IP must be set in production")
		return false
	}

	ipAddress := utils.GetRequestIPAddress(r)
	if ipAddress != allowedIP || r.URL.Query().Get("key") != key {
		if r.URL.Query().Get("key") == key {
			log.Printf("Received webhook request from unknown source: %s\n", ipAddress)
		}
		return false
	}
	return true
}

func (handler *Handler) HandleClickWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading click request body: %v\n", err)
		sendRequestError(w, "Bad Request")
		return
	}

	var firehoseClick models.Click
	if err := json.Unmarshal(body, &firehoseClick); err != nil {
		log.Printf("Error unmarshalling Click: %v\n", err)
		sendRequestError(w, "Bad Request")
		return
	}

	if firehoseClick.Geolocation != nil {
		firehoseClick.Geolocation.Normalize()
	}

	network := enum.NetworkEnum(firehoseClick.NetworkID)
	platform := network.GetPlatform()
	if platform != enum.PlatformEverflowPrimary && platform != enum.PlatformEverflowSecondary {
		log.Printf("Received Click for unknown platform: %d, transactionID: %s\n", firehoseClick.NetworkID, firehoseClick.TransactionID)
		log.Printf("Click data: %s\n", string(body))
		sendRequestError(w, "Bad Request")
		return
	}

	if _, err := webhook.ImportClick(handler.Ctx, handler.Store, &firehoseClick); err != nil {
		log.Printf("Error importing Click %s: %v\n", firehoseClick.TransactionID, err)
		log.Printf("Click data: %s\n", string(body))
		sendRequestError(w, "Internal Server Error")
		return
	}

	w.Write([]byte("OK"))
}

func (handler *Handler) HandleConversionWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	var firehoseConversion models.Conversion
	if err := json.Unmarshal(body, &firehoseConversion); err != nil {
		log.Printf("Error unmarshalling Conversion: %v\n", err)
		sendRequestError(w, "Bad Request")
		return
	}

	network := enum.NetworkEnum(firehoseConversion.NetworkID)
	platform := network.GetPlatform()
	if platform != enum.PlatformEverflowPrimary && platform != enum.PlatformEverflowSecondary {
		log.Printf("Received Conversion for unknown platform: %d, skipping: %s\n", firehoseConversion.NetworkID, firehoseConversion.TransactionID)
		sendRequestError(w, "Bad Request")
		return
	}

	if _, err := webhook.ImportConversion(handler.Ctx, handler.Store, &firehoseConversion); err != nil {
		log.Printf("Error importing Conversion %s: %v\n", firehoseConversion.ConversionID, err)
		log.Printf("Conversion data: %s\n", string(body))
		sendRequestError(w, "Bad Request")
		return
	}

	w.Write([]byte("OK"))
}

func (handler *Handler) HandleOfferWebhook(w http.ResponseWriter, r *http.Request) {
	if !handler.isFromEverflow(r) {
		sendNotFoundError(w, "Not Found")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	var firehoseOffer models.Offer
	if err := json.Unmarshal(body, &firehoseOffer); err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	network := enum.NetworkEnum(firehoseOffer.NetworkID)
	platform := network.GetPlatform()
	if platform != enum.PlatformEverflowPrimary && platform != enum.PlatformEverflowSecondary {
		log.Printf("Received Offer for unknown platform: %d, skipping: %d\n", firehoseOffer.NetworkID, firehoseOffer.ID)
		sendRequestError(w, "Bad Request")
		return
	}

	if _, err := webhook.ImportOffer(handler.Ctx, handler.Store, &firehoseOffer); err != nil {
		log.Printf("Error importing Offer %s: %v\n", firehoseOffer.Name, err)
		log.Printf("Offer data: %s\n", string(body))
		sendRequestError(w, "Bad Request")
		return
	}
	log.Printf("Webhook - Imported Offer %s from %s\n", firehoseOffer.Name, platform.String())

	w.Write([]byte("OK"))
}

func (handler *Handler) HandleAffiliateWebhook(w http.ResponseWriter, r *http.Request) {
	if !handler.isFromEverflow(r) {
		sendNotFoundError(w, "Not Found")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	var affiliate models.Affiliate
	if err := json.Unmarshal(body, &affiliate); err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	platform := enum.NetworkEnum(affiliate.NetworkID).GetPlatform()
	if platform != enum.PlatformEverflowPrimary && platform != enum.PlatformEverflowSecondary {
		log.Printf("Received Affiliate for unknown platform: %d, skipping: %d\n", affiliate.NetworkID, affiliate.ID)
		sendRequestError(w, "Bad Request")
		return
	}

	if err := webhook.ImportAffiliate(handler.Ctx, handler.Store, &affiliate); err != nil {
		log.Printf("Error importing Affiliate %s: %v\n", affiliate.Name, err)
		log.Printf("Affiliate data: %s\n", string(body))
		sendRequestError(w, "Bad Request")
		return
	}

	log.Printf("Webhook - Imported Affiliate %s from %s\n", affiliate.Name, platform.String())

	w.Write([]byte("OK"))
}

func (handler *Handler) HandleAdvertiserWebhook(w http.ResponseWriter, r *http.Request) {
	if !handler.isFromEverflow(r) {
		sendNotFoundError(w, "Not Found")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	var advertiser models.Advertiser
	if err := json.Unmarshal(body, &advertiser); err != nil {
		sendRequestError(w, "Bad Request")
		return
	}

	platform := enum.NetworkEnum(advertiser.NetworkID).GetPlatform()
	if platform != enum.PlatformEverflowPrimary && platform != enum.PlatformEverflowSecondary {
		log.Printf("Received Advertiser for unknown platform: %d, skipping: %d\n", advertiser.NetworkID, advertiser.ID)
		sendRequestError(w, "Bad Request")
		return
	}

	if err := webhook.ImportAdvertiser(handler.Ctx, handler.Store, &advertiser); err != nil {
		log.Printf("Error importing Advertiser %s: %v\n", advertiser.Name, err)
		log.Printf("Advertiser data: %s\n", string(body))
		sendRequestError(w, "Bad Request")
		return
	}

	log.Printf("Webhook - Imported Advertiser %s from %s\n", advertiser.Name, platform.String())

	w.Write([]byte("OK"))
}
