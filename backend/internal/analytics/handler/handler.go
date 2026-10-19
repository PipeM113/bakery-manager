package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/PipeM113/bakery-manager/internal/analytics/service"
	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
	clk clock.Clock
}

func NewAnalyticsHandler(svc *service.AnalyticsService, clk clock.Clock) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc, clk: clk}
}

// Limits of the trend, in months.
const (
	defaultTrendMonths = 6
	maxTrendMonths     = 24
)

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *AnalyticsHandler) parseMonthYear(r *http.Request) (month, year int) {
	month, year = h.clk.Month()
	if m, err := strconv.Atoi(r.URL.Query().Get("month")); err == nil && m >= 1 && m <= 12 {
		month = m
	}
	if y, err := strconv.Atoi(r.URL.Query().Get("year")); err == nil && y > 2000 {
		year = y
	}
	return
}

// GET /api/analytics/monthly?month=3&year=2024
func (h *AnalyticsHandler) Monthly(w http.ResponseWriter, r *http.Request) {
	month, year := h.parseMonthYear(r)
	metrics, err := h.svc.GetMonthlyMetrics(r.Context(), userID(r), month, year)
	if err != nil {
		httputil.JSONError(w, "error calculando métricas mensuales", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GET /api/analytics/recipes?month=3&year=2024
func (h *AnalyticsHandler) Recipes(w http.ResponseWriter, r *http.Request) {
	month, year := h.parseMonthYear(r)
	metrics, err := h.svc.GetRecipeMetrics(r.Context(), userID(r), month, year)
	if err != nil {
		httputil.JSONError(w, "error calculando métricas por receta", http.StatusInternalServerError)
		return
	}
	if metrics == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GET /api/analytics/trends?months=6
func (h *AnalyticsHandler) Trends(w http.ResponseWriter, r *http.Request) {
	months := defaultTrendMonths
	if raw := r.URL.Query().Get("months"); raw != "" {
		m, err := strconv.Atoi(raw)
		if err != nil || m < 1 || m > maxTrendMonths {
			httputil.JSONError(w, "months debe ser un número entero entre 1 y 24", http.StatusBadRequest)
			return
		}
		months = m
	}
	trends, err := h.svc.GetTrendData(r.Context(), userID(r), clock.LastMonths(h.clk(), months))
	if err != nil {
		httputil.JSONError(w, "error calculando tendencias", http.StatusInternalServerError)
		return
	}
	if trends == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}
