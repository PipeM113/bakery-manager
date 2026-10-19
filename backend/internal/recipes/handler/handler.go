package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	costs "github.com/PipeM113/bakery-manager/internal/costs/service"
	"github.com/PipeM113/bakery-manager/internal/recipes/repository"
	"github.com/PipeM113/bakery-manager/pkg/httputil"
	"github.com/PipeM113/bakery-manager/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

// PhotoUploader stores a recipe photo and returns its public URL.
type PhotoUploader interface {
	UploadRecipePhoto(ctx context.Context, recipeID, contentType string, data []byte) (string, error)
}

// recipeBody is a recipe as the client sends it. The three percentages are pointers so that
// "not sent" can be told apart from a real 0%.
type recipeBody struct {
	repository.Recipe
	IndirectCostPct *float64 `json:"indirect_cost_pct"`
	LaborCostPct    *float64 `json:"labor_cost_pct"`
	MarginPct       *float64 `json:"margin_pct"`
}

func (b recipeBody) validatePcts() error {
	return costs.CostParams{
		IndirectCostPct: b.IndirectCostPct, LaborCostPct: b.LaborCostPct, MarginPct: b.MarginPct,
	}.Validate()
}

type RecipeHandler struct {
	repo   *repository.RecipeRepository
	photos PhotoUploader
}

func NewRecipeHandler(repo *repository.RecipeRepository, photos PhotoUploader) *RecipeHandler {
	return &RecipeHandler{repo: repo, photos: photos}
}

func (h *RecipeHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.repo.GetAll(r.Context())
	if err != nil {
		httputil.JSONError(w, "error obteniendo recetas", http.StatusInternalServerError)
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
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recipe)
}

