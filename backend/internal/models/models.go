package models

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type ProxyStatus string

const (
	StatusRunning ProxyStatus = "running"
	StatusStopped ProxyStatus = "stopped"
	StatusError   ProxyStatus = "error"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	MaxProxies   int       `json:"max_proxies"`
	TelegramID   int64     `json:"telegram_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Proxy struct {
	ID                  int64       `json:"id"`
	UserID              int64       `json:"user_id"`
	Port                int         `json:"port"`
	Domain              string      `json:"domain"`
	Secret              string      `json:"secret"`
	ContainerID         string      `json:"container_id"`
	ContainerName       string      `json:"container_name"`
	Status              ProxyStatus `json:"status"`
	CreatedAt           time.Time   `json:"created_at"`
	Link                string      `json:"link,omitempty"`
	Socks5Port          int         `json:"socks5_port,omitempty"`
	Socks5User          string      `json:"socks5_user,omitempty"`
	Socks5Pass          string      `json:"socks5_pass,omitempty"`
	Socks5ContainerID   string      `json:"socks5_container_id,omitempty"`
	Socks5ContainerName string      `json:"socks5_container_name,omitempty"`
	LinkSocks5          string      `json:"link_socks5,omitempty"`
	VlessUUID           string      `json:"vless_uuid,omitempty"`
	LinkVless           string      `json:"link_vless,omitempty"`
}

// --- Plans & Subscriptions ---

type Plan struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	DurationDays       int    `json:"duration_days"`
	Price              string `json:"price"`
	PriceLabel         string `json:"price_label"`
	PriceUSDLabel      string `json:"price_usd_label,omitempty"`
	OriginalPriceLabel string `json:"original_price_label,omitempty"`
	DiscountPercent    int    `json:"discount_percent,omitempty"`
	PerMonth           string `json:"per_month"`
	MaxProxies         int    `json:"max_proxies"`
	// Stars price in Telegram Stars (XTR); 0 = not available via Stars
	StarsPrice int `json:"stars_price,omitempty"`
	// TON amount in nanoTON (1 TON = 1_000_000_000); empty = not available via TON
	TonAmount string `json:"ton_amount,omitempty"`
}

// Цены в USD по курсу ЦБ РФ ~77 ₽/$ (февраль 2026)
// Stars: ~$0.02/star; TON: ~$5/TON → 1 nanoTON = 1e-9 TON
var Plans = []Plan{
	{
		ID: "month_1", Name: "1 месяц", DurationDays: 30,
		Price: "160.00", PriceLabel: "160 ₽", PriceUSDLabel: "~$2.10", OriginalPriceLabel: "200 ₽", DiscountPercent: 20, PerMonth: "160 ₽", MaxProxies: 1,
		StarsPrice: 160, TonAmount: "1600000000",
	},
	{
		ID: "month_3", Name: "3 месяца", DurationDays: 90,
		Price: "432.00", PriceLabel: "432 ₽", PriceUSDLabel: "~$5.60", OriginalPriceLabel: "540 ₽", DiscountPercent: 20, PerMonth: "144 ₽", MaxProxies: 3,
		StarsPrice: 400, TonAmount: "2800000000", // 2.8 TON
	},
	{
		ID: "month_6", Name: "6 месяцев", DurationDays: 180,
		Price: "768.00", PriceLabel: "768 ₽", PriceUSDLabel: "~$10", OriginalPriceLabel: "960 ₽", DiscountPercent: 20, PerMonth: "128 ₽", MaxProxies: 5,
		StarsPrice: 720, TonAmount: "5600000000", // 5.6 TON
	},
	{
		ID: "year_1", Name: "1 год", DurationDays: 365,
		Price: "1344.00", PriceLabel: "1 344 ₽", PriceUSDLabel: "~$17.50", OriginalPriceLabel: "1 680 ₽", DiscountPercent: 20, PerMonth: "112 ₽", MaxProxies: 10,
		StarsPrice: 1200, TonAmount: "13600000000",
	},
}

func GetPlan(id string) *Plan {
	for i := range Plans {
		if Plans[i].ID == id {
			return &Plans[i]
		}
	}
	return nil
}

type Subscription struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	PlanID    string    `json:"plan_id"`
	PaymentID int64     `json:"payment_id"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Payment struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	PlanID     string    `json:"plan_id"`
	ExternalID string    `json:"external_id"`
	Amount     string    `json:"amount"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}
