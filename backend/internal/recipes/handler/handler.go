package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PipeM113/bakery-manager/internal/recipes/repository"
	"github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

type RecipeHandler struct {
	repo *repository.RecipeRepository
}

func NewRecipeHandler(repo *repository.RecipeRepository) *RecipeHandler {
	return &RecipeHandler{repo: repo}
}

func (h *RecipeHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.repo.GetAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"error obteniendo recetas"}`, http.StatusInternalServerError)
		return
	}
	if recipes == nil {
		recipes = []repository.Recipe{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipes)
}

func (h *RecipeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	recipe, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"receta no encontrada"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (h *RecipeHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserKey).(middleware.UserClaims)
	if !ok {
		http.Error(w, `{"error":"no autorizado"}`, http.StatusUnauthorized)
		return
	}

	var body repository.Recipe
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"request inválido"}`, http.StatusBadRequest)
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, `{"error":"el nombre es requerido"}`, http.StatusBadRequest)
		return
	}
	if body.Yield <= 0 {
		http.Error(w, `{"error":"el rendimiento debe ser mayor a 0"}`, http.StatusBadRequest)
		return
	}
	if len(body.Ingredients) == 0 {
		http.Error(w, `{"error":"la receta debe tener al menos un ingrediente"}`, http.StatusBadRequest)
		return
	}

	body.UserID = claims.ID
	if body.YieldUnit == "" {
		body.YieldUnit = "porciones"
	}
	if body.IndirectCostPct == 0 {
		body.IndirectCostPct = 0.15
	}
	if body.LaborCostPct == 0 {
		body.LaborCostPct = 0.30
	}

	if body.ParentID == nil {
		body.IsBase = true
	}

	created, err := h.repo.Create(r.Context(), body)

	if err != nil {
		http.Error(w, `{"error":"error creando receta"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *RecipeHandler) CreateVersion(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserKey).(middleware.UserClaims)
	if !ok {
		http.Error(w, `{"error":"no autorizado"}`, http.StatusUnauthorized)
		return
	}

	parentID := chi.URLParam(r, "id")
	_, err := h.repo.GetByID(r.Context(), parentID)
	if err != nil {
		http.Error(w, `{"error":"receta base no encontrada"}`, http.StatusNotFound)
		return
	}

	var body repository.Recipe
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"request inválido"}`, http.StatusBadRequest)
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, `{"error":"el nombre es requerido"}`, http.StatusBadRequest)
		return
	}

	body.UserID = claims.ID
	body.ParentID = &parentID
	body.IsBase = false

	created, err := h.repo.Create(r.Context(), body)
	if err != nil {
		http.Error(w, `{"error":"error creando versión"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *RecipeHandler) Scale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	recipe, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"receta no encontrada"}`, http.StatusNotFound)
		return
	}

	type scaleRequest struct {
		Portions float64 `json:"portions"`
	}
	var req scaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"request inválido"}`, http.StatusBadRequest)
		return
	}
	if req.Portions <= 0 {
		http.Error(w, `{"error":"porciones debe ser mayor a 0"}`, http.StatusBadRequest)
		return
	}

	factor := req.Portions / recipe.Yield
	for i := range recipe.Ingredients {
		recipe.Ingredients[i].Quantity = recipe.Ingredients[i].Quantity * factor
	}
	recipe.Yield = req.Portions

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (h *RecipeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"error eliminando receta"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
