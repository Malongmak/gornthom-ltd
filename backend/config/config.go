package config

import (
	"os"
)

func resolveFrontendDir() string {
	// Prefer a local frontend/ copy (allows per-branch overrides)
	if _, err := os.Stat("frontend"); err == nil {
		return "frontend"
	}
	return "../frontend"
}

type RouterConfig struct {
	Host     string
	Username string
	Password string
	Port     string
}

type Config struct {
	Port                string
	NodeEnv             string
	RouterType          string
	ServerIP            string
	FrontendDir         string
	PaystackKey         string
	MpesaEnv            string
	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaShortcode      string
	MpesaPasskey        string
	MpesaCallbackURL    string
	MikroTik            RouterConfig
	OpenWrt             RouterConfig
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "3000"),
		NodeEnv:             getEnv("NODE_ENV", "development"),
		RouterType:          getEnv("ROUTER_TYPE", "mikrotik"),
		ServerIP:            getEnv("SERVER_IP", "localhost"),
		FrontendDir:         getEnv("FRONTEND_DIR", resolveFrontendDir()),
		PaystackKey:         getEnv("PAYSTACK_SECRET_KEY", ""),
		MpesaEnv:            getEnv("MPESA_ENV", "sandbox"),
		MpesaConsumerKey:    getEnv("MPESA_CONSUMER_KEY", ""),
		MpesaConsumerSecret: getEnv("MPESA_CONSUMER_SECRET", ""),
		MpesaShortcode:      getEnv("MPESA_SHORTCODE", ""),
		MpesaPasskey:        getEnv("MPESA_PASSKEY", ""),
		MpesaCallbackURL:    getEnv("MPESA_CALLBACK_URL", ""),
		MikroTik: RouterConfig{
			Host:     getEnv("MIKROTIK_HOST", "192.168.88.1"),
			Username: getEnv("MIKROTIK_USER", "admin"),
			Password: getEnv("MIKROTIK_PASSWORD", ""),
			Port:     getEnv("MIKROTIK_PORT", "8728"),
		},
		OpenWrt: RouterConfig{
			Host:     getEnv("OPENWRT_HOST", "192.168.1.1"),
			Username: getEnv("OPENWRT_USER", "root"),
			Password: getEnv("OPENWRT_PASSWORD", ""),
			Port:     getEnv("OPENWRT_SSH_PORT", "22"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
