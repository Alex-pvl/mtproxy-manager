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
	VlessEmail          string      `json:"-"` // x-ui client email; empty on legacy rows
	LinkSub             string      `json:"link_sub,omitempty"` // subscription URL, or vless:// if XUI_SUB_URL unset
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
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	PlanID      string    `json:"plan_id"`
	ExternalID  string    `json:"external_id"`
	Amount      string    `json:"amount"`
	Status      string    `json:"status"`
	ProductType string    `json:"product_type"`
	Metadata    string    `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// --- Products: Stars & Premium ---

type ProductType string

const (
	ProductVPN     ProductType = "vpn"
	ProductStars   ProductType = "stars"
	ProductPremium ProductType = "premium"
)

// FragmentOrder tracks a Stars/Premium delivery via Fragment worker.
type FragmentOrder struct {
	ID                int64     `json:"id"`
	PaymentID         int64     `json:"payment_id"`
	UserID            int64     `json:"user_id"`
	Type              string    `json:"type"` // "stars" | "premium"
	RecipientUsername string    `json:"recipient_username"`
	Quantity          int       `json:"quantity"` // stars amount, or months for premium
	Status            string    `json:"status"`   // "pending"|"processing"|"delivered"|"failed"
	FragmentTxHash    string    `json:"fragment_tx_hash,omitempty"`
	Error             string    `json:"error,omitempty"`
	Attempts          int       `json:"attempts"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// StarsPackages — preset stars amounts shown on the showcase.
// Final price is computed dynamically from Fragment quote + markup.
var StarsPackages = []int{50, 100, 250, 500, 1000, 2500, 5000}

// PremiumMonthsOptions — supported gift periods for Telegram Premium.
var PremiumMonthsOptions = []int{3, 6, 12}
