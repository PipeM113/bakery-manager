package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// JSONError writes a JSON error response with the correct Content-Type.
// Use this instead of http.Error to ensure the frontend can parse error messages.
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// The lists that grow with the years are read in pages: ?limit= (1 to MaxPageSize, DefaultPageSize when absent)
// and ?offset= (0 or more). Without them the answer is the first DefaultPageSize rows, never the whole table.
const (
	DefaultPageSize = 200
	MaxPageSize     = 1000
)

// Page reads limit and offset from the query. The error text is meant for the person.
func Page(r *http.Request) (limit, offset int, err error) {
	limit, offset = DefaultPageSize, 0
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 1 || n > MaxPageSize {
			return 0, 0, fmt.Errorf("limit debe ser un entero entre 1 y %d", MaxPageSize)
		}
		limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 0 {
			return 0, 0, fmt.Errorf("offset debe ser un entero desde 0")
		}
		offset = n
	}
	return limit, offset, nil
}

// SetHasMore says in the X-Has-More header whether rows exist beyond this page.
func SetHasMore(w http.ResponseWriter, more bool) {
	w.Header().Set("X-Has-More", strconv.FormatBool(more))
}
