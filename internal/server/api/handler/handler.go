package handler

import (
	"net/http"

	"github.com/nickarnoldi/funnel-service/internal/db"
)

type Handler struct {
	*db.Store
}

func sendNotFoundError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusNotFound)
}

func sendInternalError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusInternalServerError)
}

func sendRequestError(w http.ResponseWriter, msg string) {
	http.Error(w, msg, http.StatusBadRequest)
}
