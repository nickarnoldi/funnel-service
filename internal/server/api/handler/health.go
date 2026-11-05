package handler

import "net/http"

func (handler *Handler) HandleHealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("OK"))
}
