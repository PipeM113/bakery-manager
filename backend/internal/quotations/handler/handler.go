package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	bizRepo "github.com/PipeM113/bakery-manager/internal/business/repository"
	"github.com/PipeM113/bakery-manager/internal/config"
	costSvc "github.com/PipeM113/bakery-manager/internal/costs/service"
	quoteRepo "github.com/PipeM113/bakery-manager/internal/quotations/repository"
	quoteSvc "github.com/PipeM113/bakery-manager/internal/quotations/service"
	"github.com/PipeM113/bakery-manager/internal/shared/clock"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/PipeM113/bakery-manager/pkg/money"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxLines    = 30
	maxQuantity = 100000
)

type QuotationHandler struct {
	costSvc  *costSvc.CostService
	repo     *quoteRepo.QuotationRepository
	business *bizRepo.Repository
	env      config.Business
	db       *pgxpool.Pool
	clk      clock.Clock
}

func NewQuotationHandler(cs *costSvc.CostService, db *pgxpool.Pool, clk clock.Clock, business *bizRepo.Repository, env config.Business) *QuotationHandler {
	return &QuotationHandler{costSvc: cs, repo: quoteRepo.NewQuotationRepository(db), business: business, env: env, db: db, clk: clk}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

type lineRequest struct {
	RecipeID        string   `json:"recipe_id"`
	Quantity        *float64 `json:"quantity"`
	SoldAs          string   `json:"sold_as"`
	MarginPct       *float64 `json:"margin_pct"`
	IndirectCostPct *float64 `json:"indirect_cost_pct"`
	LaborCostPct    *float64 `json:"labor_cost_pct"`
	ExtraCharge     *int64   `json:"extra_charge"`
}

type createRequest struct {
	ClientName   string        `json:"client_name"`
	DeliveryCost int64         `json:"delivery_cost"`
	Lines        []lineRequest `json:"lines"`
}

var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type badRequest struct {
	status  int
	message string
}

func (e *badRequest) Error() string { return e.message }

func bad(format string, a ...any) error {
	return &badRequest{http.StatusBadRequest, fmt.Sprintf(format, a...)}
}

// priceLine quotes one line with the cost calculation of the recipe screen. A whole recipe costs
// its suggested price; portions are priced as the recipe made for that many portions.
func (h *QuotationHandler) priceLine(r *http.Request, n int, l lineRequest) (quoteRepo.LineInput, error) {
	var out quoteRepo.LineInput
	if !uuidRE.MatchString(l.RecipeID) {
		return out, bad("la línea %d no tiene una receta válida", n)
	}
	if l.Quantity == nil || *l.Quantity != float64(int64(*l.Quantity)) || *l.Quantity < 1 || *l.Quantity > maxQuantity {
		return out, bad("la cantidad de la línea %d debe ser un entero entre 1 y %d", n, maxQuantity)
	}
	qty := int(*l.Quantity)
	soldAs := l.SoldAs
	switch soldAs {
	case "":
		soldAs = "recipe"
	case "recipe", "portion":
	default:
		return out, bad("la línea %d debe venderse como \"recipe\" o \"portion\"", n)
	}
	var extra int64
	if l.ExtraCharge != nil {
		extra = *l.ExtraCharge
	}
	if extra < 0 || extra > money.MaxPesos {
		return out, bad("el cobro extra de la línea %d no es válido", n)
	}
	params := costSvc.CostParams{
		IndirectCostPct: l.IndirectCostPct, LaborCostPct: l.LaborCostPct, MarginPct: l.MarginPct,
		ExtraCharge: int(extra),
	}
	if soldAs == "portion" {
		portions := float64(qty)
		params.Portions = &portions
	}
	if err := params.Validate(); err != nil {
		return out, bad("línea %d: %s", n, err.Error())
	}

	var name string
	var photo *string
	var indirect, labor, margin float64
	err := h.db.QueryRow(r.Context(), `
		SELECT name, photo_url, indirect_cost_pct, labor_cost_pct, margin_pct FROM recipes WHERE id = $1`, l.RecipeID,
	).Scan(&name, &photo, &indirect, &labor, &margin)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, bad("la receta de la línea %d no existe", n)
	}
	if err != nil {
		return out, err
	}
	if l.IndirectCostPct != nil {
		indirect = *l.IndirectCostPct
	}
	if l.LaborCostPct != nil {
		labor = *l.LaborCostPct
	}
	if l.MarginPct != nil {
		margin = *l.MarginPct
	}

	breakdown, err := h.costSvc.GetCostBreakdown(r.Context(), l.RecipeID, params)
	if errors.Is(err, costSvc.ErrInvalidYield) {
		return out, &badRequest{http.StatusUnprocessableEntity, fmt.Sprintf("línea %d: %s", n, err.Error())}
	}
	if err != nil {
		return out, err
	}
	price := int64(breakdown.SuggestedPrice)
	var unit, total int64
	if soldAs == "portion" {
		total = price
		unit = (total + int64(qty)/2) / int64(qty)
	} else {
		unit = price
		total = price * int64(qty)
	}
	if total > money.MaxPesos {
		return out, bad("el total de la línea %d supera el máximo permitido", n)
	}
	return quoteRepo.LineInput{
		RecipeID: l.RecipeID, RecipeName: name, PhotoURL: photo, SoldAs: soldAs, Quantity: qty,
		UnitPrice: unit, LineTotal: total, MarginPct: margin, IndirectCostPct: indirect, LaborCostPct: labor, ExtraCharge: extra,
	}, nil
}

