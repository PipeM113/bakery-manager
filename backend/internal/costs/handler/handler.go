package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/PipeM113/bakery-manager/internal/costs/domain"
	"github.com/PipeM113/bakery-manager/internal/costs/service"
	recipeRepo "github.com/PipeM113/bakery-manager/internal/recipes/repository"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

type CostHandler struct {
	svc        *service.CostService
	recipeRepo *recipeRepo.RecipeRepository
}

func NewCostHandler(svc *service.CostService, recipeRepo *recipeRepo.RecipeRepository) *CostHandler {
	return &CostHandler{svc: svc, recipeRepo: recipeRepo}
}

// queryPct reads an optional percentage. Absent gives nil; present but not a number is an error.
func queryPct(r *http.Request, key string) (*float64, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, &service.ParamError{Message: key + " no es un número válido"}
	}
	return &v, nil
}

// writeCostError maps a failure of the cost calculation to a response.
func writeCostError(w http.ResponseWriter, err error) {
	var pe *service.ParamError
	switch {
	case errors.As(err, &pe):
		httputil.JSONError(w, pe.Message, http.StatusBadRequest)
	case errors.Is(err, service.ErrInvalidYield):
		httputil.JSONError(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
	}
}

func parseQueryInt(r *http.Request, key string) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return 0
	}
	return v
}

func (h *CostHandler) GetBreakdown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	params := service.CostParams{
		ExtraCharge:  parseQueryInt(r, "extra_charge"),
		DeliveryCost: parseQueryInt(r, "delivery_cost"),
	}
	var err error
	if params.IndirectCostPct, err = queryPct(r, "indirect_cost_pct"); err != nil {
		writeCostError(w, err)
		return
	}
	if params.LaborCostPct, err = queryPct(r, "labor_cost_pct"); err != nil {
		writeCostError(w, err)
		return
	}
	if params.MarginPct, err = queryPct(r, "margin_pct"); err != nil {
		writeCostError(w, err)
		return
	}
	if err := params.Validate(); err != nil {
		writeCostError(w, err)
		return
	}
	breakdown, err := h.svc.GetCostBreakdown(r.Context(), id, params)
	if err != nil {
		writeCostError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breakdown)
}

func (h *CostHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	recipe, err := h.recipeRepo.GetByID(r.Context(), id)
	if err != nil {
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"indirect_cost_pct": recipe.IndirectCostPct,
		"labor_cost_pct":    recipe.LaborCostPct,
		"margin_pct":        recipe.MarginPct,
		"extra_charge":      recipe.ExtraCharge,
		"delivery_cost":     recipe.DeliveryCost,
	})
}

func (h *CostHandler) UpdateCosts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		IndirectCostPct float64 `json:"indirect_cost_pct"`
		LaborCostPct    float64 `json:"labor_cost_pct"`
		MarginPct       float64 `json:"margin_pct"`
		ExtraCharge     int     `json:"extra_charge"`
		DeliveryCost    int     `json:"delivery_cost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if err := (service.CostParams{
		IndirectCostPct: &req.IndirectCostPct, LaborCostPct: &req.LaborCostPct, MarginPct: &req.MarginPct,
	}).Validate(); err != nil {
		writeCostError(w, err)
		return
	}
	if req.ExtraCharge < 0 || req.DeliveryCost < 0 {
		httputil.JSONError(w, "los cobros no pueden ser negativos", http.StatusBadRequest)
		return
	}

	if err := h.recipeRepo.UpdateCosts(r.Context(), id, req.IndirectCostPct, req.LaborCostPct, req.MarginPct, req.ExtraCharge, req.DeliveryCost); err != nil {
		httputil.JSONError(w, "error actualizando costos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"indirect_cost_pct": req.IndirectCostPct,
		"labor_cost_pct":    req.LaborCostPct,
		"margin_pct":        req.MarginPct,
		"extra_charge":      req.ExtraCharge,
		"delivery_cost":     req.DeliveryCost,
	})
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
	if err := service.CheckPct("margin_pct", req.MarginPct, service.MaxMarkupPct); err != nil {
		writeCostError(w, err)
		return
	}

	breakdown, err := h.svc.GetCostBreakdown(r.Context(), id, service.CostParams{})
	if err != nil {
		writeCostError(w, err)
		return
	}

	suggestion := domain.Simulate(breakdown, req.MarginPct)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestion)
}
