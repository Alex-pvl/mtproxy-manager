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

type ProxyHandler struct {
	db      *database.DB
	docker  *docker.Manager
}

func NewProxyHandler(db *database.DB, docker *docker.Manager) *ProxyHandler {
	return &ProxyHandler{db: db, docker: docker}
}

type createProxyRequest struct {
	Domain string `json:"domain"`
	Port   int    `json:"port,omitempty"`
}

func (h *ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Domain == "" {
		writeError(w, http.StatusBadRequest, "domain is required")
		return
	}

	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if user.Role != models.RoleAdmin {
		sub, _ := h.db.GetActiveSubscription(claims.UserID)
		if sub == nil {
			writeError(w, http.StatusForbidden, "active subscription required to create proxies")
			return
		}
	}

	count, err := h.db.CountProxiesByUser(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count proxies")
		return
	}

	if count >= user.MaxProxies {
		writeError(w, http.StatusForbidden, fmt.Sprintf("proxy limit reached (%d/%d)", count, user.MaxProxies))
		return
	}

	port := req.Port
	if port == 0 {
		port, err = h.docker.AllocatePort()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "no free ports available")
			return
		}
	} else {
		used, err := h.db.IsPortUsed(port)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check port")
			return
		}
		if used {
			writeError(w, http.StatusConflict, "port already in use")
			return
		}
	}

	ctx := r.Context()

	secret, err := h.docker.GenerateSecret(ctx, req.Domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate secret: "+err.Error())
		return
	}

	containerName := fmt.Sprintf("mtg-%d", port)
	containerID, err := h.docker.CreateAndStartProxy(ctx, port, secret, containerName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start proxy: "+err.Error())
		return
	}

	proxy := &models.Proxy{
		UserID:        claims.UserID,
		Port:          port,
		Domain:        req.Domain,
		Secret:        secret,
		ContainerID:   containerID,
		ContainerName: containerName,
		Status:        models.StatusRunning,
	}

	if err := h.db.CreateProxy(proxy); err != nil {
		h.docker.RemoveProxy(ctx, containerID)
		writeError(w, http.StatusInternalServerError, "failed to save proxy")
		return
	}

	serverIP := h.docker.GetServerIP()
	if serverIP != "" {
		proxy.Link = fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s", serverIP, port, secret)
	}

	writeJSON(w, http.StatusCreated, proxy)
}

func (h *ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	proxies, err := h.db.ListProxiesByUser(claims.UserID)
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

func (h *ProxyHandler) Stop(w http.ResponseWriter, r *http.Request) {
	proxy, ok := h.getOwnedProxy(w, r)
	if !ok {
		return
	}

	if proxy.Status != models.StatusRunning {
		writeError(w, http.StatusBadRequest, "proxy is not running")
		return
	}

	if err := h.docker.StopProxy(r.Context(), proxy.ContainerID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stop proxy: "+err.Error())
		return
	}

	h.db.UpdateProxyStatus(proxy.ID, models.StatusStopped, proxy.ContainerID)
	proxy.Status = models.StatusStopped
	writeJSON(w, http.StatusOK, proxy)
}

func (h *ProxyHandler) Start(w http.ResponseWriter, r *http.Request) {
	proxy, ok := h.getOwnedProxy(w, r)
	if !ok {
		return
	}

	if proxy.Status == models.StatusRunning {
		writeError(w, http.StatusBadRequest, "proxy is already running")
		return
	}

	if err := h.docker.StartProxy(r.Context(), proxy.ContainerID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start proxy: "+err.Error())
		return
	}

	h.db.UpdateProxyStatus(proxy.ID, models.StatusRunning, proxy.ContainerID)
	proxy.Status = models.StatusRunning
	writeJSON(w, http.StatusOK, proxy)
}

func (h *ProxyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	proxy, ok := h.getOwnedProxy(w, r)
	if !ok {
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

func (h *ProxyHandler) getOwnedProxy(w http.ResponseWriter, r *http.Request) (*models.Proxy, bool) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return nil, false
	}

	proxy, err := h.db.GetProxy(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "proxy not found")
		return nil, false
	}

	if proxy.UserID != claims.UserID && claims.Role != models.RoleAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return nil, false
	}

	return proxy, true
}

