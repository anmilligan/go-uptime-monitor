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
type IListTargetsUseCase interface {
	ListTargets(ctx context.Context, userID int) ([]domain.Target, error)
}

type IDeleteTargetUseCase interface {
	DeleteTarget(ctx context.Context, id int) error
}
type TargetHandler struct {
	addTarget    IAddTargetUseCase
	getTarget    IGetTargetUseCase
	listTargets  IListTargetsUseCase
	deleteTarget IDeleteTargetUseCase
}

func NewTargetHandler(addTarget IAddTargetUseCase, getTarget IGetTargetUseCase, listTargets IListTargetsUseCase, deleteTarget IDeleteTargetUseCase) *TargetHandler {
	return &TargetHandler{
		addTarget:    addTarget,
		getTarget:    getTarget,
		listTargets:  listTargets,
		deleteTarget: deleteTarget,
	}
}

func (h *TargetHandler) List(w http.ResponseWriter, r *http.Request) {
	userIDParam := r.URL.Query().Get("user_id")

	userID, err := strconv.Atoi(userIDParam)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	targets, err := h.listTargets.ListTargets(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(targets); err != nil {
		return
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

func (h *TargetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid target id", http.StatusBadRequest)
		return
	}

	if err := h.deleteTarget.DeleteTarget(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
