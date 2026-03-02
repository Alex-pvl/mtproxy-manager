package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
)

type TelegramHandler struct {
	db     *database.DB
	jwtSvc *auth.JWTService
	cfg    *config.Config
}

func NewTelegramHandler(db *database.DB, jwtSvc *auth.JWTService, cfg *config.Config) *TelegramHandler {
	return &TelegramHandler{
		db:     db,
		jwtSvc: jwtSvc,
		cfg:    cfg,
	}
}

type telegramAuthRequest struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
	Ref       string `json:"ref"`
}

const maxAuthAge = 24 * time.Hour

// verifyTelegramHash checks the data integrity using HMAC-SHA-256
// as described at https://core.telegram.org/widgets/login#checking-authorization
func (h *TelegramHandler) verifyTelegramHash(req *telegramAuthRequest) bool {
	fields := make([]string, 0, 6)

	if req.AuthDate != 0 {
		fields = append(fields, "auth_date="+strconv.FormatInt(req.AuthDate, 10))
	}
	if req.FirstName != "" {
		fields = append(fields, "first_name="+req.FirstName)
	}
	if req.ID != 0 {
		fields = append(fields, "id="+strconv.FormatInt(req.ID, 10))
	}
	if req.LastName != "" {
		fields = append(fields, "last_name="+req.LastName)
	}
	if req.PhotoURL != "" {
		fields = append(fields, "photo_url="+req.PhotoURL)
	}
	if req.Username != "" {
		fields = append(fields, "username="+req.Username)
	}

	sort.Strings(fields)
	dataCheckString := strings.Join(fields, "\n")

	secretKey := sha256.Sum256([]byte(h.cfg.TelegramBotToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedHash), []byte(req.Hash))
}

func (h *TelegramHandler) Auth(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TelegramBotToken == "" {
		writeError(w, http.StatusServiceUnavailable, "telegram login is not configured")
		return
	}

	var req telegramAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ID == 0 || req.Hash == "" || req.AuthDate == 0 {
		writeError(w, http.StatusBadRequest, "id, hash and auth_date are required")
		return
	}

	if !h.verifyTelegramHash(&req) {
		writeError(w, http.StatusUnauthorized, "invalid telegram auth hash")
		return
	}

	authTime := time.Unix(req.AuthDate, 0)
	if time.Since(authTime) > maxAuthAge {
		writeError(w, http.StatusUnauthorized, "telegram auth data is too old")
		return
	}

	user, err := h.db.GetUserByTelegramID(req.ID)
	if err != nil {
		username := req.Username
		if username == "" {
			username = fmt.Sprintf("tg_%d", req.ID)
		}

		var referrerID *int64
		if req.Ref != "" {
			if id, err := h.db.GetUserIDByReferralCode(req.Ref); err == nil {
				referrerID = &id
			}
		}

		user, err = h.db.CreateUserByTelegram(req.ID, username, referrerID)
		if err != nil {
			existingUser, lookupErr := h.db.GetUserByUsername(username)
			if lookupErr == nil && existingUser.TelegramID == 0 {
				writeError(w, http.StatusConflict, "username already taken by another account")
				return
			}
			username = fmt.Sprintf("tg_%d", req.ID)
			user, err = h.db.CreateUserByTelegram(req.ID, username, referrerID)
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
