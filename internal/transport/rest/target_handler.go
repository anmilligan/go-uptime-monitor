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

type IUpdateTargetUseCase interface {
	UpdateTarget(ctx context.Context, target *domain.Target) error
}

type IGetCheckHistoryUseCase interface {
	GetCheckHistory(ctx context.Context, targetID int, limit int) ([]domain.Check, error)
}

type IListIncidentsUseCase interface {
	ListIncidents(ctx context.Context, targetID int, limit int) ([]domain.Incident, error)
}

type IGetLastCheckUseCase interface {
	GetLastCheck(ctx context.Context, targetID int) (*domain.Check, error)
}

type IGetOpenIncidentUseCase interface {
	GetOpenIncident(ctx context.Context, targetID int) (*domain.Incident, error)
}

type ICalculateUptimeUseCase interface {
	CalculateUptime(ctx context.Context, targetID int, targetCreatedAt time.Time, period time.Duration) (*usecase.UptimeResult, error)
}

type TargetHandler struct {
	addTarget     IAddTargetUseCase
	getTarget     IGetTargetUseCase
	listTargets   IListTargetsUseCase
	deleteTarget  IDeleteTargetUseCase
	updateTarget  IUpdateTargetUseCase
	lastCheck     IGetLastCheckUseCase
	openIncident  IGetOpenIncidentUseCase
	uptime        ICalculateUptimeUseCase
	checkHistory  IGetCheckHistoryUseCase
	listIncidents IListIncidentsUseCase
}

