package rest

import (
	"context"
	"encoding/json"
	"net/http"
)

type IDatabasePinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db IDatabasePinger
}

func NewHealthHandler(
	db IDatabasePinger,
) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h *HealthHandler) Health(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	response := healthResponse{
		Status: "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (h *HealthHandler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := h.db.Ping(r.Context()); err != nil {
		w.WriteHeader(
			http.StatusServiceUnavailable,
		)

		response := healthResponse{
			Status: "not_ready",
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			return
		}

		return
	}

	response := healthResponse{
		Status: "ready",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
