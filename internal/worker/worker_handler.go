package worker

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/lamkaka/invisible-ms/internal/shared"
)

type WorkerHandler struct {
	service *WorkerService
}

func NewWorkerHandler(service *WorkerService) *WorkerHandler {
	return &WorkerHandler{service: service}
}

func (h *WorkerHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/workers", h.ListWorkers).Methods("GET")
	router.HandleFunc("/api/workers", h.CreateWorker).Methods("POST")
	router.HandleFunc("/api/workers/{id}", h.GetWorker).Methods("GET")
	router.HandleFunc("/api/workers/{id}/roles", h.AssignRole).Methods("POST")
	router.HandleFunc("/api/workers/{id}/roles/{role}", h.UnassignRole).Methods("DELETE")
}

func (h *WorkerHandler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	companyCode := r.URL.Query().Get("company_code")

	workers, err := h.service.ListWorkers(r.Context(), companyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func (h *WorkerHandler) CreateWorker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkerID    string   `json:"worker_id"`
		PhoneNumber string   `json:"phone_number"`
		Name        string   `json:"name"`
		CompanyCode string   `json:"company_code"`
		Roles       []string `json:"roles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	worker, err := h.service.CreateWorker(r.Context(), req.WorkerID, req.PhoneNumber, req.Name, req.CompanyCode, req.Roles)
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
	json.NewEncoder(w).Encode(worker)
}

func (h *WorkerHandler) GetWorker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	worker, err := h.service.GetWorker(r.Context(), id)
	if err != nil {
		if shared.IsNotFound(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worker)
}

func (h *WorkerHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
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

func (h *WorkerHandler) UnassignRole(w http.ResponseWriter, r *http.Request) {
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
