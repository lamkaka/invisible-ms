package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type DashboardWebHandler struct {
	service     *DashboardService
	templateDir string
	templates   *template.Template
}

func NewDashboardWebHandler(service *DashboardService, templateDir string) (*DashboardWebHandler, error) {
	tmpl, err := template.ParseFiles(
		filepath.Join(templateDir, "layout.html"),
		filepath.Join(templateDir, "dashboard.html"),
		filepath.Join(templateDir, "workers.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &DashboardWebHandler{
		service:     service,
		templateDir: templateDir,
		templates:   tmpl,
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

	if err := h.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *DashboardWebHandler) WorkersPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "layout.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
