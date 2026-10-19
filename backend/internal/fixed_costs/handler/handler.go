package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PipeM113/bakery-manager/internal/fixed_costs/repository"
	"github.com/PipeM113/bakery-manager/internal/fixed_costs/service"
	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/PipeM113/bakery-manager/pkg/money"
	"github.com/go-chi/chi/v5"
)

type FixedCostHandler struct {
	svc *service.FixedCostService
	clk clock.Clock
}

func NewFixedCostHandler(svc *service.FixedCostService, clk clock.Clock) *FixedCostHandler {
	return &FixedCostHandler{svc: svc, clk: clk}
}

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

const monthMessage = "el mes debe tener formato AAAA-MM"

// monthStart turns "YYYY-MM" into the first day of that month; an empty text is this month in
// Santiago. ok is false when the text is not a month.
func (h *FixedCostHandler) monthStart(text string, given bool) (string, bool) {
	if !given {
		return h.clk.Today()[:7] + "-01", true
	}
	if !monthPattern.MatchString(text) {
		return "", false
	}
	return text + "-01", true
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *FixedCostHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	month, ok := h.monthStart(q.Get("month"), q.Has("month"))
	if !ok {
		httputil.JSONError(w, monthMessage, http.StatusBadRequest)
		return
	}
	costs, err := h.svc.List(r.Context(), userID(r), month)
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
		MonthlyAmount int64   `json:"monthly_amount"`
		ValidFrom     *string `json:"valid_from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		httputil.JSONError(w, "nombre y monto son requeridos", http.StatusBadRequest)
		return
	}
	if !money.Valid(body.MonthlyAmount, false) {
		httputil.JSONError(w, "el monto debe ser un número entero de pesos, mayor a 0 y hasta 2.000.000.000", http.StatusBadRequest)
		return
	}
	validFrom, ok := h.monthStart(deref(body.ValidFrom), body.ValidFrom != nil)
	if !ok {
		httputil.JSONError(w, monthMessage, http.StatusBadRequest)
		return
	}

	log.Printf("[fixed_costs] POST /fixed-costs user=%s name=%q amount=%d", userID(r), body.Name, body.MonthlyAmount)

	fc, err := h.svc.Create(r.Context(), userID(r), body.Name, body.MonthlyAmount, validFrom)
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
		MonthlyAmount int64   `json:"monthly_amount"`
		IsActive      *bool   `json:"is_active"`
		EffectiveFrom *string `json:"effective_from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		httputil.JSONError(w, "nombre y monto son requeridos", http.StatusBadRequest)
		return
	}
	if !money.Valid(body.MonthlyAmount, false) {
		httputil.JSONError(w, "el monto debe ser un número entero de pesos, mayor a 0 y hasta 2.000.000.000", http.StatusBadRequest)
		return
	}

	month, ok := h.monthStart(deref(body.EffectiveFrom), body.EffectiveFrom != nil)
	if !ok {
		httputil.JSONError(w, monthMessage, http.StatusBadRequest)
		return
	}
	active := body.IsActive == nil || *body.IsActive

	log.Printf("[fixed_costs] PUT /fixed-costs/%s user=%s name=%q active=%v from=%s", id, userID(r), body.Name, active, month)

	fc, err := h.svc.Update(r.Context(), id, userID(r), body.Name, body.MonthlyAmount, active, month)
	if errors.Is(err, repository.ErrNotFound) {
		httputil.JSONError(w, "costo fijo no encontrado", http.StatusNotFound)
		return
	}
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
	month, _ := h.monthStart("", false)
	err := h.svc.End(r.Context(), id, userID(r), month)
	if errors.Is(err, repository.ErrNotFound) {
		httputil.JSONError(w, "costo fijo no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[fixed_costs] Delete error id=%s user=%s: %v", id, userID(r), err)
		httputil.JSONError(w, "error eliminando costo fijo", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
