package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type WebAppHandler struct {
	db     *database.DB
	jwtSvc *auth.JWTService
	cfg    *config.Config
}

func NewWebAppHandler(db *database.DB, jwtSvc *auth.JWTService, cfg *config.Config) *WebAppHandler {
	return &WebAppHandler{db: db, jwtSvc: jwtSvc, cfg: cfg}
}

type webAppAuthRequest struct {
	InitData string `json:"init_data"`
	Ref      string `json:"ref"`
}

type webAppUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
}

// Auth validates Telegram Mini App initData and returns a JWT.
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func (h *WebAppHandler) Auth(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TelegramBotToken == "" {
		writeError(w, http.StatusServiceUnavailable, "telegram bot token is not configured")
		return
	}

	var req webAppAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.InitData == "" {
		writeError(w, http.StatusBadRequest, "init_data is required")
		return
	}

	userData, err := h.validateInitData(req.InitData)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("invalid init_data: %v", err))
		return
	}

	if userData.ID == 0 {
		writeError(w, http.StatusBadRequest, "user id missing from init_data")
		return
	}

	user, err := h.db.GetUserByTelegramID(userData.ID)
	if err != nil {
		username := userData.Username
		if username == "" {
			username = fmt.Sprintf("tg_%d", userData.ID)
		}

		var referrerID *int64
		if req.Ref != "" {
			if id, refErr := h.db.GetUserIDByReferralCode(req.Ref); refErr == nil {
				referrerID = &id
			}
		}

		user, err = h.db.CreateUserByTelegram(userData.ID, username, referrerID)
		if err != nil {
			username = fmt.Sprintf("tg_%d", userData.ID)
			user, err = h.db.CreateUserByTelegram(userData.ID, username, referrerID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to create user")
				return
			}
		}
	}

	jwtToken, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	resp := authResponse{Token: jwtToken}
	resp.User.ID = user.ID
	resp.User.Username = user.Username
	resp.User.Role = string(user.Role)

	writeJSON(w, http.StatusOK, resp)
}

func (h *WebAppHandler) validateInitData(initData string) (*webAppUser, error) {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}

	hash := vals.Get("hash")
	if hash == "" {
		return nil, fmt.Errorf("missing hash")
	}

	authDateStr := vals.Get("auth_date")
	if authDateStr != "" {
		authDate, err := strconv.ParseInt(authDateStr, 10, 64)
		if err == nil {
			if time.Since(time.Unix(authDate, 0)) > maxAuthAge {
				return nil, fmt.Errorf("init_data is too old")
			}
		}
	}

	// Build data_check_string: sorted key=value pairs excluding "hash"
	pairs := make([]string, 0, len(vals))
	for k := range vals {
		if k == "hash" {
			continue
		}
		pairs = append(pairs, k+"="+vals.Get(k))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC-SHA256("WebAppData", bot_token)
	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	secretMAC.Write([]byte(h.cfg.TelegramBotToken))
	secretKey := secretMAC.Sum(nil)

	// computed_hash = HMAC-SHA256(secret_key, data_check_string)
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(computedHash), []byte(hash)) {
		return nil, fmt.Errorf("hash mismatch")
	}

	// Parse user from init_data
	userJSON := vals.Get("user")
	if userJSON == "" {
		return nil, fmt.Errorf("missing user field")
	}

	var userData webAppUser
	if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
		return nil, fmt.Errorf("parse user: %w", err)
	}

	return &userData, nil
}
