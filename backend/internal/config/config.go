package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort        string
	JWTSecret         string
	DatabaseURL       string
	BaseURL           string
	DefaultMaxProxies int
	MetricsAddr       string // Prometheus /metrics listener; keep it off the public interface

	AdminUsername   string
	AdminPassword   string
	AdminTelegramID int64

	TelegramBotToken    string
	TelegramBotUsername string
	TelegramPayURL      string // Mini App link sent in bot messages, e.g. https://t.me/botname/pay
	// TelegramWebhookSecret must match secret_token passed to setWebhook; empty = header not checked.
	TelegramWebhookSecret string
	TGClientID            string // Telegram OIDC (BotFather → Web Login)
	TGClientSecret        string

	CryptoBotToken   string
	TonWalletAddress string

	// RollyPay (SBP). Callback URL in the terminal settings: <BASE_URL>/api/payments/sbp/webhook.
	RollyPayAPIKey        string
	RollyPaySigningSecret string
	RollyPayBaseURL       string
	RollyPayTest          bool // create sandbox payments and accept sandbox callbacks

	// 3x-ui v3 panel. VPN is disabled when XUIURL is empty.
	XUIURL        string
	XUIPathPrefix string // panel base path, e.g. "lpQXPBx6hDzvwgafKO"
	XUIAPIToken   string // Settings → Security → API Token
	XUIInboundID  int
	XUISubURL     string // subscription base, e.g. https://example.com:2096/sub/
}

func Load() *Config {
	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "3000"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://mtproxy:mtproxy@localhost:5432/mtproxy?sslmode=disable"),
		BaseURL:           getEnv("BASE_URL", ""),
		DefaultMaxProxies: getEnvInt("DEFAULT_MAX_PROXIES", 5),
		MetricsAddr:       getEnv("METRICS_ADDR", "127.0.0.1:9464"),

		AdminUsername:   getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:   getEnv("ADMIN_PASSWORD", ""),
		AdminTelegramID: int64(getEnvInt("ADMIN_TELEGRAM_ID", 0)),

		TelegramBotToken:      getEnv("TG_BOT_TOKEN", ""),
		TelegramBotUsername:   getEnv("TG_BOT_USERNAME", ""),
		TelegramPayURL:        getEnv("TG_PAY_URL", "https://t.me/staytg_bot/pay"),
		TelegramWebhookSecret: getEnv("TG_WEBHOOK_SECRET", ""),
		TGClientID:            getEnv("TG_CLIENT_ID", ""),
		TGClientSecret:        getEnv("TG_CLIENT_SECRET", ""),

		CryptoBotToken:   getEnv("CRYPTOBOT_TOKEN", ""),
		TonWalletAddress: getEnv("TON_WALLET_ADDRESS", ""),

		RollyPayAPIKey:        getEnv("ROLLYPAY_API_KEY", ""),
		RollyPaySigningSecret: getEnv("ROLLYPAY_SIGNING_SECRET", ""),
		RollyPayBaseURL:       getEnv("ROLLYPAY_BASE_URL", "https://rollypay.io"),
		RollyPayTest:          getEnv("ROLLYPAY_TEST", "") == "true",

		XUIURL:        getEnv("XUI_URL", ""),
		XUIPathPrefix: getEnv("XUI_PATH_PREFIX", ""),
		XUIAPIToken:   getEnv("XUI_API_TOKEN", ""),
		XUIInboundID:  getEnvInt("XUI_INBOUND_ID", 1),
		XUISubURL:     getEnv("XUI_SUB_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
