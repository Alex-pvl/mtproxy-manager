package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort        string
	JWTSecret         string
	DBPath            string
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
	AdminPassword     string
}

func Load() *Config {
	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		DBPath:            getEnv("DB_PATH", "./data/mtproxy.db"),
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
		AdminPassword:     getEnv("ADMIN_PASSWORD", ""),
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
