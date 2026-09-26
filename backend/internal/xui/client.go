package xui

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	pathPrefix string
	username   string
	password   string
	apiToken   string
	subURL     string
	inboundID  int
	http       *http.Client
	mu         sync.Mutex
	inbound    *Inbound
}

type Inbound struct {
	ID             int      `json:"id"`
	Remark         string   `json:"remark"`
	Protocol       string   `json:"protocol"`
	Port           int      `json:"port"`
	Enable         bool     `json:"enable"`
	Settings       jsonText `json:"settings"`
	StreamSettings jsonText `json:"streamSettings"`
}

// jsonText holds raw JSON that x-ui sends either as a JSON-encoded string
// (3x-ui v2) or as a plain object (v3).
type jsonText string

func (t *jsonText) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*t = jsonText(s)
		return nil
	}
	*t = jsonText(b)
	return nil
}

type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

type streamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security"`
	RealitySettings *realitySettings `json:"realitySettings,omitempty"`
	TLSSettings     *tlsSettings     `json:"tlsSettings,omitempty"`
	WSSettings      *wsSettings      `json:"wsSettings,omitempty"`
	GRPCSettings    *grpcSettings    `json:"grpcSettings,omitempty"`
	XHTTPSettings   *xhttpSettings   `json:"xhttpSettings,omitempty"`
}

type realitySettings struct {
	ServerNames []string `json:"serverNames"`
	ShortIds    []string `json:"shortIds"`
	Settings    struct {
		PublicKey   string `json:"publicKey"`
		Fingerprint string `json:"fingerprint"`
		ServerName  string `json:"serverName"`
		SpiderX     string `json:"spiderX"`
	} `json:"settings"`
}

type tlsSettings struct {
	ServerName string   `json:"serverName"`
	Alpn       []string `json:"alpn,omitempty"`
}

type wsSettings struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
}

type grpcSettings struct {
	ServiceName string `json:"serviceName"`
}

type xhttpSettings struct {
	Path string `json:"path"`
	Host string `json:"host"`
	Mode string `json:"mode"`
}

type xuiClient struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	SubID      string `json:"subId"`
}

// bearerTransport adds the panel API token to every request.
type bearerTransport struct{ token string }

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

// apiToken, when set, replaces username/password login (3x-ui v3 requires a
// CSRF token for /login, Bearer requests skip it).
func NewClient(baseURL, pathPrefix, username, password, apiToken, subURL string, inboundID int) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("cookiejar: %w", err)
	}

	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		pathPrefix: strings.Trim(pathPrefix, "/"),
		username:   username,
		password:   password,
		apiToken:   apiToken,
		subURL:     subURL,
		inboundID:  inboundID,
		http: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
	}
	if apiToken != "" {
		c.http.Transport = bearerTransport{apiToken}
	}

	if err := c.login(); err != nil {
		return nil, fmt.Errorf("xui login: %w", err)
	}

	if err := c.refreshInbound(); err != nil {
		return nil, fmt.Errorf("xui fetch inbound: %w", err)
	}

	return c, nil
}

// DescribeForLog returns non-secret panel context for error logs (URLs, inbound ids, protocol).
func (c *Client) DescribeForLog() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	host := c.baseURL
	if u, err := url.Parse(c.baseURL); err == nil && u.Host != "" {
		host = u.Host
	}
	pfx := c.pathPrefix
	if pfx == "" {
		pfx = "-"
	}
	inID, inPort, inProto, inRem := 0, 0, "", ""
	if c.inbound != nil {
		inID = c.inbound.ID
		inPort = c.inbound.Port
		inProto = c.inbound.Protocol
		inRem = c.inbound.Remark
	}
	return fmt.Sprintf("xui_host=%s path_prefix=%s configured_inbound_id=%d cached_inbound={db_id:%d listen_port:%d protocol:%s remark:%q}",
		host, pfx, c.inboundID, inID, inPort, inProto, inRem)
}

func (c *Client) apiURL(path string) string {
	path = strings.TrimLeft(path, "/")
	if c.pathPrefix != "" {
		return fmt.Sprintf("%s/%s/%s", c.baseURL, c.pathPrefix, path)
	}
	return fmt.Sprintf("%s/%s", c.baseURL, path)
}

