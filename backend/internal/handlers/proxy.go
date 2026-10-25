package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
)

// ProxyHandler serves /api/proxies: the current user's VPN configs.
type ProxyHandler struct {
	db  *database.DB
	vpn *VPN
}

func NewProxyHandler(db *database.DB, vpn *VPN) *ProxyHandler {
	return &ProxyHandler{db: db, vpn: vpn}
}

func (h *ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	if user.Role != models.RoleAdmin {
		if sub, _ := h.db.GetActiveSubscription(user.ID); sub == nil {
			writeError(w, http.StatusForbidden, "active subscription required to create a connection")
			return
		}
	}
	count, err := h.db.CountProxiesByUser(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count connections")
		return
	}
	if count >= user.MaxProxies {
		writeError(w, http.StatusForbidden, fmt.Sprintf("connection limit reached (%d/%d)", count, user.MaxProxies))
		return
	}

	proxy, err := h.vpn.Create(user)
	if err != nil {
		log.Printf("create vpn user=%d: %v", user.ID, err)
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}
	writeJSON(w, http.StatusCreated, proxy)
}

// StartTrial gives a Telegram user who never subscribed one day and one
// connection capped at 2 GiB.
func (h *ProxyHandler) StartTrial(w http.ResponseWriter, r *http.Request) {
	user, err := h.db.GetUserByID(getClaims(r).UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	if ok, err := h.db.TrialAvailable(user); err != nil || !ok {
		writeError(w, http.StatusForbidden, "free trial is not available")
		return
	}
	if h.vpn.xui == nil {
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}
	sub, started, err := h.db.StartTrial(user.ID, time.Now())
	if err != nil || !started {
		writeError(w, http.StatusForbidden, "free trial is not available")
		return
	}
	if err := h.db.UpdateUser(user.ID, user.Role, 1, user.HideSubBanner); err != nil {
		log.Printf("trial user=%d: set max proxies: %v", user.ID, err)
	}
	proxy, err := h.vpn.Create(user)
	if err != nil {
		// The trial stays used; the user can still create the connection from "My services".
		log.Printf("trial user=%d: create vpn: %v", user.ID, err)
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable, please try again later")
		return
	}
	log.Printf("trial started: user=%d expires=%s", user.ID, sub.ExpiresAt.Format(time.RFC3339))
	writeJSON(w, http.StatusCreated, proxy)
}

func (h *ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.vpn.List(getClaims(r).UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list connections")
		return
	}
	writeJSON(w, http.StatusOK, proxies)
}

func (h *ProxyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	proxy, err := h.db.GetProxy(id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && proxy.UserID != getClaims(r).UserID) {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get connection")
		return
	}
	// ponytail: recreating a trial connection would reset its 2 GiB counter.
	if sub, _ := h.db.GetActiveSubscription(proxy.UserID); sub != nil && sub.PlanID == models.TrialPlanID {
		writeError(w, http.StatusForbidden, "connections cannot be deleted during the free trial")
		return
	}
	if err := h.vpn.Delete(proxy); err != nil {
		log.Printf("delete vpn id=%d: %v", proxy.ID, err)
		writeError(w, http.StatusServiceUnavailable, "failed to delete connection, please try again later")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "connection deleted"})
}
