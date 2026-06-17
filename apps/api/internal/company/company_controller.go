package company

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type CompanyController struct {
	service *CompanyService
}

func NewCompanyController(service *CompanyService) *CompanyController {
	return &CompanyController{service: service}
}

func (h *CompanyController) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/companies", h.ListCompanies).Methods("GET")
	router.HandleFunc("/api/companies", h.CreateCompany).Methods("POST")
	router.HandleFunc("/api/companies/{code}", h.GetCompany).Methods("GET")
	router.HandleFunc("/api/companies/{code}/roles", h.AddRole).Methods("POST")
	router.HandleFunc("/api/companies/{code}/roles/{role}", h.RemoveRole).Methods("DELETE")
	router.HandleFunc("/api/companies/{code}/action-types", h.ListActionTypes).Methods("GET")
	router.HandleFunc("/api/companies/{code}/action-types", h.CreateActionType).Methods("POST")
	router.HandleFunc("/api/companies/{code}/action-types/{action}", h.UpdateActionTypeKeyword).Methods("PUT")
	router.HandleFunc("/api/companies/{code}/action-types/{action}", h.DeleteActionType).Methods("DELETE")
}

func (h *CompanyController) ListCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := h.service.ListCompanies(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(companies)
}

func (h *CompanyController) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompanyCode string `json:"company_code"`
		CompanyName string `json:"company_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	company, err := h.service.CreateCompany(r.Context(), req.CompanyCode, req.CompanyName)
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
	json.NewEncoder(w).Encode(company)
}

func (h *CompanyController) GetCompany(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	company, err := h.service.GetCompany(r.Context(), code)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(company)
}

func (h *CompanyController) AddRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	var req struct {
		RoleName   string  `json:"role_name"`
		HourlyRate float64 `json:"hourly_rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.AddRole(r.Context(), code, req.RoleName, req.HourlyRate)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrRoleAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CompanyController) RemoveRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	role := vars["role"]

	err := h.service.RemoveRole(r.Context(), code, role)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrRoleNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CompanyController) ListActionTypes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	actionTypes, err := h.service.ListActionTypes(r.Context(), code)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actionTypes)
}

func (h *CompanyController) CreateActionType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	var req struct {
		ActionType string `json:"action_type"`
		Keyword    string `json:"keyword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.CreateActionType(r.Context(), code, req.ActionType, req.Keyword)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrActionTypeAlreadyExists) || errors.Is(err, ErrKeywordAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrInvalidActionTypeName) || errors.Is(err, ErrInvalidKeyword) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *CompanyController) UpdateActionTypeKeyword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	action := vars["action"]

	var req struct {
		Keyword string `json:"keyword"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.UpdateActionTypeKeyword(r.Context(), code, action, req.Keyword)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrActionTypeNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrKeywordAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, ErrInvalidKeyword) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CompanyController) DeleteActionType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]
	action := vars["action"]

	err := h.service.DeleteActionType(r.Context(), code, action)
	if err != nil {
		if shared.IsNotFound(err) || errors.Is(err, ErrActionTypeNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, ErrCannotDeleteSystemActionType) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
