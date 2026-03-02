package handlers

import (
	"net/http"
	"time"

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
			PlanID:    sub.PlanID,
			PlanName:  planName,
			ExpiresAt: &sub.ExpiresAt,
		}
	} else {
		resp.Subscription = &subscriptionInfo{Active: false}
	}

	writeJSON(w, http.StatusOK, resp)
}
