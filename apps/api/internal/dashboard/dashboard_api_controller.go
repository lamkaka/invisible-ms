package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type DashboardAPIController struct {
	service *DashboardService
}

func NewDashboardAPIController(service *DashboardService) *DashboardAPIController {
	return &DashboardAPIController{service: service}
}

func (h *DashboardAPIController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dashboard/stats", h.GetStats).Methods("GET")
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
