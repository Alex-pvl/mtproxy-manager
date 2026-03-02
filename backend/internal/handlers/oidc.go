package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"mtproxy-manager/internal/auth"
	"mtproxy-manager/internal/config"
	"mtproxy-manager/internal/database"
)

const (
	telegramAuthURL  = "https://oauth.telegram.org/auth"
	telegramTokenURL = "https://oauth.telegram.org/token"
	telegramJWKSURL  = "https://oauth.telegram.org/.well-known/jwks.json"
	telegramIssuer   = "https://oauth.telegram.org"
	oidcCookieName   = "oidc_state"
	oidcStateTTL     = 10 * time.Minute
	jwksCacheTTL     = 1 * time.Hour
)

type OIDCHandler struct {
	db         *database.DB
	jwtSvc     *auth.JWTService
	cfg        *config.Config
	httpClient *http.Client

	jwksMu     sync.RWMutex
	jwksKeys   map[string]interface{}
	jwksExpiry time.Time
}

func NewOIDCHandler(db *database.DB, jwtSvc *auth.JWTService, cfg *config.Config) *OIDCHandler {
	return &OIDCHandler{
		db:         db,
		jwtSvc:     jwtSvc,
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		jwksKeys:   make(map[string]interface{}),
	}
}

type oidcStateClaims struct {
	State        string `json:"state"`
	CodeVerifier string `json:"cv"`
	Ref          string `json:"ref,omitempty"`
	jwt.RegisteredClaims
}

// Init starts the OIDC Authorization Code Flow with PKCE.
func (h *OIDCHandler) Init(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TGClientID == "" || h.cfg.TGClientSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "telegram OIDC is not configured")
		return
	}

	state, err := randomString(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate state")
		return
	}

	codeVerifier, err := randomString(64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate code verifier")
		return
	}

	ref := r.URL.Query().Get("ref")

	stateClaims := oidcStateClaims{
		State:        state,
		CodeVerifier: codeVerifier,
		Ref:          ref,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(oidcStateTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	stateJWT, err := jwt.NewWithClaims(jwt.SigningMethodHS256, stateClaims).SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create state token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookieName,
		Value:    stateJWT,
		Path:     "/api/auth/oidc",
		MaxAge:   int(oidcStateTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(h.cfg.BaseURL, "https"),
	})

	challenge := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(challenge[:])

	redirectURI := h.cfg.BaseURL + "/api/auth/oidc/callback"

	params := url.Values{
		"client_id":             {h.cfg.TGClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile phone"},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	http.Redirect(w, r, telegramAuthURL+"?"+params.Encode(), http.StatusFound)
}

// Callback handles the redirect from Telegram after user authorization.
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		h.redirectWithError(w, r, "missing code or state")
		return
	}

	cookie, err := r.Cookie(oidcCookieName)
	if err != nil {
		h.redirectWithError(w, r, "missing state cookie")
		return
	}

	var stateClaims oidcStateClaims
	token, err := jwt.ParseWithClaims(cookie.Value, &stateClaims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		h.redirectWithError(w, r, "invalid state cookie")
		return
	}

	if stateClaims.State != state {
		h.redirectWithError(w, r, "state mismatch")
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oidcCookieName,
		Value:    "",
		Path:     "/api/auth/oidc",
		MaxAge:   -1,
		HttpOnly: true,
	})

	redirectURI := h.cfg.BaseURL + "/api/auth/oidc/callback"
	tokenResp, err := h.exchangeCode(code, stateClaims.CodeVerifier, redirectURI)
	if err != nil {
		log.Printf("OIDC token exchange error: %v", err)
		h.redirectWithError(w, r, "token exchange failed")
		return
	}

	idClaims, err := h.validateIDToken(tokenResp.IDToken)
	if err != nil {
		log.Printf("OIDC id_token validation error: %v", err)
		h.redirectWithError(w, r, "invalid id token")
		return
	}

	telegramID := idClaims.ID
	if telegramID == 0 {
		h.redirectWithError(w, r, "missing telegram user id")
		return
	}

	username := idClaims.PreferredUsername
	if username == "" {
		username = fmt.Sprintf("tg_%d", telegramID)
	}

	user, err := h.db.GetUserByTelegramID(telegramID)
	if err != nil {
		var referrerID *int64
		if stateClaims.Ref != "" {
			if id, refErr := h.db.GetUserIDByReferralCode(stateClaims.Ref); refErr == nil {
				referrerID = &id
			}
		}

		user, err = h.db.CreateUserByTelegram(telegramID, username, referrerID)
		if err != nil {
			if existingUser, lookupErr := h.db.GetUserByUsername(username); lookupErr == nil && existingUser.TelegramID == 0 {
				username = fmt.Sprintf("tg_%d", telegramID)
			}
			user, err = h.db.CreateUserByTelegram(telegramID, username, referrerID)
			if err != nil {
				h.redirectWithError(w, r, "failed to create user")
				return
			}
		}
	}

	jwtToken, err := h.jwtSvc.GenerateToken(user)
	if err != nil {
		h.redirectWithError(w, r, "failed to generate token")
		return
	}

	frontendURL := h.cfg.BaseURL + "/?token=" + url.QueryEscape(jwtToken)
	http.Redirect(w, r, frontendURL, http.StatusFound)
}