func (c *Client) login() error {
	if c.apiToken != "" {
		return nil // Bearer token is sent on every request
	}
	form := url.Values{
		"username": {c.username},
		"password": {c.password},
	}

	resp, err := c.http.PostForm(c.apiURL("login"), form)
	if err != nil {
		return fmt.Errorf("post login: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("login failed: %s", result.Msg)
	}
	return nil
}

func (c *Client) refreshInbound() error {
	path := fmt.Sprintf("panel/api/inbounds/get/%d", c.inboundID)
	body, err := c.doGet(path)
	if err != nil {
		return err
	}

	var result struct {
		Success bool    `json:"success"`
		Msg     string  `json:"msg"`
		Obj     Inbound `json:"obj"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode inbound response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("get inbound failed: %s", result.Msg)
	}

	c.mu.Lock()
	c.inbound = &result.Obj
	c.mu.Unlock()
	return nil
}

// AddClient registers a new VLESS client in the configured x-ui inbound.
// Pass a non-zero expiryTime to set when the client access expires; zero = no expiry.
func (c *Client) AddClient(uuid, email string, expiryTime time.Time) error {
	payload, _ := json.Marshal(map[string]interface{}{
		"client":     c.buildClientEntry(uuid, email, expiryTime),
		"inboundIds": []int{c.inboundID},
	})
	return c.doPost("panel/api/clients/add", payload)
}

// UpdateClientExpiry updates the expiry time of an existing VLESS client in x-ui.
// Pass a non-zero expiryTime to set a deadline; zero = remove expiry limit.
func (c *Client) UpdateClientExpiry(uuid, email string, expiryTime time.Time) error {
	payload, _ := json.Marshal(c.buildClientEntry(uuid, email, expiryTime))
	return c.doPost("panel/api/clients/update/"+url.PathEscape(email), payload)
}

// RemoveClient deletes a client (from all inbounds) by email.
func (c *Client) RemoveClient(email string) error {
	if err := c.doPost("panel/api/clients/del/"+url.PathEscape(email), nil); err != nil {
		return fmt.Errorf("delClient: %w", err)
	}
	return nil
}

func (c *Client) buildClientEntry(uuid, email string, expiryTime time.Time) xuiClient {
	entry := xuiClient{
		ID:      uuid,
		Email:   email,
		LimitIP: 1,
		Enable:  true,
		SubID:   uuid, // ponytail: vless uuid doubles as subId, nothing extra to store
	}
	if !expiryTime.IsZero() {
		entry.ExpiryTime = expiryTime.UnixMilli()
	}
	return entry
}

// UserLink returns the client's subscription URL (time-limited by the client's
// expiryTime in x-ui). Falls back to a raw vless:// link when XUI_SUB_URL is unset.
func (c *Client) UserLink(uuid, remark string) string {
	if c.subURL != "" {
		return strings.TrimRight(c.subURL, "/") + "/" + uuid
	}
	return c.BuildLink(uuid, "", remark)
}

func (c *Client) BuildLink(uuid, serverIP, remark string) string {
	c.mu.Lock()
	inbound := c.inbound
	c.mu.Unlock()

	if inbound == nil {
		return ""
	}

	host := serverIP
	if host == "" {
		if u, err := url.Parse(c.baseURL); err == nil {
			host = u.Hostname()
		}
	}

	port := inbound.Port
	params := url.Values{}
	params.Set("encryption", "none")

	var ss streamSettings
	if inbound.StreamSettings != "" {
		_ = json.Unmarshal([]byte(inbound.StreamSettings), &ss)
	}

	network := ss.Network
	if network == "" {
		network = "tcp"
	}
	params.Set("type", network)

	switch ss.Security {
	case "reality":
		params.Set("security", "reality")
		if ss.RealitySettings != nil {
			params.Set("pbk", ss.RealitySettings.Settings.PublicKey)
			fp := ss.RealitySettings.Settings.Fingerprint
			if fp == "" {
				fp = "firefox"
			}
			params.Set("fp", fp)

			sni := ss.RealitySettings.Settings.ServerName
			if sni == "" && len(ss.RealitySettings.ServerNames) > 0 {
				sni = ss.RealitySettings.ServerNames[0]
			}
			if sni != "" {
				params.Set("sni", sni)
			}

			if len(ss.RealitySettings.ShortIds) > 0 {
				params.Set("sid", ss.RealitySettings.ShortIds[0])
			}

			spx := ss.RealitySettings.Settings.SpiderX
			if spx == "" {
				spx = "/"
			}
			params.Set("spx", spx)
		}

	case "tls":
		params.Set("security", "tls")
		if ss.TLSSettings != nil && ss.TLSSettings.ServerName != "" {
			params.Set("sni", ss.TLSSettings.ServerName)
		}
	}

	// Transport-specific parameters
	switch network {
	case "ws":
		if ss.WSSettings != nil {
			if ss.WSSettings.Path != "" {
				params.Set("path", ss.WSSettings.Path)
			}
			if h, ok := ss.WSSettings.Headers["Host"]; ok && h != "" {
				params.Set("host", h)
			}
		}
	case "grpc":
		if ss.GRPCSettings != nil && ss.GRPCSettings.ServiceName != "" {
			params.Set("serviceName", ss.GRPCSettings.ServiceName)
		}
	case "xhttp":
		if ss.XHTTPSettings != nil {
			if ss.XHTTPSettings.Path != "" {
				params.Set("path", ss.XHTTPSettings.Path)
			}
			if ss.XHTTPSettings.Host != "" {
				params.Set("host", ss.XHTTPSettings.Host)
			}
			if ss.XHTTPSettings.Mode != "" {
				params.Set("mode", ss.XHTTPSettings.Mode)
			}
		}
	}

	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		uuid, host, port, params.Encode(), url.QueryEscape(remark))
}

func (c *Client) doGet(path string) ([]byte, error) {
	resp, err := c.http.Get(c.apiURL(path))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if err := c.login(); err != nil {
			return nil, err
		}
		resp2, err := c.http.Get(c.apiURL(path))
		if err != nil {
			return nil, err
		}
		defer resp2.Body.Close()
		body, _ = io.ReadAll(resp2.Body)
	}
	return body, nil
}

func (c *Client) doPost(path string, body []byte) error {
	do := func() (*http.Response, error) {
		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}
		req, err := http.NewRequest(http.MethodPost, c.apiURL(path), reqBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return c.http.Do(req)
	}

	resp, err := do()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	status := resp.StatusCode

	// 3x-ui checkAPIAuth returns 404 with an empty body when the session cookie
	// is missing or invalid (to hide API existence). Treat like auth failure.
	needsReauth := status == http.StatusUnauthorized || status == http.StatusForbidden
	if status == http.StatusNotFound && len(bytes.TrimSpace(respBody)) == 0 &&
		strings.Contains(path, "panel/api") {
		needsReauth = true
	}

	if needsReauth {
		if err := c.login(); err != nil {
			return err
		}
		resp2, err := do()
		if err != nil {
			return err
		}
		defer resp2.Body.Close()
		respBody, _ = io.ReadAll(resp2.Body)
		status = resp2.StatusCode
	}

	return checkAPISuccess(status, respBody)
}

func checkAPISuccess(statusCode int, body []byte) error {
	body = bytes.TrimSpace(body)
	if statusCode < 200 || statusCode >= 300 {
		if len(body) == 0 {
			return fmt.Errorf("x-ui HTTP %d with empty body (wrong URL, proxy error, or panel down); check XUI_URL and XUI_PATH_PREFIX", statusCode)
		}
		var result apiResponse
		if err := json.Unmarshal(body, &result); err == nil && result.Msg != "" {
			return fmt.Errorf("x-ui HTTP %d: %s", statusCode, result.Msg)
		}
		snip := string(body)
		if len(snip) > 240 {
			snip = snip[:240] + "…"
		}
		return fmt.Errorf("x-ui HTTP %d (non-JSON or error page): %q", statusCode, snip)
	}
	if len(body) == 0 {
		return fmt.Errorf("x-ui returned HTTP %d with empty body — panel usually responds with JSON; check XUI_URL, XUI_PATH_PREFIX, reverse proxy buffering, and that addClient route exists", statusCode)
	}
	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		snip := string(body)
		if len(snip) > 240 {
			snip = snip[:240] + "…"
		}
		return fmt.Errorf("decode x-ui response (HTTP %d): %w; body starts with: %q", statusCode, err, snip)
	}
	if !result.Success {
		return fmt.Errorf("api error: %s", result.Msg)
	}
	return nil
}

func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant RFC 4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
