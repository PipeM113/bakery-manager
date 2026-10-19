package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/internal/stock_adjustments/repository"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *repository.Repository
	clk  clock.Clock
}

func New(repo *repository.Repository, clk clock.Clock) *Handler {
	return &Handler{repo: repo, clk: clk}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Create records an adjustment: merma (quantity), correccion (counted_quantity) or otro (quantity_delta).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason          string   `json:"reason"`
		Quantity        *float64 `json:"quantity"`
		CountedQuantity *float64 `json:"counted_quantity"`
		QuantityDelta   *float64 `json:"quantity_delta"`
		Unit            string   `json:"unit"`
		Note            string   `json:"note"`
		AdjustedOn      string   `json:"adjusted_on"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	today := h.clk.Today()
	if body.AdjustedOn == "" {
		body.AdjustedOn = today
	}
	if _, err := time.Parse("2006-01-02", body.AdjustedOn); err != nil {
		httputil.JSONError(w, "la fecha debe tener formato AAAA-MM-DD", http.StatusBadRequest)
		return
	}
	if body.AdjustedOn > today {
		httputil.JSONError(w, "la fecha no puede ser futura", http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	log.Printf("[stock] POST /ingredients/%s/stock-adjustments user=%s reason=%q", id, userID(r), body.Reason)
	a, err := h.repo.Record(r.Context(), repository.Input{
		UserID: userID(r), IngredientID: id, Reason: body.Reason,
		Quantity: body.Quantity, CountedQuantity: body.CountedQuantity, QuantityDelta: body.QuantityDelta,
		Unit: body.Unit, Note: body.Note, AdjustedOn: body.AdjustedOn,
	})
	var invalid *repository.ValidationError
	switch {
	case errors.As(err, &invalid):
		httputil.JSONError(w, invalid.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, repository.ErrNotFound):
		httputil.JSONError(w, "ingrediente no encontrado", http.StatusNotFound)
		return
	case err != nil:
		log.Printf("[stock] Record error: %v", err)
		httputil.JSONError(w, "error registrando el ajuste", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (h *Handler) write(w http.ResponseWriter, list []repository.Adjustment, err error) {
	if err != nil {
		log.Printf("[stock] list error: %v", err)
		httputil.JSONError(w, "error obteniendo ajustes", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// ForIngredient lists the history of one ingredient.
func (h *Handler) ForIngredient(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.ForIngredient(r.Context(), chi.URLParam(r, "id"))
	h.write(w, list, err)
}

// ForMonth lists what the user adjusted in a month (this one by default).
func (h *Handler) ForMonth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	month := h.clk.Today()[:7]
	if q.Has("month") {
		month = q.Get("month")
		if !monthPattern.MatchString(month) {
			httputil.JSONError(w, "el mes debe tener formato AAAA-MM", http.StatusBadRequest)
			return
		}
	}
	list, err := h.repo.ForMonth(r.Context(), userID(r), month+"-01")
	h.write(w, list, err)
}