// Create validates and prices every line, then saves the quotation whole. It answers JSON; the PDF
// is made from what was saved (GET /quotations/{id}/pdf).
func (h *QuotationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	req.ClientName = strings.TrimSpace(req.ClientName)
	switch {
	case req.ClientName == "":
		httputil.JSONError(w, "el nombre del cliente es requerido", http.StatusBadRequest)
		return
	case len([]rune(req.ClientName)) > 100:
		httputil.JSONError(w, "el nombre del cliente no puede superar 100 caracteres", http.StatusBadRequest)
		return
	case len(req.Lines) < 1 || len(req.Lines) > maxLines:
		httputil.JSONError(w, fmt.Sprintf("la cotización debe tener entre 1 y %d líneas", maxLines), http.StatusBadRequest)
		return
	case req.DeliveryCost < 0 || req.DeliveryCost > money.MaxPesos:
		httputil.JSONError(w, "el costo de delivery no es válido", http.StatusBadRequest)
		return
	}

	in := quoteRepo.SaveInput{CreatedBy: userID(r), ClientName: req.ClientName, DeliveryCost: req.DeliveryCost, CreatedAt: h.clk()}
	for i, l := range req.Lines {
		priced, err := h.priceLine(r, i+1, l)
		var br *badRequest
		if errors.As(err, &br) {
			httputil.JSONError(w, br.message, br.status)
			return
		}
		if err != nil {
			httputil.JSONError(w, "error calculando costos", http.StatusInternalServerError)
			return
		}
		in.Lines = append(in.Lines, priced)
	}
	q, err := h.repo.Save(r.Context(), in)
	if err != nil {
		httputil.JSONError(w, "error guardando la cotización", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q) //nolint:errcheck
}

func (h *QuotationHandler) Get(w http.ResponseWriter, r *http.Request) {
	q, err := h.repo.Get(r.Context(), chi.URLParam(r, "id"))
	if quoteRepo.IsNotFound(err) {
		httputil.JSONError(w, "cotización no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		httputil.JSONError(w, "error leyendo la cotización", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q) //nolint:errcheck
}

var asciiFold = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N")

var notFileSafe = regexp.MustCompile(`[^A-Za-z0-9]+`)

func safeFileName(client string) string {
	s := strings.Trim(notFileSafe.ReplaceAllString(asciiFold.Replace(client), "_"), "_")
	if s == "" {
		return "cliente"
	}
	return s
}

// PDF makes the quotation document from what was saved and the current business data.
func (h *QuotationHandler) PDF(w http.ResponseWriter, r *http.Request) {
	q, err := h.repo.Get(r.Context(), chi.URLParam(r, "id"))
	if quoteRepo.IsNotFound(err) {
		httputil.JSONError(w, "cotización no encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		httputil.JSONError(w, "error leyendo la cotización", http.StatusInternalServerError)
		return
	}
	profile, err := h.business.Get(r.Context())
	if err != nil {
		httputil.JSONError(w, "error leyendo los datos del negocio", http.StatusInternalServerError)
		return
	}
	if missing := profile.Missing(h.env); len(missing) > 0 {
		var fields, envs []string
		for _, m := range missing {
			if strings.HasPrefix(m, "BIZ_") {
				envs = append(envs, m)
			} else {
				fields = append(fields, m)
			}
		}
		var parts []string
		if len(fields) > 0 {
			parts = append(parts, "completa los Datos del negocio en la app (falta: "+strings.Join(fields, ", ")+")")
		}
		if len(envs) > 0 {
			parts = append(parts, "configura en Vercel las variables "+strings.Join(envs, ", "))
		}
		httputil.JSONError(w, "no se puede generar el PDF: "+strings.Join(parts, " y "), http.StatusServiceUnavailable)
		return
	}

	data := quoteSvc.QuoteData{
		ClientName: q.ClientName, DeliveryCost: q.DeliveryCost, Total: q.Total,
		Date: q.CreatedAt.In(clock.Santiago),
		Business: quoteSvc.Business{
			Name: profile.Name, Phone: profile.Phone, Instagram: profile.Instagram, Bank: profile.Bank,
			AccountType: profile.AccountType, Holder: profile.Holder, Email: profile.Email,
			Account: h.env.Account, RUT: h.env.RUT,
		},
	}
	for _, l := range q.Lines {
		data.Lines = append(data.Lines, quoteSvc.QuoteLine{
			Name: l.RecipeName, PhotoURL: l.PhotoURL, Quantity: l.Quantity, SoldAs: l.SoldAs, UnitPrice: l.UnitPrice, LineTotal: l.LineTotal,
		})
	}
	buf, err := quoteSvc.GenerateQuotePDF(data)
	if err != nil {
		httputil.JSONError(w, "error generando PDF", http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("cotizacion_%s_%s.pdf", safeFileName(q.ClientName), q.CreatedAt.In(clock.Santiago).Format(time.DateOnly))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes()) //nolint:errcheck
}

func (h *QuotationHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	validStatuses := map[string]bool{"pending": true, "confirmed": true, "cancelled": true, "": true}
	if !validStatuses[status] {
		httputil.JSONError(w, "estado inválido", http.StatusBadRequest)
		return
	}
	quotations, err := h.repo.List(r.Context(), status)
	if err != nil {
		httputil.JSONError(w, "error listando cotizaciones", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotations) //nolint:errcheck
}

func (h *QuotationHandler) setStatus(w http.ResponseWriter, r *http.Request, status, failure string) {
	if err := h.repo.UpdateStatus(r.Context(), chi.URLParam(r, "id"), status); err != nil {
		if quoteRepo.IsNotFound(err) {
			httputil.JSONError(w, "cotización no encontrada", http.StatusNotFound)
			return
		}
		httputil.JSONError(w, failure, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *QuotationHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "confirmed", "error confirmando cotización")
}

func (h *QuotationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "cancelled", "error cancelando cotización")
}
