package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/costs/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type CostHandler struct {
	svc *service.CostService
}

func NewCostHandler(svc *service.CostService) *CostHandler {
	return &CostHandler{svc: svc}
}

func parseQueryFloat(r *http.Request, key string) float64 {
	v, err := strconv.ParseFloat(r.URL.Query().Get(key), 64)
	if err != nil {
		return 0
	}
	return v
}

func (h *CostHandler) GetBreakdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	params := service.CostParams{
		IndirectCostPct: parseQueryFloat(r, "indirect_cost_pct"),
		LaborCostPct:    parseQueryFloat(r, "labor_cost_pct"),
		MarginPct:       parseQueryFloat(r, "margin_pct"),
	}
	breakdown, err := h.svc.GetCostBreakdown(r.Context(), id, params)
	if err != nil {
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
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
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if req.MarginPct < 0 {
		httputil.JSONError(w, "margin_pct debe ser mayor o igual a 0", http.StatusBadRequest)
		return
	}

	breakdown, err := h.svc.GetCostBreakdown(r.Context(), id, service.CostParams{MarginPct: req.MarginPct})
	if err != nil {
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
		return
	}

	suggestion := domain.Simulate(breakdown, req.MarginPct)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestion)
}
