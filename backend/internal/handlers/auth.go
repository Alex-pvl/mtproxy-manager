package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
)

type AuthHandler struct {
	db     *database.DB
	jwtSvc *auth.JWTService
}

func NewAuthHandler(db *database.DB, jwtSvc *auth.JWTService) *AuthHandler {
	return &AuthHandler{db: db, jwtSvc: jwtSvc}
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Ref      string `json:"ref"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Username) < 3 || len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "username must be at least 3 chars, password at least 6")
		return
	}

	var referrerID *int64
	if req.Ref != "" {
		if id, err := h.db.GetUserIDByReferralCode(req.Ref); err == nil {
			referrerID = &id
		}
	}

	user, err := h.db.CreateUser(req.Username, req.Password, referrerID)
	if err != nil {
		writeError(w, http.StatusConflict, "username already taken")
		return
	}

	token, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = user.ID
	resp.User.Username = user.Username
	resp.User.Role = string(user.Role)

	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	resp := authResponse{Token: token}
	resp.User.ID = user.ID
	resp.User.Username = user.Username
	resp.User.Role = string(user.Role)

	writeJSON(w, http.StatusOK, resp)
}

type subscriptionInfo struct {
	Active    bool       `json:"active"`
	PlanName  string     `json:"plan_name,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type meResponse struct {
	*models.User
	Subscription *subscriptionInfo `json:"subscription"`
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	resp := meResponse{User: user}

	sub, _ := h.db.GetActiveSubscription(user.ID)
	if sub != nil {
		plan := models.GetPlan(sub.PlanID)
		planName := sub.PlanID
		if plan != nil {
			planName = plan.Name
		}
		resp.Subscription = &subscriptionInfo{
			Active:    true,
			PlanName:  planName,
			ExpiresAt: &sub.ExpiresAt,
		}
	} else {
		resp.Subscription = &subscriptionInfo{Active: false}
	}

	writeJSON(w, http.StatusOK, resp)
}
