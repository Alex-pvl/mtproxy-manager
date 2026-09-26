package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db     *database.DB
	jwtSvc *auth.JWTService
}

func NewAuthHandler(db *database.DB, jwtSvc *auth.JWTService) *AuthHandler {
	return &AuthHandler{db: db, jwtSvc: jwtSvc}
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

type subscriptionInfo struct {
	Active    bool       `json:"active"`
	PlanID    string     `json:"plan_id,omitempty"`
	PlanName  string     `json:"plan_name,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type meResponse struct {
	*models.User
	Subscription *subscriptionInfo `json:"subscription"`
}

type credentialsAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func buildAuthResponse(jwtToken string, user *models.User) authResponse {
	resp := authResponse{Token: jwtToken}
	resp.User.ID = user.ID
	resp.User.Username = user.Username
	resp.User.Role = string(user.Role)
	return resp
}

func normalizeUsername(username string) string {
	return strings.TrimSpace(strings.ToLower(username))
}

func validateCredentials(req credentialsAuthRequest) (string, string, string) {
	username := normalizeUsername(req.Username)
	password := strings.TrimSpace(req.Password)

	if len(username) < 3 {
		return "", "", "username must be at least 3 characters"
	}
	if len(username) > 64 {
		return "", "", "username must be at most 64 characters"
	}
	if len(password) < 8 {
		return "", "", "password must be at least 8 characters"
	}
	if len(password) > 128 {
		return "", "", "password must be at most 128 characters"
	}

	return username, password, ""
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsAuthRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	username, password, validationErr := validateCredentials(req)
	if validationErr != "" {
		writeError(w, http.StatusBadRequest, validationErr)
		return
	}

	if _, err := h.db.GetUserByUsername(username); err == nil {
		writeError(w, http.StatusConflict, "username already taken")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to check user")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := h.db.CreateUser(username, string(hash))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	jwtToken, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, buildAuthResponse(jwtToken, user))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsAuthRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	username, password, validationErr := validateCredentials(req)
	if validationErr != "" {
		writeError(w, http.StatusBadRequest, validationErr)
		return
	}

	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if user.PasswordHash == "" {
		writeError(w, http.StatusUnauthorized, "password login is not available for this account")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	jwtToken, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, buildAuthResponse(jwtToken, user))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.db.GetUserByID(getClaims(r).UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: user, Subscription: loadSubscriptionInfo(h.db, user.ID)})
}

func loadSubscriptionInfo(db *database.DB, userID int64) *subscriptionInfo {
	sub, _ := db.GetActiveSubscription(userID)
	if sub == nil {
		return &subscriptionInfo{Active: false}
	}
	info := &subscriptionInfo{Active: true, PlanID: sub.PlanID, PlanName: sub.PlanID, ExpiresAt: &sub.ExpiresAt}
	if plan := models.GetPlan(sub.PlanID); plan != nil {
		info.PlanName = plan.Name
	}
	return info
}

// findOrCreateTelegramUser logs in a Telegram user, registering them on first
// visit (with an optional referral code). A taken username falls back to tg_<id>.
func findOrCreateTelegramUser(db *database.DB, telegramID int64, username, ref string) (*models.User, error) {
	user, err := db.GetUserByTelegramID(telegramID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var referrerID *int64
	if ref != "" {
		if id, err := db.GetUserIDByReferralCode(ref); err == nil {
			referrerID = &id
		}
	}
	fallback := fmt.Sprintf("tg_%d", telegramID)
	if username == "" {
		username = fallback
	}
	user, err = db.CreateUserByTelegram(telegramID, username, referrerID)
	if err != nil && username != fallback {
		user, err = db.CreateUserByTelegram(telegramID, fallback, referrerID)
	}
	return user, err
}
