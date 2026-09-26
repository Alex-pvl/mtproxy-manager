// Package xui talks to a 3x-ui v3 panel via its API token.
package xui

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL   string // panel URL including base path, no trailing slash
	apiToken  string
	subURL    string // subscription base, no trailing slash
	inboundID int
	http      *http.Client
}

type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

type client struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	LimitIP    int    `json:"limitIp"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	SubID      string `json:"subId"`
}

// NewClient checks that the panel answers and the inbound exists.
func NewClient(panelURL, pathPrefix, apiToken, subURL string, inboundID int) (*Client, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("XUI_API_TOKEN is required")
	}
	if subURL == "" {
		return nil, fmt.Errorf("XUI_SUB_URL is required")
	}
	base := strings.TrimRight(panelURL, "/")
	if p := strings.Trim(pathPrefix, "/"); p != "" {
		base += "/" + p
	}
	c := &Client{
		baseURL:   base,
		apiToken:  apiToken,
		subURL:    strings.TrimRight(subURL, "/"),
		inboundID: inboundID,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
	if _, err := c.call(http.MethodGet, fmt.Sprintf("panel/api/inbounds/get/%d", inboundID), nil); err != nil {
		return nil, fmt.Errorf("get inbound %d: %w", inboundID, err)
	}
	return c, nil
}

// SubscriptionURL is what users import into their VPN app. It stops serving
// configs once the client's expiryTime passes.
func (c *Client) SubscriptionURL(uuid string) string {
	return c.subURL + "/" + uuid
}

// AddClient creates a VLESS client; the uuid doubles as its subscription id.
func (c *Client) AddClient(uuid, email string, expiry time.Time) error {
	_, err := c.call(http.MethodPost, "panel/api/clients/add", map[string]any{
		"client":     newClient(uuid, email, expiry),
		"inboundIds": []int{c.inboundID},
	})
	return err
}

// UpdateClientExpiry moves the client's expiry; zero time = no expiry.
func (c *Client) UpdateClientExpiry(uuid, email string, expiry time.Time) error {
	_, err := c.call(http.MethodPost, "panel/api/clients/update/"+url.PathEscape(email), newClient(uuid, email, expiry))
	return err
}

func (c *Client) RemoveClient(email string) error {
	_, err := c.call(http.MethodPost, "panel/api/clients/del/"+url.PathEscape(email), nil)
	return err
}

func newClient(uuid, email string, expiry time.Time) client {
	cl := client{ID: uuid, Email: email, LimitIP: 1, Enable: true, SubID: uuid}
	if !expiry.IsZero() {
		cl.ExpiryTime = expiry.UnixMilli()
	}
	return cl
}

func (c *Client) call(method, path string, body any) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+"/"+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		// 3x-ui answers 404 with an empty body when auth fails.
		return nil, fmt.Errorf("%s %s: HTTP %d, non-JSON body %q (check XUI_URL, XUI_PATH_PREFIX, XUI_API_TOKEN)",
			method, path, resp.StatusCode, truncate(respBody, 200))
	}
	if !result.Success {
		return nil, fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, result.Msg)
	}
	return result.Obj, nil
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}

func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
