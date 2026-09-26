package handlers

import (
	"errors"
	"fmt"
	"log"
	"time"

	"mtproxy-manager/internal/database"
	"mtproxy-manager/internal/models"
	"mtproxy-manager/internal/xui"
)

var errVPNUnavailable = errors.New("vpn panel is not configured")

// VPN keeps VPN records in the database and their clients in 3x-ui in step.
type VPN struct {
	db  *database.DB
	xui *xui.Client // nil when the panel is not configured or unreachable at startup
}

func NewVPN(db *database.DB, xuiClient *xui.Client) *VPN {
	return &VPN{db: db, xui: xuiClient}
}

// Create adds a client that expires together with the user's subscription.
func (v *VPN) Create(user *models.User) (*models.Proxy, error) {
	if v.xui == nil {
		return nil, errVPNUnavailable
	}
	existing, err := v.db.ListProxiesByUser(user.ID)
	if err != nil {
		return nil, err
	}
	uuid, err := xui.GenerateUUID()
	if err != nil {
		return nil, err
	}
	var expiry time.Time
	if sub, _ := v.db.GetActiveSubscription(user.ID); sub != nil {
		expiry = sub.ExpiresAt
	}
	p := &models.Proxy{UserID: user.ID, VlessUUID: uuid, VlessEmail: nextClientName(user.Username, existing)}
	if err := v.xui.AddClient(uuid, p.VlessEmail, expiry); err != nil {
		return nil, fmt.Errorf("x-ui add client %s: %w", p.VlessEmail, err)
	}
	if err := v.db.CreateProxy(p); err != nil {
		if rmErr := v.xui.RemoveClient(p.VlessEmail); rmErr != nil {
			log.Printf("vpn: orphaned x-ui client %s: %v", p.VlessEmail, rmErr)
		}
		return nil, err
	}
	v.fillLink(p)
	return p, nil
}

// Delete removes the x-ui client first so a failure leaves nothing orphaned.
func (v *VPN) Delete(p *models.Proxy) error {
	if v.xui == nil {
		return errVPNUnavailable
	}
	if err := v.xui.RemoveClient(p.VlessEmail); err != nil {
		return fmt.Errorf("x-ui remove client %s: %w", p.VlessEmail, err)
	}
	return v.db.DeleteProxy(p.ID)
}

// DeleteAllOf removes every VPN of a user; used before deleting the user.
func (v *VPN) DeleteAllOf(userID int64) error {
	proxies, err := v.db.ListProxiesByUser(userID)
	if err != nil {
		return err
	}
	for i := range proxies {
		if err := v.Delete(&proxies[i]); err != nil {
			return err
		}
	}
	return nil
}

// SyncExpiry pushes the subscription end date to all of the user's clients.
func (v *VPN) SyncExpiry(userID int64, expiresAt time.Time) {
	if v.xui == nil {
		return
	}
	proxies, err := v.db.ListProxiesByUser(userID)
	if err != nil {
		log.Printf("vpn: sync expiry user=%d: %v", userID, err)
		return
	}
	for _, p := range proxies {
		if err := v.xui.UpdateClientExpiry(p.VlessUUID, p.VlessEmail, expiresAt); err != nil {
			log.Printf("vpn: sync expiry %s: %v", p.VlessEmail, err)
		}
	}
}

func (v *VPN) List(userID int64) ([]models.Proxy, error) {
	proxies, err := v.db.ListProxiesByUser(userID)
	for i := range proxies {
		v.fillLink(&proxies[i])
	}
	return proxies, err
}

func (v *VPN) fillLink(p *models.Proxy) {
	if v.xui != nil {
		p.LinkSub = v.xui.SubscriptionURL(p.VlessUUID)
	}
}

// nextClientName picks "staytg.org-<username>-<n>" with the smallest free n.
func nextClientName(username string, existing []models.Proxy) string {
	taken := make(map[string]bool, len(existing))
	for _, p := range existing {
		taken[p.VlessEmail] = true
	}
	for n := 1; ; n++ {
		if name := fmt.Sprintf("staytg.org-%s-%d", username, n); !taken[name] {
			return name
		}
	}
}
