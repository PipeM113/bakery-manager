package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PipeM113/bakery-manager/internal/ingredients/repository"
	"github.com/go-chi/chi/v5"
)

type IngredientHandler struct {
	repo *repository.IngredientRepository
}

func NewIngredientHandler(repo *repository.IngredientRepository) *IngredientHandler {
	return &IngredientHandler{repo: repo}
}

func (h *IngredientHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	ingredients, err := h.repo.GetAll(r.Context())
	if err != nil {
		http.Error(w, `{"error":"error obteniendo insumos"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"request inválido"}`, http.StatusBadRequest)
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || body.DefaultUnit == "" || body.PricePerUnit <= 0 {
		http.Error(w, `{"error":"nombre, unidad y precio son requeridos"}`, http.StatusBadRequest)
		return
	}

	created, err := h.repo.Create(r.Context(), body)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			http.Error(w, `{"error":"ya existe un insumo con ese nombre"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"error creando insumo"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"request inválido"}`, http.StatusBadRequest)
		return
	}

	body.ID = id
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || body.DefaultUnit == "" || body.PricePerUnit <= 0 {
		http.Error(w, `{"error":"nombre, unidad y precio son requeridos"}`, http.StatusBadRequest)
		return
	}

	updated, err := h.repo.Update(r.Context(), body)
	if err != nil {
		http.Error(w, `{"error":"error actualizando insumo"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *IngredientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"error eliminando insumo"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IngredientHandler) GetPriceHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	history, err := h.repo.GetPriceHistory(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"error obteniendo historial"}`, http.StatusInternalServerError)
		return
	}
	if history == nil {
		history = []repository.PriceHistory{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}
