package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/PipeM113/bakery-manager/internal/auth/service"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	mid "github.com/PipeM113/bakery-manager/pkg/middleware"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// tooManyAttempts answers 429 with how long to wait, in the header and in the text.
func tooManyAttempts(w http.ResponseWriter, locked *service.LockedError) {
	secs := int(math.Ceil(locked.RetryAfter.Seconds()))
	minutes := (secs + 59) / 60
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	unit := "minutos"
	if minutes == 1 {
		unit = "minuto"
	}
	httputil.JSONError(w, "Demasiados intentos fallidos. Vuelve a intentar en "+strconv.Itoa(minutes)+" "+unit+".", http.StatusTooManyRequests)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}

	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	var locked *service.LockedError
	switch {
	case errors.As(err, &locked):
		tooManyAttempts(w, locked)
		return
	case errors.Is(err, service.ErrInvalidCredentials):
		httputil.JSONError(w, "credenciales inválidas", http.StatusUnauthorized)
		return
	case err != nil:
		httputil.JSONError(w, "error iniciando sesión", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{Token: token})
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword changes the password of the person who is logged in.
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(mid.UserKey).(mid.UserClaims)
	if !ok {
		httputil.JSONError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	var req passwordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	err := h.svc.ChangePassword(r.Context(), claims.ID, req.CurrentPassword, req.NewPassword)
	var locked *service.LockedError
	var invalid *service.ValidationError
	switch {
	case errors.As(err, &locked):
		tooManyAttempts(w, locked)
	case errors.As(err, &invalid):
		httputil.JSONError(w, invalid.Message, http.StatusBadRequest)
	case errors.Is(err, service.ErrWrongCurrentPassword):
		httputil.JSONError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrUnknownUser):
		httputil.JSONError(w, "no autorizado", http.StatusUnauthorized)
	case err != nil:
		httputil.JSONError(w, "error cambiando la contraseña", http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