func (h *RecipeHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserKey).(middleware.UserClaims)
	if !ok {
		httputil.JSONError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	var in recipeBody
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if err := in.validatePcts(); err != nil {
		httputil.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	body := in.Recipe

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		httputil.JSONError(w, "el nombre es requerido", http.StatusBadRequest)
		return
	}
	body.Description = strings.TrimSpace(body.Description)
	if body.Description == "" {
		httputil.JSONError(w, "la descripción es requerida", http.StatusBadRequest)
		return
	}
	if body.Yield <= 0 {
		httputil.JSONError(w, "el rendimiento debe ser mayor a 0", http.StatusBadRequest)
		return
	}
	if len(body.Ingredients) == 0 {
		httputil.JSONError(w, "la receta debe tener al menos un ingrediente", http.StatusBadRequest)
		return
	}
	for _, ing := range body.Ingredients {
		if ing.Quantity <= 0 {
			httputil.JSONError(w, "la cantidad de cada ingrediente debe ser mayor a 0", http.StatusBadRequest)
			return
		}
	}

	body.CreatedBy = &claims.ID
	if body.YieldUnit == "" {
		body.YieldUnit = "porciones"
	}
	// A new recipe starts with the same base in the three percentages; an explicit 0 is kept.
	body.IndirectCostPct, body.LaborCostPct, body.MarginPct = costs.DefaultPct, costs.DefaultPct, costs.DefaultPct
	if in.IndirectCostPct != nil {
		body.IndirectCostPct = *in.IndirectCostPct
	}
	if in.LaborCostPct != nil {
		body.LaborCostPct = *in.LaborCostPct
	}
	if in.MarginPct != nil {
		body.MarginPct = *in.MarginPct
	}
	if body.ParentID == nil {
		body.IsBase = true
	}

	log.Printf("[recipes] POST /recipes user=%s name=%q ingredients=%d", claims.ID, body.Name, len(body.Ingredients))

	created, err := h.repo.Create(r.Context(), body)
	if err != nil {
		log.Printf("[recipes] Create error user=%s name=%q: %v", claims.ID, body.Name, err)
		httputil.JSONError(w, "error creando receta", http.StatusInternalServerError)
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
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
		return
	}

	type scaleRequest struct {
		Portions float64 `json:"portions"`
	}
	var req scaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if req.Portions <= 0 {
		httputil.JSONError(w, "porciones debe ser mayor a 0", http.StatusBadRequest)
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

func (h *RecipeHandler) SaveAs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Portions float64 `json:"portions"`
		NewName  string  `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if req.Portions < 1 {
		httputil.JSONError(w, "porciones debe ser al menos 1", http.StatusBadRequest)
		return
	}
	req.NewName = strings.TrimSpace(req.NewName)
	if req.NewName == "" {
		httputil.JSONError(w, "el nombre es requerido", http.StatusBadRequest)
		return
	}

	original, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
		return
	}
	if original.Yield <= 0 {
		httputil.JSONError(w, "la receta original no tiene rendimiento válido", http.StatusBadRequest)
		return
	}

	scaleFactor := req.Portions / original.Yield
	recipe, err := h.repo.SaveScaled(r.Context(), id, scaleFactor, req.NewName)
	if err != nil {
		httputil.JSONError(w, "error guardando receta", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func (h *RecipeHandler) SaveScaled(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	type saveScaledRequest struct {
		ScaleFactor float64 `json:"scale_factor"`
		NewName     string  `json:"new_name"`
	}
	var req saveScaledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if req.ScaleFactor <= 0 {
		httputil.JSONError(w, "scale_factor debe ser mayor a 0", http.StatusBadRequest)
		return
	}
	req.NewName = strings.TrimSpace(req.NewName)
	if req.NewName == "" {
		httputil.JSONError(w, "el nombre es requerido", http.StatusBadRequest)
		return
	}

	recipe, err := h.repo.SaveScaled(r.Context(), id, req.ScaleFactor, req.NewName)
	if err != nil {
		httputil.JSONError(w, "error guardando receta escalada", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(recipe)
}

func (h *RecipeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := r.Context().Value(middleware.UserKey).(middleware.UserClaims)
	if !ok {
		httputil.JSONError(w, "no autorizado", http.StatusUnauthorized)
		return
	}
	log.Printf("[recipes] DELETE /recipes/%s user=%s", id, claims.ID)
	if err := h.repo.Delete(r.Context(), id); err != nil {
		log.Printf("[recipes] Delete error id=%s user=%s: %v", id, claims.ID, err)
		httputil.JSONError(w, "error eliminando receta", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RecipeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := r.Context().Value(middleware.UserKey).(middleware.UserClaims)
	if !ok {
		httputil.JSONError(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	var in recipeBody
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httputil.JSONError(w, "request inválido", http.StatusBadRequest)
		return
	}
	if err := in.validatePcts(); err != nil {
		httputil.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	body := in.Recipe
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		httputil.JSONError(w, "el nombre es requerido", http.StatusBadRequest)
		return
	}
	if body.Yield <= 0 {
		httputil.JSONError(w, "el rendimiento debe ser mayor a 0", http.StatusBadRequest)
		return
	}
	if len(body.Ingredients) == 0 {
		httputil.JSONError(w, "la receta debe tener al menos un ingrediente", http.StatusBadRequest)
		return
	}
	for _, ing := range body.Ingredients {
		if ing.Quantity <= 0 {
			httputil.JSONError(w, "la cantidad de cada ingrediente debe ser mayor a 0", http.StatusBadRequest)
			return
		}
	}

	// Percentages that were not sent keep the values the recipe already has.
	if in.IndirectCostPct == nil || in.LaborCostPct == nil {
		current, err := h.repo.GetByID(r.Context(), id)
		if err != nil {
			httputil.JSONError(w, "receta no encontrada", http.StatusNotFound)
			return
		}
		body.IndirectCostPct, body.LaborCostPct = current.IndirectCostPct, current.LaborCostPct
	}
	if in.IndirectCostPct != nil {
		body.IndirectCostPct = *in.IndirectCostPct
	}
	if in.LaborCostPct != nil {
		body.LaborCostPct = *in.LaborCostPct
	}

	log.Printf("[recipes] PUT /recipes/%s user=%s name=%q", id, claims.ID, body.Name)

	updated, err := h.repo.Update(r.Context(), id, body)
	if err != nil {
		log.Printf("[recipes] Update error id=%s user=%s: %v", id, claims.ID, err)
		httputil.JSONError(w, "error actualizando receta", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// maxPhotoBytes is the largest photo accepted. A Vercel function takes request bodies of up to
// 4.5 MB, so the photo plus the multipart framing must stay under that.
const maxPhotoBytes = 4 << 20

// multipartOverhead is what the request may carry besides the photo (boundaries and headers).
const multipartOverhead = 64 << 10

var recipeIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var photoTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

func (h *RecipeHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	if h.photos == nil {
		httputil.JSONError(w, "servicio de imágenes no configurado", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if !recipeIDPattern.MatchString(id) {
		httputil.JSONError(w, "id de receta inválido", http.StatusBadRequest)
		return
	}

	data, status, msg := readPhotoPart(w, r)
	if msg != "" {
		httputil.JSONError(w, msg, status)
		return
	}

	// The type is decided by the content: the name and the declared type are the client's word.
	contentType := http.DetectContentType(data)
	if !photoTypes[contentType] {
		httputil.JSONError(w, "tipo de archivo no permitido (solo JPEG, PNG, WebP)", http.StatusBadRequest)
		return
	}

	photoURL, err := h.photos.UploadRecipePhoto(r.Context(), id, contentType, data)
	if err != nil {
		log.Printf("[recipes] photo upload id=%s: %v", id, err)
		httputil.JSONError(w, "error subiendo foto", http.StatusBadGateway)
		return
	}

	if err := h.repo.UpdatePhotoURL(r.Context(), id, photoURL); err != nil {
		httputil.JSONError(w, "error guardando URL de foto", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"photo_url": photoURL})
}

// readPhotoPart returns the bytes of the "photo" field without ever holding more than the limit
// plus one byte. On failure it returns the HTTP status and a message.
func readPhotoPart(w http.ResponseWriter, r *http.Request) ([]byte, int, string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoBytes+multipartOverhead)
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, http.StatusBadRequest, "solicitud multipart inválida"
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, http.StatusBadRequest, "no se recibió archivo con nombre 'photo'"
		}
		if err != nil {
			return nil, statusForBodyError(err), messageForBodyError(err)
		}
		if part.FormName() != "photo" {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(part, maxPhotoBytes+1))
		if err != nil {
			return nil, statusForBodyError(err), messageForBodyError(err)
		}
		if len(data) > maxPhotoBytes {
			return nil, http.StatusRequestEntityTooLarge, "archivo demasiado grande (máx 4MB)"
		}
		return data, 0, ""
	}
}

func statusForBodyError(err error) int {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

func messageForBodyError(err error) string {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return "archivo demasiado grande (máx 4MB)"
	}
	return "solicitud multipart inválida"
}
