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
	CreatedAt    time.Time `json:"created_at"`
}

type Proxy struct {
	ID            int64       `json:"id"`
	UserID        int64       `json:"user_id"`
	Port          int         `json:"port"`
	Domain        string      `json:"domain"`
	Secret        string      `json:"secret"`
	ContainerID   string      `json:"container_id"`
	ContainerName string      `json:"container_name"`
	Status        ProxyStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	Link          string      `json:"link,omitempty"`
}

// --- Plans & Subscriptions ---

type Plan struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DurationDays int    `json:"duration_days"`
	Price        string `json:"price"`
	PriceLabel   string `json:"price_label"`
	PerMonth     string `json:"per_month"`
	MaxProxies   int    `json:"max_proxies"`
}

var Plans = []Plan{
	{ID: "month_1", Name: "1 месяц", DurationDays: 30, Price: "190.00", PriceLabel: "190 ₽", PerMonth: "190 ₽", MaxProxies: 5},
	{ID: "month_3", Name: "3 месяца", DurationDays: 90, Price: "490.00", PriceLabel: "490 ₽", PerMonth: "163 ₽", MaxProxies: 5},
	{ID: "month_6", Name: "6 месяцев", DurationDays: 180, Price: "990.00", PriceLabel: "990 ₽", PerMonth: "165 ₽", MaxProxies: 5},
	{ID: "year_1", Name: "1 год", DurationDays: 365, Price: "1990.00", PriceLabel: "1 990 ₽", PerMonth: "166 ₽", MaxProxies: 5},
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