func (h *OIDCHandler) redirectWithError(w http.ResponseWriter, r *http.Request, msg string) {
	frontendURL := h.cfg.BaseURL + "/?auth_error=" + url.QueryEscape(msg)
	http.Redirect(w, r, frontendURL, http.StatusFound)
}

// --- Token Exchange ---

type oidcTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token"`
}

func (h *OIDCHandler) exchangeCode(code, codeVerifier, redirectURI string) (*oidcTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {h.cfg.TGClientID},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest("POST", telegramTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	credentials := base64.StdEncoding.EncodeToString(
		[]byte(h.cfg.TGClientID + ":" + h.cfg.TGClientSecret),
	)
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp oidcTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// --- ID Token Validation ---

type telegramIDClaims struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	PreferredUsername  string `json:"preferred_username"`
	Picture           string `json:"picture"`
	PhoneNumber       string `json:"phone_number"`
	jwt.RegisteredClaims
}

func (h *OIDCHandler) validateIDToken(rawToken string) (*telegramIDClaims, error) {
	keys, err := h.getJWKS()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	var claims telegramIDClaims
	token, err := jwt.ParseWithClaims(rawToken, &claims, func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		key, found := keys[kid]
		if !found {
			// Key not found — invalidate cache and retry once
			h.invalidateJWKSCache()
			keys, err = h.getJWKS()
			if err != nil {
				return nil, err
			}
			key, found = keys[kid]
			if !found {
				return nil, fmt.Errorf("unknown kid: %s", kid)
			}
		}

		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	if iss, _ := claims.GetIssuer(); iss != telegramIssuer {
		return nil, fmt.Errorf("invalid issuer: %s", iss)
	}

	aud, _ := claims.GetAudience()
	if !sliceContains(aud, h.cfg.TGClientID) {
		return nil, fmt.Errorf("invalid audience")
	}

	return &claims, nil
}

func sliceContains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// --- JWKS ---

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func (h *OIDCHandler) getJWKS() (map[string]interface{}, error) {
	h.jwksMu.RLock()
	if time.Now().Before(h.jwksExpiry) && len(h.jwksKeys) > 0 {
		keys := h.jwksKeys
		h.jwksMu.RUnlock()
		return keys, nil
	}
	h.jwksMu.RUnlock()

	h.jwksMu.Lock()
	defer h.jwksMu.Unlock()

	if time.Now().Before(h.jwksExpiry) && len(h.jwksKeys) > 0 {
		return h.jwksKeys, nil
	}

	resp, err := h.httpClient.Get(telegramJWKSURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}

	keys := make(map[string]interface{})
	for _, k := range jwks.Keys {
		switch k.Kty {
		case "RSA":
			pub, err := parseRSAJWK(k)
			if err != nil {
				log.Printf("OIDC: skipping RSA key %s: %v", k.Kid, err)
				continue
			}
			keys[k.Kid] = pub
		case "EC":
			pub, err := parseECJWK(k)
			if err != nil {
				log.Printf("OIDC: skipping EC key %s: %v", k.Kid, err)
				continue
			}
			keys[k.Kid] = pub
		}
	}

	h.jwksKeys = keys
	h.jwksExpiry = time.Now().Add(jwksCacheTTL)

	return keys, nil
}

func (h *OIDCHandler) invalidateJWKSCache() {
	h.jwksMu.Lock()
	h.jwksExpiry = time.Time{}
	h.jwksMu.Unlock()
}

func parseRSAJWK(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}

func parseECJWK(k jwkKey) (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve: %s", k.Crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n], nil
}
