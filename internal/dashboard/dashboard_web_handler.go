package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type DashboardWebHandler struct {
	service       *DashboardService
	templateDir   string
	dashboardTmpl *template.Template
	staffTmpl     *template.Template
	actionsTmpl   *template.Template
}

func NewDashboardWebHandler(service *DashboardService, templateDir string) (*DashboardWebHandler, error) {
	dashboardTmpl, err := template.ParseFiles(
		filepath.Join(templateDir, "layout.html"),
		filepath.Join(templateDir, "dashboard.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dashboard templates: %w", err)
	}

	staffTmpl, err := template.ParseFiles(
		filepath.Join(templateDir, "layout.html"),
		filepath.Join(templateDir, "staff.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse staff templates: %w", err)
	}

	actionsTmpl, err := template.ParseFiles(
		filepath.Join(templateDir, "layout.html"),
		filepath.Join(templateDir, "actions.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actions templates: %w", err)
	}

	return &DashboardWebHandler{
		service:       service,
		templateDir:   templateDir,
		dashboardTmpl: dashboardTmpl,
		staffTmpl:     staffTmpl,
		actionsTmpl:   actionsTmpl,
	}, nil
}

func (h *DashboardWebHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/dashboard", h.DashboardPage).Methods("GET")
	router.HandleFunc("/staff", h.StaffPage).Methods("GET")
	router.HandleFunc("/actions", h.ActionsPage).Methods("GET")
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

	if err := h.dashboardTmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *DashboardWebHandler) StaffPage(w http.ResponseWriter, r *http.Request) {
	if err := h.staffTmpl.ExecuteTemplate(w, "staff.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *DashboardWebHandler) ActionsPage(w http.ResponseWriter, r *http.Request) {
	if err := h.actionsTmpl.ExecuteTemplate(w, "actions.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
