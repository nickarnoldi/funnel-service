package api

import (
	"net/http"

	"github.com/nickarnoldi/funnel-service/internal/server/api/handler"
	"github.com/nickarnoldi/funnel-service/internal/server/middleware"
)

func AddRoutes(router *http.ServeMux, handler *handler.Handler) {
	router.HandleFunc("/health", handler.HandleHealthCheck)
	router.HandleFunc("POST /abandoned", handler.HandleStoreAbandonedLead)
	router.HandleFunc("POST /webhooks/clicks", middleware.EnsureIsEverflow(handler.HandleClickWebhook))
	router.HandleFunc("POST /webhooks/conversions", middleware.EnsureIsEverflow(handler.HandleConversionWebhook))
	router.HandleFunc("POST /webhooks/offers", handler.HandleOfferWebhook)
	router.HandleFunc("POST /webhooks/affiliates", handler.HandleAffiliateWebhook)
	router.HandleFunc("POST /webhooks/advertisers", handler.HandleAdvertiserWebhook)
	router.HandleFunc("GET /js/funnel-flight.js", handler.HandleGetFunnelFlightJS)
	router.HandleFunc("GET /js/funnel-flight.min.js", handler.HandleGetFunnelFlightJS)
	// router.HandleFunc("GET /js/funnel-flight.js", middleware.EnsureSession(handler.handleGetFunnelFlightJS, handler.Store))
}
