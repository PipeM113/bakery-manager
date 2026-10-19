package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PipeM113/bakery-manager/internal/sales/repository"
	"github.com/PipeM113/bakery-manager/internal/sales/service"
	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/PipeM113/bakery-manager/pkg/money"
	"github.com/go-chi/chi/v5"
)

const maxOrderLines = 50

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type OrderHandler struct {
	svc *service.SaleService
	clk clock.Clock
}

func NewOrderHandler(svc *service.SaleService, clk clock.Clock) *OrderHandler {
	return &OrderHandler{svc: svc, clk: clk}
}

// Create registers an order: several products, each a sale, with an optional total price of its own.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CustomerName string `json:"customer_name"`
		Notes        string `json:"notes"`
		OrderDate    string `json:"order_date"`
		TotalPrice   *int64 `json:"total_price"`
		Lines        []struct {
			RecipeID  string  `json:"recipe_id"`
			Quantity  int     `json:"quantity"`
			UnitPrice int64   `json:"unit_price"`
			SoldAs    *string `json:"sold_as"`
		} `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	bad := func(msg string) { httputil.JSONError(w, msg, http.StatusBadRequest) }

	if n := len(body.Lines); n < 1 || n > maxOrderLines {
		bad("el pedido debe tener entre 1 y 50 productos")
		return
	}
	if body.TotalPrice != nil && !money.Valid(*body.TotalPrice, true) {
		bad("el total debe ser un número entero de pesos, 0 o más y hasta 2.000.000.000")
		return
	}
	today := h.clk.Today()
	if body.OrderDate == "" {
		body.OrderDate = today
	}
	if _, err := time.Parse("2006-01-02", body.OrderDate); err != nil {
		bad("la fecha debe tener formato AAAA-MM-DD")
		return
	}
	if body.OrderDate > today {
		bad("la fecha no puede ser futura")
		return
	}

	lines := make([]repository.LineInput, len(body.Lines))
	for i, l := range body.Lines {
		soldAs := "recipe"
		if l.SoldAs != nil {
			soldAs = *l.SoldAs
		}
		switch {
		case l.RecipeID == "":
			bad("cada producto necesita una receta")
			return
		case l.Quantity <= 0:
			bad("la cantidad de cada producto debe ser un entero mayor a 0")
			return
		case !money.Valid(l.UnitPrice, true):
			bad("el precio de cada producto debe ser un número entero de pesos, 0 o más")
			return
		case soldAs != "recipe" && soldAs != "portion":
			bad(`"sold_as" debe ser "recipe" (receta completa) o "portion" (porción)`)
			return
		}
		lines[i] = repository.LineInput{RecipeID: l.RecipeID, QuantitySold: l.Quantity, UnitPrice: l.UnitPrice, SoldAs: soldAs}
	}

	log.Printf("[orders] POST /orders user=%s lines=%d", userID(r), len(lines))
	order, err := h.svc.CreateOrder(r.Context(), repository.OrderInput{
		CreatedBy: userID(r), CustomerName: strings.TrimSpace(body.CustomerName), Notes: strings.TrimSpace(body.Notes),
		OrderDate: body.OrderDate, TotalPrice: body.TotalPrice, Lines: lines,
	})
	var invalid *repository.ValidationError
	if errors.As(err, &invalid) {
		bad(invalid.Error())
		return
	}
	if err != nil {
		log.Printf("[orders] CreateOrder error: %v", err)
		httputil.JSONError(w, "error registrando el pedido", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

// List gives the orders of a month (this one by default), newest first.
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	month := h.clk.Today()[:7]
	if q.Has("month") {
		month = q.Get("month")
		if !monthPattern.MatchString(month) {
			httputil.JSONError(w, "el mes debe tener formato AAAA-MM", http.StatusBadRequest)
			return
		}
	}
	orders, err := h.svc.ListOrders(r.Context(), month+"-01")
	if err != nil {
		log.Printf("[orders] ListOrders error: %v", err)
		httputil.JSONError(w, "error obteniendo pedidos", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	order, err := h.svc.GetOrder(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, repository.ErrNotFound) {
		httputil.JSONError(w, "pedido no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[orders] GetOrder error: %v", err)
		httputil.JSONError(w, "error obteniendo el pedido", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// Delete cancels the whole order and gives back the stock its lines took.
func (h *OrderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Printf("[orders] DELETE /orders/%s user=%s", id, userID(r))
	err := h.svc.DeleteOrder(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		httputil.JSONError(w, "pedido no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("[orders] DeleteOrder error id=%s: %v", id, err)
		httputil.JSONError(w, "error anulando el pedido", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
