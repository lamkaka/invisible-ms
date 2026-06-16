package dashboard

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type DashboardWebHandler struct {
	service     *DashboardService
	templateDir string
}

func NewDashboardWebHandler(service *DashboardService, templateDir string) (*DashboardWebHandler, error) {
	return &DashboardWebHandler{
		service:     service,
		templateDir: templateDir,
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

	tmpl, err := template.ParseFiles(
		filepath.Join(h.templateDir, "layout.html"),
		filepath.Join(h.templateDir, "dashboard.html"),
	)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "layout.html", data)
}

func (h *DashboardWebHandler) WorkersPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		filepath.Join(h.templateDir, "layout.html"),
		filepath.Join(h.templateDir, "workers.html"),
	)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "layout.html", nil)
}
