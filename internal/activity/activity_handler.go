package activity

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type ActivityHandler struct {
	webhookService *WebhookService
	sessionService *SessionService
	webhookSecret  string
}

func NewActivityHandler(webhookService *WebhookService, sessionService *SessionService, webhookSecret string) *ActivityHandler {
	return &ActivityHandler{
		webhookService: webhookService,
		sessionService: sessionService,
		webhookSecret:  webhookSecret,
	}
}

func (h *ActivityHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/webhook/message", h.HandleWebhook).Methods("POST")
	router.HandleFunc("/api/activities", h.ListActivities).Methods("GET")
	router.HandleFunc("/api/activities/sessions", h.ListSessions).Methods("GET")
}

func (h *ActivityHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	secret := r.Header.Get("X-Webhook-Secret")
	if subtle.ConstantTimeCompare([]byte(secret), []byte(h.webhookSecret)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log, err := h.webhookService.ProcessWebhook(r.Context(), payload)
	if err != nil {
		switch {
		case shared.IsNotFound(err):
			http.Error(w, err.Error(), http.StatusNotFound)
		case shared.IsAlreadyExists(err):
			http.Error(w, err.Error(), http.StatusConflict)
		case shared.IsInvalidInput(err):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(log)
}

func (h *ActivityHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	staffID := r.URL.Query().Get("staff_id")
	companyCode := r.URL.Query().Get("company_code")

	if staffID == "" && companyCode == "" {
		http.Error(w, "either staff_id or company_code is required", http.StatusBadRequest)
		return
	}

	from, err := parseTimeParam(r.URL.Query().Get("from"), time.Now().AddDate(0, 0, -7))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to, err := parseTimeParam(r.URL.Query().Get("to"), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logs, err := h.sessionService.GetActivities(r.Context(), staffID, companyCode, from, to)
	if err != nil {
		switch {
		case shared.IsInvalidInput(err):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *ActivityHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	from, err := parseTimeParam(r.URL.Query().Get("from"), time.Now().AddDate(0, 0, -7))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to, err := parseTimeParam(r.URL.Query().Get("to"), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessions, err := h.sessionService.GetSessions(r.Context(), companyCode, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// parseTimeParam parses a RFC3339 time string. If the string is empty, it
// returns the provided default value. If the string is non-empty and cannot be
// parsed, it returns an error.
func parseTimeParam(s string, defaultVal time.Time) (time.Time, error) {
	if s == "" {
		return defaultVal, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format %q: %w", s, shared.ErrInvalidInput)
	}
	return t, nil
}
