package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/PipeM113/bakery-manager/internal/recipe_categories/repository"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/go-chi/chi/v5"
)

const maxNameLength = 50

type Handler struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Handler { return &Handler{repo: repo} }

// validName trims the name, collapses the spaces inside it and checks its length.
func validName(raw string) (string, string) {
	name := strings.Join(strings.Fields(raw), " ")
	switch {
	case name == "":
		return "", "el nombre es requerido"
	case utf8.RuneCountInString(name) > maxNameLength:
		return "", "el nombre no puede superar 50 caracteres"
	}
	return name, ""
}

func (h *Handler) fail(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		httputil.JSONError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, repository.ErrDuplicate):
		httputil.JSONError(w, err.Error(), http.StatusConflict)
	default:
		log.Printf("[recipe-categories] %s: %v", fallback, err)
		httputil.JSONError(w, fallback, http.StatusInternalServerError)
	}
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.List(r.Context())
	if err != nil {
		h.fail(w, err, "error obteniendo los tipos")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	name, ok := decodeName(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Create(r.Context(), name)
	if err != nil {
		h.fail(w, err, "error creando el tipo")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	name, ok := decodeName(w, r)
	if !ok {
		return
	}
	c, err := h.repo.Rename(r.Context(), chi.URLParam(r, "id"), name)
	if err != nil {
		h.fail(w, err, "error renombrando el tipo")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		h.fail(w, err, "error eliminando el tipo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
