package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/docker"
	"mtproxy-manager/internal/models"
)

type AdminHandler struct {
	db     *database.DB
	docker *docker.Manager
}

func NewAdminHandler(db *database.DB, docker *docker.Manager) *AdminHandler {
	return &AdminHandler{db: db, docker: docker}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	type userWithCount struct {
		models.User
		ProxyCount int `json:"proxy_count"`
	}

	result := make([]userWithCount, 0, len(users))
	for _, u := range users {
		count, _ := h.db.CountProxiesByUser(u.ID)
		result = append(result, userWithCount{User: u, ProxyCount: count})
	}

	writeJSON(w, http.StatusOK, result)
}

type updateUserRequest struct {
	Role       string `json:"role,omitempty"`
	MaxProxies *int   `json:"max_proxies,omitempty"`
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := user.Role
	if req.Role != "" {
		if req.Role != string(models.RoleUser) && req.Role != string(models.RoleAdmin) {
			writeError(w, http.StatusBadRequest, "role must be 'user' or 'admin'")
			return
		}
		role = models.Role(req.Role)
	}

	maxProxies := user.MaxProxies
	if req.MaxProxies != nil {
		maxProxies = *req.MaxProxies
	}

	if err := h.db.UpdateUser(id, role, maxProxies); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	user.Role = role
	user.MaxProxies = maxProxies
	writeJSON(w, http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	claims := getClaims(r)
	if claims != nil && claims.UserID == id {
		writeError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}

	proxies, err := h.db.ListProxiesByUser(id)
	if err == nil {
		for _, p := range proxies {
			if p.ContainerID != "" {
				h.docker.RemoveProxy(r.Context(), p.ContainerID)
			}
			h.db.DeleteProxy(p.ID)
		}
	}

	if err := h.db.DeleteUser(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func (h *AdminHandler) ListAllProxies(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.db.ListAllProxies()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	serverIP := h.docker.GetServerIP()
	for i := range proxies {
		if serverIP != "" {
			proxies[i].Link = fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s", serverIP, proxies[i].Port, proxies[i].Secret)
		}
	}

	if proxies == nil {
		proxies = []models.Proxy{}
	}

	writeJSON(w, http.StatusOK, proxies)
}

func (h *AdminHandler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return
	}

	proxy, err := h.db.GetProxy(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return
	}

	if proxy.ContainerID != "" {
		h.docker.RemoveProxy(r.Context(), proxy.ContainerID)
	}

	if err := h.db.DeleteProxy(proxy.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete proxy")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "proxy deleted"})
}
