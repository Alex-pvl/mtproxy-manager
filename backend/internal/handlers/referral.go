package handlers

import (
	"net/http"
	"strings"

	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
)

type ReferralHandler struct {
	db  *database.DB
	cfg *config.Config
}

func NewReferralHandler(db *database.DB, cfg *config.Config) *ReferralHandler {
	return &ReferralHandler{db: db, cfg: cfg}
}

func (h *ReferralHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	_, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	code, err := h.db.GetOrCreateReferralCode(claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get referral code")
		return
	}

	baseURL := strings.TrimSuffix(h.cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://example.com"
	}
	referralLink := baseURL + "/register?ref=" + code

	invited, _ := h.db.CountReferredBy(claims.UserID)
	bonusDays, _ := h.db.SumBonusDaysReceived(claims.UserID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"referral_link":       referralLink,
		"invited_count":       invited,
		"bonus_days_received": bonusDays,
	})
}
