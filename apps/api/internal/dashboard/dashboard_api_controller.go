package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type DashboardAPIController struct {
	service *DashboardService
}

func NewDashboardAPIController(service *DashboardService) *DashboardAPIController {
	return &DashboardAPIController{service: service}
}

func (h *DashboardAPIController) RegisterRoutes(r chi.Router) {
	r.Get("/api/dashboard/stats", h.GetStats)
}

func (h *DashboardAPIController) GetStats(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	stats, err := h.service.GetStats(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
