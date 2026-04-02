package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/PipeM113/bakery-manager/internal/fixed_costs/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

type FixedCostHandler struct {
	svc *service.FixedCostService
}

func NewFixedCostHandler(svc *service.FixedCostService) *FixedCostHandler {
	return &FixedCostHandler{svc: svc}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *FixedCostHandler) List(w http.ResponseWriter, r *http.Request) {
	costs, err := h.svc.List(r.Context(), userID(r))
	if err != nil {
		httputil.JSONError(w, "error obteniendo costos fijos", http.StatusInternalServerError)
		return
	}
	if costs == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(costs)
}

func (h *FixedCostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string  `json:"name"`
		MonthlyAmount float64 `json:"monthly_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || body.MonthlyAmount <= 0 {
		httputil.JSONError(w, "nombre y monto son requeridos", http.StatusBadRequest)
		return
	}

	log.Printf("[fixed_costs] POST /fixed-costs user=%s name=%q amount=%.2f", userID(r), body.Name, body.MonthlyAmount)

	fc, err := h.svc.Create(r.Context(), userID(r), body.Name, body.MonthlyAmount)
	if err != nil {
		log.Printf("[fixed_costs] Create error user=%s: %v", userID(r), err)
		httputil.JSONError(w, "error creando costo fijo", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(fc)
}

func (h *FixedCostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name          string  `json:"name"`
		MonthlyAmount float64 `json:"monthly_amount"`
		IsActive      bool    `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || body.MonthlyAmount <= 0 {
		httputil.JSONError(w, "nombre y monto son requeridos", http.StatusBadRequest)
		return
	}

	log.Printf("[fixed_costs] PUT /fixed-costs/%s user=%s name=%q active=%v", id, userID(r), body.Name, body.IsActive)

	fc, err := h.svc.Update(r.Context(), id, userID(r), body.Name, body.MonthlyAmount, body.IsActive)
	if err != nil {
		log.Printf("[fixed_costs] Update error id=%s user=%s: %v", id, userID(r), err)
		httputil.JSONError(w, "error actualizando costo fijo", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fc)
}

func (h *FixedCostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Printf("[fixed_costs] DELETE /fixed-costs/%s user=%s", id, userID(r))
	if err := h.svc.Delete(r.Context(), id, userID(r)); err != nil {
		log.Printf("[fixed_costs] Delete error id=%s user=%s: %v", id, userID(r), err)
		httputil.JSONError(w, "error eliminando costo fijo", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
