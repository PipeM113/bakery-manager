package handler

import (
	"encoding/json"
	"net/http"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/costs/service"
	"github.com/go-chi/chi/v5"
)

type CostHandler struct {
	svc *service.CostService
}

func NewCostHandler(svc *service.CostService) *CostHandler {
	return &CostHandler{svc: svc}
}

func (h *CostHandler) GetBreakdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	breakdown, err := h.svc.GetCostBreakdown(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"receta no encontrada"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breakdown)
}

func (h *CostHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		MarginPct float64 `json:"margin_pct"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"request inválido"}`, http.StatusBadRequest)
		return
	}
	if req.MarginPct <= 0 {
		http.Error(w, `{"error":"margin_pct debe ser mayor a 0"}`, http.StatusBadRequest)
		return
	}

	breakdown, err := h.svc.GetCostBreakdown(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"receta no encontrada"}`, http.StatusNotFound)
		return
	}

	suggestion := domain.Simulate(breakdown, req.MarginPct)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestion)
}
