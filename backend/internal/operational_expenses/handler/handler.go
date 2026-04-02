package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PipeM113/bakery-manager/internal/operational_expenses/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

// santiagoLoc is loaded once at startup; falls back to UTC if unavailable.
var santiagoLoc *time.Location

func init() {
	var err error
	santiagoLoc, err = time.LoadLocation("America/Santiago")
	if err != nil {
		log.Printf("[expenses] WARNING: could not load America/Santiago timezone: %v — falling back to UTC", err)
		santiagoLoc = time.UTC
	}
}

type ExpenseHandler struct {
	svc *service.ExpenseService
}

func NewExpenseHandler(svc *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
		Category    string  `json:"category"`
		ExpenseDate string  `json:"expense_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Description = strings.TrimSpace(body.Description)
	body.Category = strings.TrimSpace(body.Category)
	if body.Description == "" || body.Amount <= 0 || body.Category == "" || body.ExpenseDate == "" {
		httputil.JSONError(w, "descripción, monto, categoría y fecha son requeridos", http.StatusBadRequest)
		return
	}
	if body.Category != "ingredientes" && body.Category != "servicios" && body.Category != "otros" {
		httputil.JSONError(w, "categoría inválida", http.StatusBadRequest)
		return
	}
	date, err := time.ParseInLocation("2006-01-02", body.ExpenseDate, santiagoLoc)
	if err != nil {
		httputil.JSONError(w, "fecha inválida, use formato YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	todaySantiago := time.Now().In(santiagoLoc).Truncate(24 * time.Hour)
	if date.After(todaySantiago) {
		httputil.JSONError(w, "la fecha no puede ser futura", http.StatusBadRequest)
		return
	}

	log.Printf("[expenses] POST /expenses user=%s description=%q amount=%.2f date=%s", userID(r), body.Description, body.Amount, body.ExpenseDate)

	expense, err := h.svc.Create(r.Context(), userID(r), body.Description, body.Category, body.ExpenseDate, body.Amount)
	if err != nil {
		log.Printf("[expenses] Create error user=%s: %v", userID(r), err)
		httputil.JSONError(w, "error creando gasto", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	now := time.Now().In(santiagoLoc)
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	month := int(now.Month())
	year := now.Year()
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

	expenses, err := h.svc.ListByMonth(r.Context(), userID(r), month, year)
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
		Description string  `json:"description"`
		Amount      float64 `json:"amount"`
		Category    string  `json:"category"`
		ExpenseDate string  `json:"expense_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	body.Description = strings.TrimSpace(body.Description)
	body.Category = strings.TrimSpace(body.Category)
	if body.Description == "" || body.Amount <= 0 || body.Category == "" || body.ExpenseDate == "" {
		httputil.JSONError(w, "descripción, monto, categoría y fecha son requeridos", http.StatusBadRequest)
		return
	}
	if body.Category != "ingredientes" && body.Category != "servicios" && body.Category != "otros" {
		httputil.JSONError(w, "categoría inválida", http.StatusBadRequest)
		return
	}
	date, err := time.ParseInLocation("2006-01-02", body.ExpenseDate, santiagoLoc)
	if err != nil {
		httputil.JSONError(w, "fecha inválida, use formato YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	todaySantiago := time.Now().In(santiagoLoc).Truncate(24 * time.Hour)
	if date.After(todaySantiago) {
		httputil.JSONError(w, "la fecha no puede ser futura", http.StatusBadRequest)
		return
	}

	log.Printf("[expenses] PUT /expenses/%s user=%s description=%q amount=%.2f", id, userID(r), body.Description, body.Amount)

	expense, err := h.svc.Update(r.Context(), id, userID(r), body.Description, body.Category, body.ExpenseDate, body.Amount)
	if err != nil {
		log.Printf("[expenses] Update error id=%s user=%s: %v", id, userID(r), err)
		httputil.JSONError(w, "error actualizando gasto", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expense)
}

func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Printf("[expenses] DELETE /expenses/%s user=%s", id, userID(r))
	if err := h.svc.Delete(r.Context(), id, userID(r)); err != nil {
		log.Printf("[expenses] Delete error id=%s user=%s: %v", id, userID(r), err)
		httputil.JSONError(w, "error eliminando gasto", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
