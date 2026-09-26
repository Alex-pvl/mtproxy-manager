package handlers

import (
	"fmt"
	"log"
	"net/http"
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

func (h *ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
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
			writeError(w, http.StatusForbidden, "active subscription required to create a VPN")
			return
		}
	}

	count, err := h.db.CountProxiesByUser(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count VPNs")
		return
	}
	if count >= user.MaxProxies {
		writeError(w, http.StatusForbidden, fmt.Sprintf("VPN limit reached (%d/%d)", count, user.MaxProxies))
		return
	}

	if h.xuiClient == nil {
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}

	// A unique numeric id per VPN record; also used to derive the x-ui client email.
	// ponytail: reuse the existing port allocator as the record id — no separate id scheme.
	id, err := h.docker.AllocatePort()
	if err != nil {
		log.Printf("allocate vpn id: %v", err)
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}

	uuid, err := xui.GenerateUUID()
	if err != nil {
		log.Printf("vless: generate uuid failed: user_id=%d err=%v", claims.UserID, err)
		writeError(w, http.StatusInternalServerError, "failed to create VPN")
		return
	}

	// Expiry matches the user's active subscription so x-ui blocks access when it ends.
	var expiryTime time.Time
	if sub, subErr := h.db.GetActiveSubscription(claims.UserID); subErr == nil && sub != nil {
		expiryTime = sub.ExpiresAt
	}
	email := vlessEmail(id, claims.UserID)
	if err := h.xuiClient.AddClient(uuid, email, expiryTime); err != nil {
		expStr := "none"
		if !expiryTime.IsZero() {
			expStr = expiryTime.UTC().Format(time.RFC3339)
		}
		log.Printf("vless: x-ui addClient failed: %s user_id=%d xui_client_email=%q vless_uuid=%s subscription_expiry_utc=%s err=%v",
			h.xuiClient.DescribeForLog(), claims.UserID, email, uuid, expStr, err)
		writeError(w, http.StatusInternalServerError, "failed to create VPN")
		return
	}

	proxy := &models.Proxy{
		UserID:    claims.UserID,
		Port:      id,
		Status:    models.StatusRunning,
		VlessUUID: uuid,
	}

	if err := h.db.CreateProxy(proxy); err != nil {
		_ = h.xuiClient.RemoveClient(uuid)
		writeError(w, http.StatusInternalServerError, "failed to save VPN")
		return
	}

	remark := fmt.Sprintf("stay-proxy-%d", proxy.ID)
	proxy.LinkSub = h.xuiClient.UserLink(uuid, remark)
	if proxy.LinkSub == "" {
		log.Printf("vless: UserLink returned empty: user_id=%d proxy_db_id=%d vless_uuid=%q remark=%q %s",
			claims.UserID, proxy.ID, uuid, remark, h.xuiClient.DescribeForLog())
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
		writeError(w, http.StatusInternalServerError, "failed to list VPNs")
		return
	}

	for i := range proxies {
		if h.xuiClient != nil && proxies[i].VlessUUID != "" {
			remark := fmt.Sprintf("stay-proxy-%d", proxies[i].ID)
			proxies[i].LinkSub = h.xuiClient.UserLink(proxies[i].VlessUUID, remark)
		}
	}

	if proxies == nil {
		proxies = []models.Proxy{}
	}

	writeJSON(w, http.StatusOK, proxies)
}

func (h *ProxyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	proxy, ok := h.getOwnedProxy(w, r)
	if !ok {
		return
	}

	if h.xuiClient != nil && proxy.VlessUUID != "" {
		if err := h.xuiClient.RemoveClient(proxy.VlessUUID); err != nil {
			log.Printf("xui remove client (proxy id=%d uuid=%s): %v", proxy.ID, proxy.VlessUUID, err)
		}
	}

	if err := h.db.DeleteProxy(proxy.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete VPN")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "vpn deleted"})
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
		writeError(w, http.StatusBadRequest, "invalid vpn id")
		return nil, false
	}

	proxy, err := h.db.GetProxy(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "vpn not found")
		return nil, false
	}

	if proxy.UserID != claims.UserID && claims.Role != models.RoleAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return nil, false
	}

	return proxy, true
}

// vlessEmail returns a deterministic x-ui client email for a given record id and user.
// This lets us reconstruct it later when updating expiry without storing it separately.
func vlessEmail(id int, userID int64) string {
	return fmt.Sprintf("proxy-%d-user-%d", id, userID)
}
