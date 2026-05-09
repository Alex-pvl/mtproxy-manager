// Package fragment is a thin HTTP client for the fragment-worker Python sidecar
// which handles Telegram Stars / Premium purchases via Fragment.
package fragment

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Available reports whether the worker is configured (URL set).
func (c *Client) Available() bool {
	return c.baseURL != ""
}

type Quote struct {
	// TonCostNano is the actual TON cost on Fragment (without markup), in nanoTON.
	TonCostNano int64 `json:"ton_cost_nano"`
	// TonRubRate — RUB per 1 TON, as quoted by the worker.
	TonRubRate float64 `json:"ton_rub_rate"`
	// TonUsdRate — USD per 1 TON.
	TonUsdRate float64 `json:"ton_usd_rate"`
}

type Balance struct {
	TonBalanceNano int64 `json:"ton_balance_nano"`
}

type UsernameCheck struct {
	OK          bool   `json:"ok"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type PurchaseRequest struct {
	OrderID   int64  `json:"order_id"`
	Type      string `json:"type"` // "stars" | "premium"
	Recipient string `json:"recipient"`
	Quantity  int    `json:"quantity"` // stars amount, or months
}

type PurchaseResult struct {
	Status string `json:"status"` // "delivered" | "processing" | "failed"
	TxHash string `json:"tx_hash,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (c *Client) doJSON(method, path string, body, out interface{}) error {
	if c.baseURL == "" {
		return errors.New("fragment worker not configured")
	}
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(respBody, &e)
		if e.Error == "" {
			e.Error = string(respBody)
		}
		return fmt.Errorf("fragment worker %d: %s", resp.StatusCode, e.Error)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

// Quote returns the current TON cost for a given product. Fragment requires a
// real recipient to compute price, so callers must pass one.
func (c *Client) Quote(productType string, quantity int, recipient string) (*Quote, error) {
	q := &Quote{}
	path := fmt.Sprintf("/quote?type=%s&quantity=%d&recipient=%s",
		productType, quantity, strings.TrimPrefix(strings.TrimSpace(recipient), "@"))
	if err := c.doJSON("GET", path, nil, q); err != nil {
		return nil, err
	}
	return q, nil
}

// Balance returns the worker wallet's available TON balance.
func (c *Client) Balance() (*Balance, error) {
	b := &Balance{}
	if err := c.doJSON("GET", "/balance", nil, b); err != nil {
		return nil, err
	}
	return b, nil
}

// CheckUsername validates a Telegram username can receive Stars/Premium and
// returns display name + avatar URL when Fragment knows the user.
func (c *Client) CheckUsername(username, productType string) (*UsernameCheck, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if productType == "" {
		productType = "stars"
	}
	out := &UsernameCheck{}
	path := fmt.Sprintf("/username/check?u=%s&type=%s", username, productType)
	if err := c.doJSON("GET", path, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Purchase delivers a Stars/Premium order. The worker may respond synchronously
// (status="delivered") or accept and start processing asynchronously.
func (c *Client) Purchase(req PurchaseRequest) (*PurchaseResult, error) {
	out := &PurchaseResult{}
	if err := c.doJSON("POST", "/purchase", req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// OrderStatus polls the worker for an in-flight order status.
func (c *Client) OrderStatus(orderID int64) (*PurchaseResult, error) {
	out := &PurchaseResult{}
	if err := c.doJSON("GET", fmt.Sprintf("/order/%d", orderID), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
