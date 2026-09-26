package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// httpClient is shared by all outbound calls (payment providers, Telegram).
var httpClient = &http.Client{Timeout: 30 * time.Second}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func readJSON(r *http.Request, out any) error {
	return json.NewDecoder(r.Body).Decode(out)
}

// getClaims is only valid behind middleware.AuthRequired, which guarantees non-nil.
func getClaims(r *http.Request) *auth.Claims {
	return middleware.GetClaims(r)
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
