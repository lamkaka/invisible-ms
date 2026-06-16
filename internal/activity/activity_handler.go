package activity

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
	if secret != h.webhookSecret {
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(log)
}

func (h *ActivityHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	staffID := r.URL.Query().Get("staff_id")
	companyCode := r.URL.Query().Get("company_code")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, _ := time.Parse(time.RFC3339, fromStr)
	to, _ := time.Parse(time.RFC3339, toStr)

	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -7)
	}
	if to.IsZero() {
		to = time.Now()
	}

	logs, err := h.sessionService.GetActivities(r.Context(), staffID, companyCode, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *ActivityHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, _ := time.Parse(time.RFC3339, fromStr)
	to, _ := time.Parse(time.RFC3339, toStr)

	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -7)
	}
	if to.IsZero() {
		to = time.Now()
	}

	sessions, err := h.sessionService.GetSessions(r.Context(), companyCode, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}
