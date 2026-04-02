package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PipeM113/bakery-manager/internal/ingredients/repository"
	"github.com/PipeM113/bakery-manager/internal/ingredients/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

var allowedUnits = map[string]bool{
	"gr": true, "kg": true, "ml": true, "lt": true, "und": true,
}

type IngredientHandler struct {
	repo      *repository.IngredientRepository
	importSvc *service.IngredientImportService
}

func NewIngredientHandler(repo *repository.IngredientRepository, importSvc *service.IngredientImportService) *IngredientHandler {
	return &IngredientHandler{repo: repo, importSvc: importSvc}
}

func userID(r *http.Request) string {
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	return claims.ID
}

func (h *IngredientHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	ingredients, err := h.repo.GetAll(r.Context())
	if err != nil {
		httputil.JSONError(w, "error obteniendo insumos", http.StatusInternalServerError)
		return
	}
	if ingredients == nil {
		ingredients = []repository.Ingredient{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ingredients)
}

func (h *IngredientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body repository.Ingredient
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	body.Brand = strings.TrimSpace(body.Brand)
	if body.Name == "" {
		httputil.JSONError(w, "el nombre es requerido", http.StatusBadRequest)
		return
	}
	if body.Brand == "" {
		httputil.JSONError(w, "la marca es requerida", http.StatusBadRequest)
		return
	}
	if !allowedUnits[body.DefaultUnit] {
		httputil.JSONError(w, "unidad inválida, use: gr, kg, ml, lt, und", http.StatusBadRequest)
		return
	}
	if body.PackagePrice <= 0 {
		httputil.JSONError(w, "el precio del paquete debe ser mayor a 0", http.StatusBadRequest)
		return
	}
	if body.PackageSize <= 0 {
		httputil.JSONError(w, "el tamaño del paquete debe ser mayor a 0", http.StatusBadRequest)
		return
	}

	log.Printf("[ingredients] POST /ingredients user=%s name=%q brand=%q unit=%s pkg_price=%.2f",
		userID(r), body.Name, body.Brand, body.DefaultUnit, body.PackagePrice)

	created, err := h.repo.Create(r.Context(), body)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			httputil.JSONError(w, "ya existe un insumo con ese nombre", http.StatusConflict)
			return
		}
		log.Printf("[ingredients] Create error: %v", err)
		httputil.JSONError(w, "error creando insumo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *IngredientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body repository.Ingredient
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}

	body.ID = id
	body.Name = strings.TrimSpace(body.Name)
	body.Brand = strings.TrimSpace(body.Brand)
	if body.Name == "" {
		httputil.JSONError(w, "el nombre es requerido", http.StatusBadRequest)
		return
	}
	if body.Brand == "" {
		httputil.JSONError(w, "la marca es requerida", http.StatusBadRequest)
		return
	}
	if !allowedUnits[body.DefaultUnit] {
		httputil.JSONError(w, "unidad inválida, use: gr, kg, ml, lt, und", http.StatusBadRequest)
		return
	}
	if body.PackagePrice <= 0 {
		httputil.JSONError(w, "el precio del paquete debe ser mayor a 0", http.StatusBadRequest)
		return
	}
	if body.PackageSize <= 0 {
		httputil.JSONError(w, "el tamaño del paquete debe ser mayor a 0", http.StatusBadRequest)
		return
	}

	log.Printf("[ingredients] PUT /ingredients/%s user=%s name=%q", id, userID(r), body.Name)

	updated, err := h.repo.Update(r.Context(), body)
	if err != nil {
		log.Printf("[ingredients] Update error id=%s: %v", id, err)
		httputil.JSONError(w, "error actualizando insumo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *IngredientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Printf("[ingredients] DELETE /ingredients/%s user=%s", id, userID(r))
	if err := h.repo.Delete(r.Context(), id); err != nil {
		log.Printf("[ingredients] Delete error id=%s: %v", id, err)
		httputil.JSONError(w, "error eliminando insumo", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IngredientHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	history, err := h.repo.GetPriceHistory(r.Context(), id)
	if err != nil {
		httputil.JSONError(w, "error obteniendo historial", http.StatusInternalServerError)
		return
	}
	if history == nil {
		history = []repository.PriceHistory{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *IngredientHandler) Export(w http.ResponseWriter, r *http.Request) {
	ingredients, err := h.repo.GetAll(r.Context())
	if err != nil {
		httputil.JSONError(w, "error obteniendo insumos", http.StatusInternalServerError)
		return
	}
	if ingredients == nil {
		ingredients = []repository.Ingredient{}
	}

	buf, err := service.ExportIngredientsToExcel(ingredients)
	if err != nil {
		httputil.JSONError(w, "error generando Excel", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("insumos_%s.xlsx", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(buf.Bytes())
}

func (h *IngredientHandler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httputil.JSONError(w, "archivo demasiado grande (máx 10MB)", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httputil.JSONError(w, "campo 'file' requerido", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "import-*.xlsx")
	if err != nil {
		httputil.JSONError(w, "error interno", http.StatusInternalServerError)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		httputil.JSONError(w, "error leyendo archivo", http.StatusInternalServerError)
		return
	}
	tmp.Close()

	rows, skipped, err := service.ParseIngredientsFromExcel(tmpName)
	if err != nil {
		httputil.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.importSvc.Import(r.Context(), rows, skipped)
	if err != nil {
		httputil.JSONError(w, "error importando insumos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
