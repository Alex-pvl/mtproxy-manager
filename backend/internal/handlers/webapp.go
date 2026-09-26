package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
)

const maxInitDataAge = 24 * time.Hour

// WebAppHandler logs users in from the Telegram Mini App.
type WebAppHandler struct {
	db     *database.DB
	jwtSvc *auth.JWTService
	cfg    *config.Config
}

func NewWebAppHandler(db *database.DB, jwtSvc *auth.JWTService, cfg *config.Config) *WebAppHandler {
	return &WebAppHandler{db: db, jwtSvc: jwtSvc, cfg: cfg}
}

func (h *WebAppHandler) Auth(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TelegramBotToken == "" {
		writeError(w, http.StatusServiceUnavailable, "telegram bot token is not configured")
		return
	}
	var req struct {
		InitData string `json:"init_data"`
		Ref      string `json:"ref"`
	}
	if err := readJSON(r, &req); err != nil || req.InitData == "" {
		writeError(w, http.StatusBadRequest, "init_data is required")
		return
	}
	tgUser, err := validateInitData(req.InitData, h.cfg.TelegramBotToken, time.Now())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid init_data: "+err.Error())
		return
	}
	user, err := findOrCreateTelegramUser(h.db, tgUser.ID, tgUser.Username, req.Ref)
	if err != nil {
		log.Printf("webapp auth tg=%d: %v", tgUser.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	token, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	writeJSON(w, http.StatusOK, buildAuthResponse(token, user))
}

type webAppUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// validateInitData checks Mini App initData as described in
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func validateInitData(initData, botToken string, now time.Time) (*webAppUser, error) {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	hash := vals.Get("hash")
	if hash == "" {
		return nil, errors.New("missing hash")
	}

	pairs := make([]string, 0, len(vals))
	for k := range vals {
		if k != "hash" {
			pairs = append(pairs, k+"="+vals.Get(k))
		}
	}
	sort.Strings(pairs)

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(pairs, "\n")))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(hash)) {
		return nil, errors.New("hash mismatch")
	}

	authDate, err := strconv.ParseInt(vals.Get("auth_date"), 10, 64)
	if err != nil {
		return nil, errors.New("missing auth_date")
	}
	if now.Sub(time.Unix(authDate, 0)) > maxInitDataAge {
		return nil, errors.New("init_data is too old")
	}

	var user webAppUser
	if err := json.Unmarshal([]byte(vals.Get("user")), &user); err != nil || user.ID == 0 {
		return nil, errors.New("missing user")
	}
	return &user, nil
}
