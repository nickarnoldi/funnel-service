package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/server/api/handler/requests"
	"github.com/nickarnoldi/funnel-service/internal/utils"
)

func (handler *Handler) HandleStoreAbandonedLead(w http.ResponseWriter, r *http.Request) {
	store := handler.Store

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendNotFoundError(w, "Not Found")
		return
	}

	body = []byte(strings.ToValidUTF8(string(body), ""))

	var request requests.AbandonedLeadRequest
	if err := json.Unmarshal(body, &request); err != nil {
		sendNotFoundError(w, "Not Found")
		return
	}

	request.IPAddress = utils.GetRequestIPAddress(r)
	request.Referrer = r.Referer()

	result, err := request.Process(store)
	if err != nil || result.AbandonedLead == nil {
		log.Printf("Error processing abandoned lead request: %v", err)
		sendInternalError(w, "Unable to process abandoned lead request")
		return
	}

	lead := result.AbandonedLead
	if err = lead.Save(store); err != nil {
		log.Printf("Error creating abandoned lead: %v", err)
		sendInternalError(w, "Unable to create abandoned lead")
		return
	}

	if result.Step != nil {
		if _, err = handler.Queries.CreateAbandonedLeadFunnelStep(handler.Ctx, db.CreateAbandonedLeadFunnelStepParams{
			AbandonedLeadID: lead.ID,
			FunnelStepID:    result.Step.ID,
		}); err != nil {
			log.Printf("Error creating abandoned lead funnel step: %v", err)
			log.Printf("Abandoned Lead ID: %d, Funnel Step ID: %d", lead.ID, result.Step.ID)
			log.Printf("Abandoned Step data: %+v", result.Step)
			sendInternalError(w, "Unable to create abandoned lead funnel step")
			return
		}
	}

	w.Write([]byte("OK"))
}
