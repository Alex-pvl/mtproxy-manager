package models

import "time"

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
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

// Proxy is a user's VPN config: one 3x-ui client. The name is historical.
type Proxy struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	VlessUUID  string    `json:"-"`
	VlessEmail string    `json:"name"`               // x-ui client email, e.g. staytg.org-alice-1
	LinkSub    string    `json:"link_sub,omitempty"` // subscription URL
	CreatedAt  time.Time `json:"created_at"`
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

// Цены в USD по курсу ЦБ РФ ~78 ₽/$ (июль 2026)
// Stars: ~$0.02/star; GRAM (ex-TON): ~$1.56, ₽/$ ~78 → ~121.7 ₽/GRAM (01.07.2026)
// TonAmount в nanoGRAM (1 GRAM = 1e9); суммы = рублёвая цена ÷ 121.7
var Plans = []Plan{
	{
		ID: "month_1", Name: "1 месяц", DurationDays: 30,
		Price: "200.00", PriceLabel: "200 ₽", PriceUSDLabel: "~$2.60", PerMonth: "200 ₽", MaxProxies: 1,
		StarsPrice: 200, TonAmount: "1640000000", // 1.64 GRAM (~200 ₽)
	},
	{
		ID: "month_3", Name: "3 месяца", DurationDays: 90,
		Price: "540.00", PriceLabel: "540 ₽", PriceUSDLabel: "~$7", PerMonth: "180 ₽", MaxProxies: 3,
		StarsPrice: 500, TonAmount: "4440000000", // 4.44 GRAM (~540 ₽)
	},
	{
		ID: "month_6", Name: "6 месяцев", DurationDays: 180,
		Price: "960.00", PriceLabel: "960 ₽", PriceUSDLabel: "~$12.50", PerMonth: "160 ₽", MaxProxies: 5,
		StarsPrice: 900, TonAmount: "7890000000", // 7.89 GRAM (~960 ₽)
	},
	{
		ID: "year_1", Name: "1 год", DurationDays: 365,
		Price: "1680.00", PriceLabel: "1 680 ₽", PriceUSDLabel: "~$21.80", PerMonth: "140 ₽", MaxProxies: 10,
		StarsPrice: 1500, TonAmount: "13810000000", // 13.81 GRAM (~1680 ₽)
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
	ExternalID string    `json:"external_id"` // provider id, prefixed: sbp_, stars_, stay_ (TON), bare = CryptoBot
	Amount     string    `json:"amount"`
	Status     string    `json:"status"` // pending | paid | canceled
	CreatedAt  time.Time `json:"created_at"`
}
