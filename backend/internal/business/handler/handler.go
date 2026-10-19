package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/PipeM113/bakery-manager/internal/business/repository"
	"github.com/PipeM113/bakery-manager/internal/config"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
)

type BusinessHandler struct {
	repo *repository.Repository
	env  config.Business
}

func NewBusinessHandler(repo *repository.Repository, env config.Business) *BusinessHandler {
	return &BusinessHandler{repo: repo, env: env}
}

// response is the profile plus what is still missing to make a PDF. The RUT and the account
// number are never sent: only the name of the variable that is not set.
type response struct {
	repository.Profile
	Missing []string `json:"missing"`
}

func (h *BusinessHandler) write(w http.ResponseWriter, p repository.Profile) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{Profile: p, Missing: p.Missing(h.env)}) //nolint:errcheck
}

func (h *BusinessHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, err := h.repo.Get(r.Context())
	if err != nil {
		httputil.JSONError(w, "error leyendo los datos del negocio", http.StatusInternalServerError)
		return
	}
	h.write(w, p)
}

func (h *BusinessHandler) Save(w http.ResponseWriter, r *http.Request) {
	var in repository.Profile
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	claims, _ := r.Context().Value(mid.UserKey).(mid.UserClaims)
	p, err := h.repo.Save(r.Context(), in, claims.ID)
	var invalid *repository.ValidationError
	if errors.As(err, &invalid) {
		httputil.JSONError(w, invalid.Message, http.StatusBadRequest)
		return
	}
	if err != nil {
		httputil.JSONError(w, "error guardando los datos del negocio", http.StatusInternalServerError)
		return
	}
	h.write(w, p)
}
