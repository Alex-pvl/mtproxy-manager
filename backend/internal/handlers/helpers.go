package handlers

import (
	"encoding/json"
	"net/http"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/middleware"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func getClaims(r *http.Request) *auth.Claims {
	return middleware.GetClaims(r)
}
