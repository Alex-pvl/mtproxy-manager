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
	DefaultMaxProxies int
	MTGImage          string
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
		DefaultMaxProxies: getEnvInt("DEFAULT_MAX_PROXIES", 5),
		MTGImage:          getEnv("MTG_IMAGE", "nineseconds/mtg:2"),
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
