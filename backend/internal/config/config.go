package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort           string
	JWTSecret            string
	DatabaseURL          string
	PortMin              int
	PortMax              int
	Socks5PortMin        int
	Socks5PortMax        int
	DefaultMaxProxies    int
	MTGImage             string
	GostImage            string
	ServerIP             string
	CryptoBotToken       string
	DigitalPayAPIKey     string
	DigitalPayBaseURL    string
	DigitalPaySBPBackURL string
	BaseURL              string
	AdminUsername        string
	AdminPassword        string
	AdminTelegramID      int64
	TelegramBotToken     string
	TelegramBotUsername  string
	// TelegramPayURL — ссылка на оплату (например Mini App: https://t.me/botname/pay)
	TelegramPayURL string
	TGClientID     string
	TGClientSecret string

	// TON wallet address for direct TON payments
	TonWalletAddress string

	// Fragment worker (Python sidecar) for Stars / Premium fulfillment
	FragmentWorkerURL   string
	FragmentWorkerToken string
	StarsMarkupPct      int
	PremiumMarkupPct    int
	// Minimum TON balance (in nanoTON) below which orders are blocked.
	MinTonBalanceNano int64

	// x-ui / 3x-ui panel integration for VLESS link generation
	XUIEnabled    bool
	XUIURL        string
	XUIPathPrefix string // custom panel base path (e.g. "vwtLfHqxkCntctQ"), empty = default
	XUIUsername   string
	XUIPassword   string
	XUIAPIToken   string // 3x-ui Settings → Security → API Token; replaces login when set
	XUIInboundID  int
	XUISubURL     string // 3x-ui subscription base, e.g. https://tagwaiter.ru:2096/sub/
}

func Load() *Config {
	xuiURL := getEnv("XUI_URL", "")
	return &Config{
		ServerPort:           getEnv("SERVER_PORT", "3000"),
		JWTSecret:            getEnv("JWT_SECRET", "change-me-in-production"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://mtproxy:mtproxy@localhost:5432/mtproxy?sslmode=disable"),
		PortMin:              getEnvInt("PORT_MIN", 8000),
		PortMax:              getEnvInt("PORT_MAX", 9000),
		Socks5PortMin:        getEnvInt("SOCKS5_PORT_MIN", 10000),
		Socks5PortMax:        getEnvInt("SOCKS5_PORT_MAX", 10999),
		DefaultMaxProxies:    getEnvInt("DEFAULT_MAX_PROXIES", 5),
		MTGImage:             getEnv("MTG_IMAGE", "nineseconds/mtg:2"),
		GostImage:            getEnv("GOST_IMAGE", "ginuerzh/gost:2.12"),
		ServerIP:             getEnv("SERVER_IP", ""),
		CryptoBotToken:       getEnv("CRYPTOBOT_TOKEN", ""),
		DigitalPayAPIKey:     getEnv("DIGITALPAY_API_KEY", ""),
		DigitalPayBaseURL:    getEnv("DIGITALPAY_BASE_URL", "https://digitalpay.cc"),
		DigitalPaySBPBackURL: getEnv("DIGITALPAY_SBP_BACK_URL", ""),
		BaseURL:              getEnv("BASE_URL", ""),
		AdminUsername:        getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:        getEnv("ADMIN_PASSWORD", ""),
		AdminTelegramID:      int64(getEnvInt("ADMIN_TELEGRAM_ID", 0)),
		TelegramBotToken:     getEnv("TG_BOT_TOKEN", ""),
		TelegramBotUsername:  getEnv("TG_BOT_USERNAME", ""),
		TelegramPayURL:       getEnv("TG_PAY_URL", "https://t.me/staytg_bot/pay"),
		TGClientID:           getEnv("TG_CLIENT_ID", ""),
		TGClientSecret:       getEnv("TG_CLIENT_SECRET", ""),
		TonWalletAddress:     getEnv("TON_WALLET_ADDRESS", ""),
		FragmentWorkerURL:    getEnv("FRAGMENT_WORKER_URL", ""),
		FragmentWorkerToken:  getEnv("FRAGMENT_WORKER_TOKEN", ""),
		StarsMarkupPct:       getEnvInt("STARS_MARKUP_PCT", 15),
		PremiumMarkupPct:     getEnvInt("PREMIUM_MARKUP_PCT", 15),
		MinTonBalanceNano:    int64(getEnvInt("MIN_TON_BALANCE_NANO", 500000000)),
		XUIEnabled:           xuiURL != "",
		XUIURL:               xuiURL,
		XUIPathPrefix:        getEnv("XUI_PATH_PREFIX", ""),
		XUIUsername:          getEnv("XUI_USERNAME", "admin"),
		XUIPassword:          getEnv("XUI_PASSWORD", ""),
		XUIAPIToken:          getEnv("XUI_API_TOKEN", ""),
		XUIInboundID:         getEnvInt("XUI_INBOUND_ID", 1),
		XUISubURL:            getEnv("XUI_SUB_URL", ""),
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
