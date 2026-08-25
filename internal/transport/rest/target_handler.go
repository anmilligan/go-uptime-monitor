package rest

import (
	"Uptime_Monitor/internal/domain"
	"Uptime_Monitor/internal/usecase"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type IAddTargetUseCase interface {
	AddTarget(ctx context.Context, req usecase.AddTargetRequest) (*domain.Target, error)
}

type IGetTargetUseCase interface {
	GetTarget(ctx context.Context, id int) (*domain.Target, error)
}

type TargetHandler struct {
	addTarget IAddTargetUseCase
	getTarget IGetTargetUseCase
}

func NewTargetHandler(addTarget IAddTargetUseCase, getTarget IGetTargetUseCase) *TargetHandler {
	return &TargetHandler{
		addTarget: addTarget,
		getTarget: getTarget,
	}
}

func (h *TargetHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid target id", http.StatusBadRequest)
		return
	}

	target, err := h.getTarget.GetTarget(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(target); err != nil {
		return
	}
}

type createTargetRequest struct {
	UserID          int    `json:"user_id"`
	Kind            string `json:"kind"`
	Address         string `json:"address"`
	IntervalSeconds int64  `json:"interval_seconds"`
	TimeoutSeconds  int64  `json:"timeout_seconds"`
	ExpectedStatus  int    `json:"expected_status"`
	ExpectedBody    string `json:"expected_body"`
}

func (h *TargetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTargetRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	addTargetReq := usecase.AddTargetRequest{
		UserID:         req.UserID,
		Kind:           req.Kind,
		Address:        req.Address,
		Interval:       time.Duration(req.IntervalSeconds) * time.Second,
		Timeout:        time.Duration(req.TimeoutSeconds) * time.Second,
		ExpectedStatus: req.ExpectedStatus,
		ExpectedBody:   req.ExpectedBody,
	}

	target, err := h.addTarget.AddTarget(r.Context(), addTargetReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(target); err != nil {
		return
	}
}
