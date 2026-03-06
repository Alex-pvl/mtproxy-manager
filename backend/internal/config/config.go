package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort        string
	JWTSecret         string
	DatabaseURL       string
	PortMin           int
	PortMax           int
	Socks5PortMin     int
	Socks5PortMax     int
	DefaultMaxProxies int
	MTGImage          string
	GostImage         string
	ServerIP          string
	CryptoBotToken    string
	BaseURL           string
	AdminUsername     string
	AdminTelegramID   int64
	TelegramBotToken  string
	TGClientID        string
	TGClientSecret    string

	// x-ui / 3x-ui panel integration for VLESS link generation
	XUIEnabled    bool
	XUIURL        string
	XUIPathPrefix string // custom panel base path (e.g. "vwtLfHqxkCntctQ"), empty = default
	XUIUsername   string
	XUIPassword   string
	XUIInboundID  int
}

func Load() *Config {
	xuiURL := getEnv("XUI_URL", "")
	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "3000"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://mtproxy:mtproxy@localhost:5432/mtproxy?sslmode=disable"),
		PortMin:           getEnvInt("PORT_MIN", 8000),
		PortMax:           getEnvInt("PORT_MAX", 9000),
		Socks5PortMin:     getEnvInt("SOCKS5_PORT_MIN", 10000),
		Socks5PortMax:     getEnvInt("SOCKS5_PORT_MAX", 10999),
		DefaultMaxProxies: getEnvInt("DEFAULT_MAX_PROXIES", 5),
		MTGImage:          getEnv("MTG_IMAGE", "nineseconds/mtg:2"),
		GostImage:         getEnv("GOST_IMAGE", "ginuerzh/gost:2.12"),
		ServerIP:          getEnv("SERVER_IP", ""),
		CryptoBotToken:    getEnv("CRYPTOBOT_TOKEN", ""),
		BaseURL:           getEnv("BASE_URL", ""),
		AdminUsername:     getEnv("ADMIN_USERNAME", "admin"),
		AdminTelegramID:   int64(getEnvInt("ADMIN_TELEGRAM_ID", 0)),
		TelegramBotToken:  getEnv("TG_BOT_TOKEN", ""),
		TGClientID:        getEnv("TG_CLIENT_ID", ""),
		TGClientSecret:    getEnv("TG_CLIENT_SECRET", ""),
		XUIEnabled:        xuiURL != "",
		XUIURL:            xuiURL,
		XUIPathPrefix:     getEnv("XUI_PATH_PREFIX", ""),
		XUIUsername:       getEnv("XUI_USERNAME", "admin"),
		XUIPassword:       getEnv("XUI_PASSWORD", ""),
		XUIInboundID:      getEnvInt("XUI_INBOUND_ID", 1),
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
