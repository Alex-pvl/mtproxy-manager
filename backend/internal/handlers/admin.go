package handlers

import (
	"log"
	"net/http"

	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
)

type AdminHandler struct {
	db  *database.DB
	vpn *VPN
}

func NewAdminHandler(db *database.DB, vpn *VPN) *AdminHandler {
	return &AdminHandler{db: db, vpn: vpn}
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

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	user, err := h.db.GetUserByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	var req struct {
		Role          string `json:"role"`
		MaxProxies    *int   `json:"max_proxies"`
		HideSubBanner *bool  `json:"hide_sub_banner"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	switch models.Role(req.Role) {
	case "":
	case models.RoleUser, models.RoleAdmin:
		user.Role = models.Role(req.Role)
	default:
		writeError(w, http.StatusBadRequest, "role must be 'user' or 'admin'")
		return
	}
	if req.MaxProxies != nil {
		if *req.MaxProxies < 0 {
			writeError(w, http.StatusBadRequest, "max_proxies must be >= 0")
			return
		}
		user.MaxProxies = *req.MaxProxies
	}
	if req.HideSubBanner != nil {
		user.HideSubBanner = *req.HideSubBanner
	}
	if err := h.db.UpdateUser(id, user.Role, user.MaxProxies, user.HideSubBanner); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if getClaims(r).UserID == id {
		writeError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}
	if err := h.vpn.DeleteAllOf(id); err != nil {
		log.Printf("admin delete user=%d vpns: %v", id, err)
		writeError(w, http.StatusServiceUnavailable, "failed to delete user's connections in x-ui")
		return
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
		writeError(w, http.StatusInternalServerError, "failed to list connections")
		return
	}
	for i := range proxies {
		h.vpn.fillLink(&proxies[i])
	}
	writeJSON(w, http.StatusOK, proxies)
}

func (h *AdminHandler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	proxy, err := h.db.GetProxy(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}
	if err := h.vpn.Delete(proxy); err != nil {
		log.Printf("admin delete vpn id=%d: %v", id, err)
		writeError(w, http.StatusServiceUnavailable, "failed to delete connection in x-ui")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "connection deleted"})
}
