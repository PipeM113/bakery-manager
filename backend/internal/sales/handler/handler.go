package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/PipeM113/bakery-manager/internal/sales/repository"
	"github.com/PipeM113/bakery-manager/internal/sales/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

type SaleHandler struct {
	svc *service.SaleService
}

func NewSaleHandler(svc *service.SaleService) *SaleHandler {
	return &SaleHandler{svc: svc}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *SaleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RecipeID     string  `json:"recipe_id"`
		QuantitySold int     `json:"quantity_sold"`
		UnitPrice    float64 `json:"unit_price"`
		SaleDate     string  `json:"sale_date"`
		Notes        string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("[sales] decode body error: %v", err)
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	log.Printf("[sales] POST /sales user=%s recipe_id=%q qty=%d price=%.2f date=%q",
		userID(r), body.RecipeID, body.QuantitySold, body.UnitPrice, body.SaleDate)

	if body.RecipeID == "" || body.QuantitySold <= 0 || body.UnitPrice < 0 {
		msg := "receta, cantidad y precio son requeridos"
		log.Printf("[sales] validación fallida: %s", msg)
		httputil.JSONError(w, msg, http.StatusBadRequest)
		return
	}

	input := repository.RegisterSaleInput{
		UserID:       userID(r),
		RecipeID:     body.RecipeID,
		QuantitySold: body.QuantitySold,
		UnitPrice:    body.UnitPrice,
		SaleDate:     body.SaleDate,
		Notes:        body.Notes,
	}

	sale, err := h.svc.RegisterSale(r.Context(), input)
	if err != nil {
		var stockErr *repository.InsufficientStockError
		if errors.As(err, &stockErr) {
			httputil.JSONError(w, stockErr.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("[sales] RegisterSale error: %v", err)
		httputil.JSONError(w, "error registrando venta", http.StatusInternalServerError)
		return
	}
	log.Printf("[sales] venta registrada id=%s recipe=%s qty=%d", sale.ID, sale.RecipeName, sale.QuantitySold)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sale)
}

func (h *SaleHandler) List(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	sales, err := h.svc.GetAll(r.Context(), userID(r), from, to)
	if err != nil {
		httputil.JSONError(w, "error obteniendo ventas", http.StatusInternalServerError)
		return
	}
	if sales == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sales)
}

func (h *SaleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Printf("[sales] DELETE /sales/%s user=%s", id, userID(r))
	if err := h.svc.DeleteSale(r.Context(), id, userID(r)); err != nil {
		if err.Error() == "sale not found" {
			httputil.JSONError(w, "venta no encontrada", http.StatusNotFound)
			return
		}
		log.Printf("[sales] DeleteSale error id=%s: %v", id, err)
		httputil.JSONError(w, "error eliminando venta", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
