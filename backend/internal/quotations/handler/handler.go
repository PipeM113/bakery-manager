package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	costSvc "github.com/PipeM113/bakery-manager/internal/costs/service"
	quoteRepo "github.com/PipeM113/bakery-manager/internal/quotations/repository"
	quoteSvc "github.com/PipeM113/bakery-manager/internal/quotations/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuotationHandler struct {
	costSvc *costSvc.CostService
	repo    *quoteRepo.QuotationRepository
	db      *pgxpool.Pool
}

func NewQuotationHandler(cs *costSvc.CostService, db *pgxpool.Pool) *QuotationHandler {
	return &QuotationHandler{
		costSvc: cs,
		repo:    quoteRepo.NewQuotationRepository(db),
		db:      db,
	}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

type generateRequest struct {
	RecipeID        string  `json:"recipe_id"`
	ClientName      string  `json:"client_name"`
	MarginPct       float64 `json:"margin_pct"`
	IndirectCostPct float64 `json:"indirect_cost_pct"`
	LaborCostPct    float64 `json:"labor_cost_pct"`
	ExtraCharge     int     `json:"extra_charge"`
	DeliveryCost    int     `json:"delivery_cost"`
}

func (h *QuotationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RecipeID == "" {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	req.ClientName = strings.TrimSpace(req.ClientName)
	if req.ClientName == "" {
		httputil.JSONError(w, "el nombre del cliente es requerido", http.StatusBadRequest)
		return
	}
	if len(req.ClientName) > 100 {
		httputil.JSONError(w, "el nombre del cliente no puede superar 100 caracteres", http.StatusBadRequest)
		return
	}
	if req.MarginPct < 0 {
		httputil.JSONError(w, "el margen no puede ser negativo", http.StatusBadRequest)
		return
	}

	var recipe struct {
		Name        string
		Description string
		Yield       float64
		YieldUnit   string
		PhotoURL    *string
	}
	err := h.db.QueryRow(r.Context(), `
		SELECT name, description, yield, yield_unit, photo_url
		FROM recipes WHERE id = $1`, req.RecipeID,
	).Scan(&recipe.Name, &recipe.Description, &recipe.Yield, &recipe.YieldUnit, &recipe.PhotoURL)
	if err != nil {
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
		return
	}
	if strings.TrimSpace(recipe.Description) == "" {
		httputil.JSONError(w, "la receta debe tener descripción para generar cotización", http.StatusBadRequest)
		return
	}

	breakdown, err := h.costSvc.GetCostBreakdown(r.Context(), req.RecipeID, costSvc.CostParams{
		IndirectCostPct: req.IndirectCostPct,
		LaborCostPct:    req.LaborCostPct,
		MarginPct:       req.MarginPct,
		ExtraCharge:     req.ExtraCharge,
		DeliveryCost:    req.DeliveryCost,
	})
	if err != nil {
		httputil.JSONError(w, "error calculando costos", http.StatusInternalServerError)
		return
	}

	// SuggestedPrice already includes ceilTo500 + extra_charge + delivery_cost
	finalPrice := breakdown.SuggestedPrice

	buf, err := quoteSvc.GenerateQuotePDF(quoteSvc.QuoteData{
		RecipeName:   recipe.Name,
		ClientName:   req.ClientName,
		Description:  recipe.Description,
		Yield:        recipe.Yield,
		YieldUnit:    recipe.YieldUnit,
		PhotoURL:     recipe.PhotoURL,
		BasePrice:    breakdown.BasePrice,
		DeliveryCost: breakdown.DeliveryCost,
		FinalPrice:   finalPrice,
		Date:         time.Now(),
	})
	if err != nil {
		httputil.JSONError(w, "error generando PDF", http.StatusInternalServerError)
		return
	}

	// Persist quotation record as pending
	uid := userID(r)
	if uid != "" {
		_, _ = h.repo.Save(r.Context(), quoteRepo.SaveInput{
			UserID:     uid,
			RecipeID:   req.RecipeID,
			ClientName: req.ClientName,
			MarginPct:  req.MarginPct,
			FinalPrice: finalPrice,
		})
	}

	safeName := strings.NewReplacer(" ", "_", "/", "-", "\\", "-").Replace(recipe.Name)
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("cotizacion_%s_%s.pdf", safeName, date)

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

	quotations, err := h.repo.List(r.Context(), userID(r), status)
	if err != nil {
		httputil.JSONError(w, "error listando cotizaciones", http.StatusInternalServerError)
		return
	}
	if quotations == nil {
		quotations = []quoteRepo.Quotation{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotations) //nolint:errcheck
}

func (h *QuotationHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.UpdateStatus(r.Context(), id, userID(r), "confirmed"); err != nil {
		if quoteRepo.IsNotFound(err) {
			httputil.JSONError(w, "cotización no encontrada", http.StatusNotFound)
			return
		}
		httputil.JSONError(w, "error confirmando cotización", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *QuotationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.UpdateStatus(r.Context(), id, userID(r), "cancelled"); err != nil {
		if quoteRepo.IsNotFound(err) {
			httputil.JSONError(w, "cotización no encontrada", http.StatusNotFound)
			return
		}
		httputil.JSONError(w, "error cancelando cotización", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
