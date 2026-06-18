package dashboard

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

type DashboardWebController struct {
	service       *DashboardService
	templateDir   string
	dashboardTmpl *template.Template
	staffTmpl     *template.Template
	actionsTmpl   *template.Template
	rolesTmpl     *template.Template
}

func NewDashboardWebController(service *DashboardService, templateDir string) (*DashboardWebController, error) {
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

	rolesTmpl, err := template.ParseFiles(
		filepath.Join(templateDir, "layout.html"),
		filepath.Join(templateDir, "roles.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse roles templates: %w", err)
	}

	return &DashboardWebController{
		service:       service,
		templateDir:   templateDir,
		dashboardTmpl: dashboardTmpl,
		staffTmpl:     staffTmpl,
		actionsTmpl:   actionsTmpl,
		rolesTmpl:     rolesTmpl,
	}, nil
}

func (h *DashboardWebController) RegisterRoutes(r chi.Router) {
	r.Get("/dashboard", h.DashboardPage)
	r.Get("/staff", h.StaffPage)
	r.Get("/actions", h.ActionsPage)
	r.Get("/roles", h.RolesPage)
}

func (h *DashboardWebController) DashboardPage(w http.ResponseWriter, r *http.Request) {
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

func (h *DashboardWebController) StaffPage(w http.ResponseWriter, r *http.Request) {
	if err := h.staffTmpl.ExecuteTemplate(w, "staff.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *DashboardWebController) ActionsPage(w http.ResponseWriter, r *http.Request) {
	if err := h.actionsTmpl.ExecuteTemplate(w, "actions.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *DashboardWebController) RolesPage(w http.ResponseWriter, r *http.Request) {
	if err := h.rolesTmpl.ExecuteTemplate(w, "roles.html", nil); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