func NewTargetHandler(
	addTarget IAddTargetUseCase,
	getTarget IGetTargetUseCase,
	listTargets IListTargetsUseCase,
	deleteTarget IDeleteTargetUseCase,
	updateTarget IUpdateTargetUseCase,
	lastCheck IGetLastCheckUseCase,
	openIncident IGetOpenIncidentUseCase,
	uptime ICalculateUptimeUseCase,
	checkHistory IGetCheckHistoryUseCase,
	listIncidents IListIncidentsUseCase,
) *TargetHandler {
	return &TargetHandler{
		addTarget:     addTarget,
		getTarget:     getTarget,
		listTargets:   listTargets,
		deleteTarget:  deleteTarget,
		lastCheck:     lastCheck,
		openIncident:  openIncident,
		uptime:        uptime,
		updateTarget:  updateTarget,
		checkHistory:  checkHistory,
		listIncidents: listIncidents,
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
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid target id",
			http.StatusBadRequest,
		)
		return
	}

	target, err := h.getTarget.GetTarget(
		r.Context(),
		id,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	check, err := h.lastCheck.GetLastCheck(
		r.Context(),
		target.ID,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	incident, err := h.openIncident.GetOpenIncident(
		r.Context(),
		target.ID,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	uptime24h, err := h.uptime.CalculateUptime(
		r.Context(),
		target.ID,
		target.CreatedAt,
		24*time.Hour,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	uptime7d, err := h.uptime.CalculateUptime(
		r.Context(),
		target.ID,
		target.CreatedAt,
		7*24*time.Hour,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	status := "unknown"

	switch {
	case !target.Active:
		status = "paused"

	case incident != nil:
		status = "down"

	case check == nil:
		status = "unknown"

	case check.IsSuccess:
		status = "up"

	default:
		status = "degraded"
	}

	var lastCheck *checkResponse

	if check != nil {
		lastCheck = &checkResponse{
			ID:             check.ID,
			TargetID:       check.TargetID,
			Status:         check.Status,
			ResponseTimeMS: check.ResponseTime.Milliseconds(),
			IsSuccess:      check.IsSuccess,
			CheckedAt:      check.CheckedAt,
		}
	}

	var openIncident *incidentResponse

	if incident != nil {
		var closedAt *time.Time

		if !incident.ClosedAt.IsZero() {
			value := incident.ClosedAt
			closedAt = &value
		}

		openIncident = &incidentResponse{
			ID:        incident.ID,
			TargetID:  incident.TargetID,
			Status:    incident.Status,
			CreatedAt: incident.CreatedAt,
			ClosedAt:  closedAt,
		}
	}

	response := targetDetailsResponse{
		ID:              target.ID,
		UserID:          target.UserID,
		Kind:            target.Kind,
		Address:         target.Address,
		IntervalSeconds: int64(target.Interval / time.Second),
		TimeoutSeconds:  int64(target.Timeout / time.Second),
		ExpectedStatus:  target.ExpectedStatus,
		ExpectedBody:    target.ExpectedBody,
		Active:          target.Active,
		CreatedAt:       target.CreatedAt,

		Status:       status,
		LastCheck:    lastCheck,
		OpenIncident: openIncident,

		Uptime: uptimeResponse{
			Last24Hours: uptimePeriodResponse{
				HasData:            uptime24h.HasData,
				Percentage:         uptime24h.Percentage,
				ObservedForSeconds: int64(uptime24h.ObservedFor / time.Second),
			},
			Last7Days: uptimePeriodResponse{
				HasData:            uptime7d.HasData,
				Percentage:         uptime7d.Percentage,
				ObservedForSeconds: int64(uptime7d.ObservedFor / time.Second),
			},
		},
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
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

type updateTargetRequest struct {
	Kind            *string `json:"kind"`
	Address         *string `json:"address"`
	IntervalSeconds *int64  `json:"interval_seconds"`
	TimeoutSeconds  *int64  `json:"timeout_seconds"`
	ExpectedStatus  *int    `json:"expected_status"`
	ExpectedBody    *string `json:"expected_body"`
}

func applyTargetUpdate(target *domain.Target, req updateTargetRequest) {
	if req.Kind != nil {
		target.Kind = *req.Kind
	}

	if req.Address != nil {
		target.Address = *req.Address
	}

	if req.IntervalSeconds != nil {
		target.Interval = time.Duration(*req.IntervalSeconds) * time.Second
	}

	if req.TimeoutSeconds != nil {
		target.Timeout = time.Duration(*req.TimeoutSeconds) * time.Second
	}

	if req.ExpectedStatus != nil {
		target.ExpectedStatus = *req.ExpectedStatus
	}

	if req.ExpectedBody != nil {
		target.ExpectedBody = *req.ExpectedBody
	}
}
func (h *TargetHandler) Update(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid target id", http.StatusBadRequest)
		return
	}

	var req updateTargetRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	target, err := h.getTarget.GetTarget(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	applyTargetUpdate(target, req)

	if err := h.updateTarget.UpdateTarget(r.Context(), target); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(target); err != nil {
		return
	}
}

type checkResponse struct {
	ID             int       `json:"id"`
	TargetID       int       `json:"target_id"`
	Status         int       `json:"status"`
	ResponseTimeMS int64     `json:"response_time_ms"`
	IsSuccess      bool      `json:"is_success"`
	CheckedAt      time.Time `json:"checked_at"`
}

type incidentResponse struct {
	ID        int        `json:"id"`
	TargetID  int        `json:"target_id"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

type uptimePeriodResponse struct {
	HasData            bool    `json:"has_data"`
	Percentage         float64 `json:"percentage"`
	ObservedForSeconds int64   `json:"observed_for_seconds"`
}

type uptimeResponse struct {
	Last24Hours uptimePeriodResponse `json:"last_24_hours"`
	Last7Days   uptimePeriodResponse `json:"last_7_days"`
}

type targetDetailsResponse struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	Kind            string    `json:"kind"`
	Address         string    `json:"address"`
	IntervalSeconds int64     `json:"interval_seconds"`
	TimeoutSeconds  int64     `json:"timeout_seconds"`
	ExpectedStatus  int       `json:"expected_status"`
	ExpectedBody    string    `json:"expected_body"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`

	Status       string            `json:"status"`
	LastCheck    *checkResponse    `json:"last_check"`
	Uptime       uptimeResponse    `json:"uptime"`
	OpenIncident *incidentResponse `json:"open_incident"`
}

func (h *TargetHandler) GetChecks(
	w http.ResponseWriter,
	r *http.Request,
) {
	idParam := chi.URLParam(r, "id")

	targetID, err := strconv.Atoi(idParam)
	if err != nil || targetID <= 0 {
		http.Error(
			w,
			"invalid target id",
			http.StatusBadRequest,
		)
		return
	}

	limit := 50

	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit <= 0 || limit > 200 {
			http.Error(
				w,
				"limit must be between 1 and 200",
				http.StatusBadRequest,
			)
			return
		}
	}

	checks, err := h.checkHistory.GetCheckHistory(
		r.Context(),
		targetID,
		limit,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	response := make(
		[]checkResponse,
		0,
		len(checks),
	)

	for _, check := range checks {
		response = append(
			response,
			checkResponse{
				ID:             check.ID,
				TargetID:       check.TargetID,
				Status:         check.Status,
				ResponseTimeMS: check.ResponseTime.Milliseconds(),
				IsSuccess:      check.IsSuccess,
				CheckedAt:      check.CheckedAt,
			},
		)
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (h *TargetHandler) GetIncidents(
	w http.ResponseWriter,
	r *http.Request,
) {
	idParam := chi.URLParam(r, "id")

	targetID, err := strconv.Atoi(idParam)
	if err != nil || targetID <= 0 {
		http.Error(
			w,
			"invalid target id",
			http.StatusBadRequest,
		)
		return
	}

	limit := 20

	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil || limit <= 0 || limit > 100 {
			http.Error(
				w,
				"limit must be between 1 and 100",
				http.StatusBadRequest,
			)
			return
		}
	}

	incidents, err := h.listIncidents.ListIncidents(
		r.Context(),
		targetID,
		limit,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	response := make(
		[]incidentResponse,
		0,
		len(incidents),
	)

	for _, incident := range incidents {
		var closedAt *time.Time

		if !incident.ClosedAt.IsZero() {
			value := incident.ClosedAt
			closedAt = &value
		}

		response = append(
			response,
			incidentResponse{
				ID:        incident.ID,
				TargetID:  incident.TargetID,
				Status:    incident.Status,
				CreatedAt: incident.CreatedAt,
				ClosedAt:  closedAt,
			},
		)
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
