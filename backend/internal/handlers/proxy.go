package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/docker"
	"mtproxy-manager/internal/models"
	"mtproxy-manager/internal/xui"

	"github.com/go-chi/chi/v5"
)

type ProxyHandler struct {
	db        *database.DB
	docker    *docker.Manager
	xuiClient *xui.Client
	serverIP  string
}

func NewProxyHandler(db *database.DB, docker *docker.Manager, xuiClient *xui.Client) *ProxyHandler {
	return &ProxyHandler{db: db, docker: docker, xuiClient: xuiClient}
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
			log.Printf("allocate port: %v", err)
			writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
			return
		}
	} else {
		used, err := h.db.IsPortUsed(port)
		if err != nil {
			log.Printf("check port %d: %v", port, err)
			writeError(w, http.StatusInternalServerError, "failed to create proxy")
			return
		}
		if used {
			writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
			return
		}
	}

	socks5Port, err := h.docker.AllocateSOCKS5Port()
	if err != nil {
		log.Printf("allocate socks5 port: %v", err)
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}

	ctx := r.Context()

	secret, err := h.docker.GenerateSecret(ctx, req.Domain)
	if err != nil {
		log.Printf("generate secret: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	socks5User, socks5Pass, err := generateSocks5Credentials()
	if err != nil {
		log.Printf("generate socks5 credentials: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	containerName := fmt.Sprintf("mtg-%d", port)
	containerID, err := h.docker.CreateAndStartProxy(ctx, port, secret, containerName)
	if err != nil {
		log.Printf("create proxy container: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	socks5ContainerName := fmt.Sprintf("socks5-%d", socks5Port)
	socks5ContainerID, err := h.docker.CreateAndStartSOCKS5Proxy(ctx, socks5Port, socks5User, socks5Pass, socks5ContainerName)
	if err != nil {
		h.docker.RemoveProxy(ctx, containerID)
		log.Printf("create socks5 container: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	// Generate VLESS UUID and register client in x-ui if enabled.
	// Expiry is set to match the user's active subscription so x-ui
	// automatically blocks access when the subscription ends.
	var vlessUUID string
	if h.xuiClient != nil {
		uuid, err := xui.GenerateUUID()
		if err != nil {
			log.Printf("generate vless uuid: %v", err)
		} else {
			var expiryTime time.Time
			if sub, subErr := h.db.GetActiveSubscription(claims.UserID); subErr == nil && sub != nil {
				expiryTime = sub.ExpiresAt
			}
			email := vlessEmail(port, claims.UserID)
			if err := h.xuiClient.AddClient(uuid, email, expiryTime); err != nil {
				log.Printf("xui add client (proxy port=%d): %v", port, err)
			} else {
				vlessUUID = uuid
			}
		}
	}

	proxy := &models.Proxy{
		UserID:              claims.UserID,
		Port:                port,
		Domain:              req.Domain,
		Secret:              secret,
		ContainerID:         containerID,
		ContainerName:       containerName,
		Status:              models.StatusRunning,
		Socks5Port:          socks5Port,
		Socks5User:          socks5User,
		Socks5Pass:          socks5Pass,
		Socks5ContainerID:   socks5ContainerID,
		Socks5ContainerName: socks5ContainerName,
		VlessUUID:           vlessUUID,
	}

	if err := h.db.CreateProxy(proxy); err != nil {
		h.docker.RemoveProxy(ctx, containerID)
		h.docker.RemoveProxy(ctx, socks5ContainerID)
		if h.xuiClient != nil && vlessUUID != "" {
			_ = h.xuiClient.RemoveClient(vlessUUID)
		}
		writeError(w, http.StatusInternalServerError, "failed to save proxy")
		return
	}

	serverIP := h.docker.GetServerIP()
	if serverIP != "" {
		proxy.Link = fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s", serverIP, port, secret)
		proxy.LinkSocks5 = fmt.Sprintf("https://t.me/socks?server=%s&port=%d&user=%s&pass=%s",
			serverIP, socks5Port, url.QueryEscape(socks5User), url.QueryEscape(socks5Pass))
	}
	if h.xuiClient != nil && vlessUUID != "" {
		remark := fmt.Sprintf("stay-proxy-%d", proxy.ID)
		proxy.LinkVless = h.xuiClient.BuildLink(vlessUUID, serverIP, remark)
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
			if proxies[i].Socks5Port > 0 && proxies[i].Socks5User != "" {
				proxies[i].LinkSocks5 = fmt.Sprintf("https://t.me/socks?server=%s&port=%d&user=%s&pass=%s",
					serverIP, proxies[i].Socks5Port, url.QueryEscape(proxies[i].Socks5User), url.QueryEscape(proxies[i].Socks5Pass))
			}
		}
		if h.xuiClient != nil && proxies[i].VlessUUID != "" {
			remark := fmt.Sprintf("stay-proxy-%d", proxies[i].ID)
			proxies[i].LinkVless = h.xuiClient.BuildLink(proxies[i].VlessUUID, serverIP, remark)
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
		log.Printf("stop proxy %d: %v", proxy.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to stop proxy")
		return
	}

	if proxy.Socks5ContainerID != "" {
		_ = h.docker.StopProxy(r.Context(), proxy.Socks5ContainerID)
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
		log.Printf("start proxy %d: %v", proxy.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to start proxy")
		return
	}

	if proxy.Socks5ContainerID != "" {
		_ = h.docker.StartProxy(r.Context(), proxy.Socks5ContainerID)
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
	if proxy.Socks5ContainerID != "" {
		h.docker.RemoveProxy(r.Context(), proxy.Socks5ContainerID)
	}
	if h.xuiClient != nil && proxy.VlessUUID != "" {
		if err := h.xuiClient.RemoveClient(proxy.VlessUUID); err != nil {
			log.Printf("xui remove client (proxy id=%d uuid=%s): %v", proxy.ID, proxy.VlessUUID, err)
		}
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

// vlessEmail returns a deterministic x-ui client email for a given proxy port and user.
// This lets us reconstruct it later when updating expiry without storing it separately.
func vlessEmail(port int, userID int64) string {
	return fmt.Sprintf("proxy-%d-user-%d", port, userID)
}

func generateSocks5Credentials() (user, pass string, err error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	user = "s" + hex.EncodeToString(b[:6])
	pass = hex.EncodeToString(b[6:])
	return user, pass, nil
}
