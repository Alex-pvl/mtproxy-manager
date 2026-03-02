package handlers

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"

	"github.com/golang-jwt/jwt/v5"
)

type TelegramHandler struct {
	db     *database.DB
	jwtSvc *auth.JWTService
	cfg    *config.Config

	jwksMu    sync.RWMutex
	jwksCache map[string]*rsa.PublicKey
	jwksExp   time.Time
}

func NewTelegramHandler(db *database.DB, jwtSvc *auth.JWTService, cfg *config.Config) *TelegramHandler {
	return &TelegramHandler{
		db:     db,
		jwtSvc: jwtSvc,
		cfg:    cfg,
	}
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (h *TelegramHandler) getPublicKeys() (map[string]*rsa.PublicKey, error) {
	h.jwksMu.RLock()
	if h.jwksCache != nil && time.Now().Before(h.jwksExp) {
		defer h.jwksMu.RUnlock()
		return h.jwksCache, nil
	}
	h.jwksMu.RUnlock()

	h.jwksMu.Lock()
	defer h.jwksMu.Unlock()

	if h.jwksCache != nil && time.Now().Before(h.jwksExp) {
		return h.jwksCache, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://oauth.telegram.org/.well-known/jwks.json")
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS returned status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			continue
		}

		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)

		keys[k.Kid] = &rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		}
	}

	h.jwksCache = keys
	h.jwksExp = time.Now().Add(1 * time.Hour)

	return keys, nil
}

type telegramClaims struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
	PhoneNumber       string `json:"phone_number"`
	jwt.RegisteredClaims
}

type telegramAuthRequest struct {
	IDToken string `json:"id_token"`
	Ref     string `json:"ref"`
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

	if req.IDToken == "" {
		writeError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	keys, err := h.getPublicKeys()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch Telegram keys")
		return
	}

	token, err := jwt.ParseWithClaims(req.IDToken, &telegramClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		key, found := keys[kid]
		if !found {
			return nil, fmt.Errorf("key %s not found in JWKS", kid)
		}

		return key, nil
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid telegram token")
		return
	}

	claims, ok := token.Claims.(*telegramClaims)
	if !ok || !token.Valid {
		writeError(w, http.StatusUnauthorized, "invalid telegram token claims")
		return
	}

	if claims.Issuer != "https://oauth.telegram.org" {
		writeError(w, http.StatusUnauthorized, "invalid token issuer")
		return
	}

	if claims.ID == 0 {
		writeError(w, http.StatusUnauthorized, "telegram user id not found in token")
		return
	}

	user, err := h.db.GetUserByTelegramID(claims.ID)
	if err != nil {
		username := claims.PreferredUsername
		if username == "" {
			username = fmt.Sprintf("tg_%d", claims.ID)
		}

		var referrerID *int64
		if req.Ref != "" {
			if id, err := h.db.GetUserIDByReferralCode(req.Ref); err == nil {
				referrerID = &id
			}
		}

		user, err = h.db.CreateUserByTelegram(claims.ID, username, referrerID)
		if err != nil {
			existingUser, lookupErr := h.db.GetUserByUsername(username)
			if lookupErr == nil && existingUser.TelegramID == 0 {
				writeError(w, http.StatusConflict, "username already taken by another account")
				return
			}
			username = fmt.Sprintf("tg_%d", claims.ID)
			user, err = h.db.CreateUserByTelegram(claims.ID, username, referrerID)
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
