package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/PipeM113/bakery-manager/internal/expense_categories/repository"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

const maxNameLength = 50

type CategoryHandler struct {
	repo *repository.CategoryRepository
}

func NewCategoryHandler(repo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

// validName trims the name and checks it. The cost of ingredients is taken from the sales,
// so a category with that name would count it twice.
func validName(raw string) (string, string) {
	name := strings.TrimSpace(raw)
	switch {
	case name == "":
		return "", "el nombre es requerido"
	case utf8.RuneCountInString(name) > maxNameLength:
		return "", "el nombre no puede superar 50 caracteres"
	case strings.EqualFold(name, "ingredientes"):
		return "", "el costo de los ingredientes ya sale de las ventas: no se registra como gasto"
	}
	return name, ""
}

func (h *CategoryHandler) fail(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		httputil.JSONError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, repository.ErrDuplicate):
		httputil.JSONError(w, err.Error(), http.StatusConflict)
	case errors.Is(err, repository.ErrInUse):
		httputil.JSONError(w, "no se puede borrar: la categoría tiene gastos", http.StatusConflict)
	default:
		httputil.JSONError(w, fallback, http.StatusInternalServerError)
	}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.List(r.Context())
	if err != nil {
		h.fail(w, err, "error obteniendo categorías")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func decodeName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return "", false
	}
	name, msg := validName(body.Name)
	if msg != "" {
		httputil.JSONError(w, msg, http.StatusBadRequest)
		return "", false
	}
	return name, true
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	name, ok := decodeName(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Create(r.Context(), name)
	if err != nil {
		h.fail(w, err, "error creando categoría")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func (h *CategoryHandler) Rename(w http.ResponseWriter, r *http.Request) {
	name, ok := decodeName(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Rename(r.Context(), chi.URLParam(r, "id"), name)
	if err != nil {
		h.fail(w, err, "error renombrando categoría")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		h.fail(w, err, "error eliminando categoría")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
