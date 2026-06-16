package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type DashboardAPIHandler struct {
	service *DashboardService
}

func NewDashboardAPIHandler(service *DashboardService) *DashboardAPIHandler {
	return &DashboardAPIHandler{service: service}
}

func (h *DashboardAPIHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dashboard/stats", h.GetStats).Methods("GET")
}

func (h *DashboardAPIHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	stats, err := h.service.GetStats(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
