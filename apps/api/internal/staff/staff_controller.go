package staff

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type StaffController struct {
	service *StaffService
}

func NewStaffController(service *StaffService) *StaffController {
	return &StaffController{service: service}
}

func (h *StaffController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/staff", h.ListStaff).Methods("GET")
	router.HandleFunc("/api/staff", h.CreateStaff).Methods("POST")
	router.HandleFunc("/api/staff/{id}", h.GetStaff).Methods("GET")
	router.HandleFunc("/api/staff/{id}/roles", h.AssignRole).Methods("POST")
	router.HandleFunc("/api/staff/{id}/roles/{role}", h.UnassignRole).Methods("DELETE")
}

func (h *StaffController) ListStaff(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")
	if companyCode == "" {
		http.Error(w, "company_code is required", http.StatusBadRequest)
		return
	}

	staff, err := h.service.ListStaff(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(staff)
}

func (h *StaffController) CreateStaff(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StaffID     string   `json:"staff_id"`
		PhoneNumber string   `json:"phone_number"`
		Name        string   `json:"name"`
		CompanyCode string   `json:"company_code"`
		Roles       []string `json:"roles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	staff, err := h.service.CreateStaff(r.Context(), req.StaffID, req.PhoneNumber, req.Name, req.CompanyCode, req.Roles)
	if err != nil {
		if shared.IsAlreadyExists(err) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(staff)
}

func (h *StaffController) GetStaff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	staff, err := h.service.GetStaff(r.Context(), id)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(staff)
}

func (h *StaffController) AssignRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		RoleName string `json:"role_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.AssignRole(r.Context(), id, req.RoleName)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrRoleAlreadyAssigned) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *StaffController) UnassignRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	role := vars["role"]

	err := h.service.UnassignRole(r.Context(), id, role)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
