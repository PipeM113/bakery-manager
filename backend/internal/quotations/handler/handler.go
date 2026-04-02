package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	costSvc "github.com/PipeM113/bakery-manager/internal/costs/service"
	quoteSvc "github.com/PipeM113/bakery-manager/internal/quotations/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuotationHandler struct {
	costSvc *costSvc.CostService
	db      *pgxpool.Pool
}

func NewQuotationHandler(cs *costSvc.CostService, db *pgxpool.Pool) *QuotationHandler {
	return &QuotationHandler{costSvc: cs, db: db}
}

type generateRequest struct {
	RecipeID        string  `json:"recipe_id"`
	MarginPct       float64 `json:"margin_pct"`
	IndirectCostPct float64 `json:"indirect_cost_pct"`
	LaborCostPct    float64 `json:"labor_cost_pct"`
}

func (h *QuotationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RecipeID == "" {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
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
	})
	if err != nil {
		httputil.JSONError(w, "error calculando costos", http.StatusInternalServerError)
		return
	}

	finalPrice := math.Ceil(breakdown.TotalCost*(1+req.MarginPct)/500) * 500

	buf, err := quoteSvc.GenerateQuotePDF(quoteSvc.QuoteData{
		RecipeName:  recipe.Name,
		Description: recipe.Description,
		Yield:       recipe.Yield,
		YieldUnit:   recipe.YieldUnit,
		PhotoURL:    recipe.PhotoURL,
		TotalCost:   breakdown.TotalCost,
		MarginPct:   req.MarginPct,
		FinalPrice:  finalPrice,
		Date:        time.Now(),
	})
	if err != nil {
		httputil.JSONError(w, "error generando PDF", http.StatusInternalServerError)
		return
	}

	safeName := strings.NewReplacer(" ", "_", "/", "-", "\\", "-").Replace(recipe.Name)
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("cotizacion_%s_%s.pdf", safeName, date)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes()) //nolint:errcheck
}
