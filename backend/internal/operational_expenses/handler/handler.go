package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PipeM113/bakery-manager/internal/operational_expenses/repository"
	"github.com/PipeM113/bakery-manager/internal/operational_expenses/service"
	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/PipeM113/bakery-manager/pkg/money"
	"github.com/go-chi/chi/v5"
)

type ExpenseHandler struct {
	svc *service.ExpenseService
	clk clock.Clock
}

func NewExpenseHandler(svc *service.ExpenseService, clk clock.Clock) *ExpenseHandler {
	return &ExpenseHandler{svc: svc, clk: clk}
}

// checkExpenseDate accepts a real date that is not later than today in Santiago, whatever the hour.
func (h *ExpenseHandler) checkExpenseDate(value string) string {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "fecha inválida, use formato YYYY-MM-DD"
	}
	if value > h.clk.Today() { // same-length ISO dates sort like calendar dates
		return "la fecha no puede ser futura"
	}
	return ""
}

var categoryIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func writeExpenseError(w http.ResponseWriter, err error, fallback string) {
	if errors.Is(err, repository.ErrUnknownCategory) {
		httputil.JSONError(w, "la categoría no existe", http.StatusBadRequest)
		return
	}
	httputil.JSONError(w, fallback, http.StatusInternalServerError)
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Description string `json:"description"`
		Amount      int64  `json:"amount"`
		CategoryID  string `json:"category_id"`
		ExpenseDate string `json:"expense_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Description = strings.TrimSpace(body.Description)
	body.CategoryID = strings.TrimSpace(body.CategoryID)
	if body.Description == "" || body.CategoryID == "" || body.ExpenseDate == "" {
		httputil.JSONError(w, "descripción, monto, categoría y fecha son requeridos", http.StatusBadRequest)
		return
	}
	if !money.Valid(body.Amount, false) {
		httputil.JSONError(w, "el monto debe ser un número entero de pesos, mayor a 0 y hasta 2.000.000.000", http.StatusBadRequest)
		return
	}
	if !categoryIDPattern.MatchString(body.CategoryID) {
		httputil.JSONError(w, "categoría inválida", http.StatusBadRequest)
		return
	}
	if msg := h.checkExpenseDate(body.ExpenseDate); msg != "" {
		httputil.JSONError(w, msg, http.StatusBadRequest)
		return
	}

	log.Printf("[expenses] POST /expenses user=%s amount=%d date=%s", userID(r), body.Amount, body.ExpenseDate)

	expense, err := h.svc.Create(r.Context(), userID(r), body.Description, body.CategoryID, body.ExpenseDate, body.Amount)
	if err != nil {
		log.Printf("[expenses] Create error user=%s: %v", userID(r), err)
		writeExpenseError(w, err, "error creando gasto")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	month, year := h.clk.Month()
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y > 2000 {
			year = y
		}
	}

	expenses, err := h.svc.ListByMonth(r.Context(), month, year)
	if err != nil {
		httputil.JSONError(w, "error obteniendo gastos", http.StatusInternalServerError)
		return
	}
	if expenses == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expenses)
}

func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Description string `json:"description"`
		Amount      int64  `json:"amount"`
		CategoryID  string `json:"category_id"`
		ExpenseDate string `json:"expense_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Description = strings.TrimSpace(body.Description)
	body.CategoryID = strings.TrimSpace(body.CategoryID)
	if body.Description == "" || body.CategoryID == "" || body.ExpenseDate == "" {
		httputil.JSONError(w, "descripción, monto, categoría y fecha son requeridos", http.StatusBadRequest)
		return
	}
	if !money.Valid(body.Amount, false) {
		httputil.JSONError(w, "el monto debe ser un número entero de pesos, mayor a 0 y hasta 2.000.000.000", http.StatusBadRequest)
		return
	}
	if !categoryIDPattern.MatchString(body.CategoryID) {
		httputil.JSONError(w, "categoría inválida", http.StatusBadRequest)
		return
	}
	if msg := h.checkExpenseDate(body.ExpenseDate); msg != "" {
		httputil.JSONError(w, msg, http.StatusBadRequest)
		return
	}

	log.Printf("[expenses] PUT /expenses/%s user=%s amount=%d", id, userID(r), body.Amount)

	expense, err := h.svc.Update(r.Context(), id, body.Description, body.CategoryID, body.ExpenseDate, body.Amount)
	if err != nil {
		log.Printf("[expenses] Update error id=%s user=%s: %v", id, userID(r), err)
		writeExpenseError(w, err, "error actualizando gasto")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Printf("[expenses] DELETE /expenses/%s user=%s", id, userID(r))
	if err := h.svc.Delete(r.Context(), id); err != nil {
		log.Printf("[expenses] Delete error id=%s user=%s: %v", id, userID(r), err)
		httputil.JSONError(w, "error eliminando gasto", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
