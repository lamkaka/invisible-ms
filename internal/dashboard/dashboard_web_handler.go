package dashboard

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type DashboardWebHandler struct {
	service  *DashboardService
	template *template.Template
}

func NewDashboardWebHandler(service *DashboardService, templateDir string) (*DashboardWebHandler, error) {
	tmpl, err := template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return nil, err
	}

	return &DashboardWebHandler{
		service:  service,
		template: tmpl,
	}, nil
}

func (h *DashboardWebHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/dashboard", h.DashboardPage).Methods("GET")
	router.HandleFunc("/workers", h.WorkersPage).Methods("GET")
}

func (h *DashboardWebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	stats, err := h.service.GetStats(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Stats *DashboardStats
	}{
		Stats: stats,
	}

	h.template.ExecuteTemplate(w, "dashboard.html", data)
}

func (h *DashboardWebHandler) WorkersPage(w http.ResponseWriter, r *http.Request) {
	h.template.ExecuteTemplate(w, "workers.html", nil)
}
