package handlers

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/config"
)

type SettingsHandler struct {
	cfg     *config.Config
	envPath string
}

func NewSettingsHandler(cfg *config.Config) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, envPath: ".env"}
}

// GET /api/settings — returns non-sensitive config values
func (h *SettingsHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"server": gin.H{
			"port":        h.cfg.Port,
			"serverIP":    h.cfg.ServerIP,
			"nodeEnv":     h.cfg.NodeEnv,
			"routerType":  h.cfg.RouterType,
			"frontendDir": h.cfg.FrontendDir,
		},
		"router": gin.H{
			"mikrotikHost": h.cfg.MikroTik.Host,
			"mikrotikUser": h.cfg.MikroTik.Username,
			"mikrotikPort": h.cfg.MikroTik.Port,
			"openwrtHost":  h.cfg.OpenWrt.Host,
			"openwrtUser":  h.cfg.OpenWrt.Username,
			"openwrtPort":  h.cfg.OpenWrt.Port,
		},
		"payment": gin.H{
			"paystackPublicKey": os.Getenv("PAYSTACK_PUBLIC_KEY"),
			// secret key intentionally omitted from GET
		},
		"business": gin.H{
			"businessPhone": os.Getenv("BUSINESS_PHONE"),
			"businessEmail": os.Getenv("BUSINESS_EMAIL"),
		},
	})
}

// PATCH /api/settings — writes changed keys to .env
func (h *SettingsHandler) Update(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Map frontend keys → .env keys
	keyMap := map[string]string{
		"PORT":                  "PORT",
		"SERVER_IP":             "SERVER_IP",
		"NODE_ENV":              "NODE_ENV",
		"ROUTER_TYPE":           "ROUTER_TYPE",
		"FRONTEND_DIR":          "FRONTEND_DIR",
		"MIKROTIK_HOST":         "MIKROTIK_HOST",
		"MIKROTIK_USER":         "MIKROTIK_USER",
		"MIKROTIK_PASSWORD":     "MIKROTIK_PASSWORD",
		"MIKROTIK_PORT":         "MIKROTIK_PORT",
		"OPENWRT_HOST":          "OPENWRT_HOST",
		"OPENWRT_USER":          "OPENWRT_USER",
		"OPENWRT_PASSWORD":      "OPENWRT_PASSWORD",
		"OPENWRT_SSH_PORT":      "OPENWRT_SSH_PORT",
		"PAYSTACK_SECRET_KEY":   "PAYSTACK_SECRET_KEY",
		"PAYSTACK_PUBLIC_KEY":   "PAYSTACK_PUBLIC_KEY",
		"MPESA_ENV":             "MPESA_ENV",
		"MPESA_CONSUMER_KEY":    "MPESA_CONSUMER_KEY",
		"MPESA_CONSUMER_SECRET": "MPESA_CONSUMER_SECRET",
		"MPESA_SHORTCODE":       "MPESA_SHORTCODE",
		"MPESA_PASSKEY":         "MPESA_PASSKEY",
		"MPESA_CALLBACK_URL":    "MPESA_CALLBACK_URL",
		"BUSINESS_PHONE":        "BUSINESS_PHONE",
		"BUSINESS_EMAIL":        "BUSINESS_EMAIL",
	}

	// Only allow keys in the whitelist
	updates := map[string]string{}
	for k, v := range body {
		if envKey, ok := keyMap[k]; ok {
			updates[envKey] = v
		}
	}

	if err := updateEnvFile(h.envPath, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Settings saved. Restart the server to apply changes.",
	})
}

// updateEnvFile reads the .env, updates matching keys, writes back
func updateEnvFile(path string, updates map[string]string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	var lines []string
	updated := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// skip comments and blanks
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if val, ok := updates[key]; ok {
				lines = append(lines, key+"="+val)
				updated[key] = true
				continue
			}
		}
		lines = append(lines, line)
	}
	f.Close()

	// Append any keys not already in the file
	for k, v := range updates {
		if !updated[k] {
			lines = append(lines, k+"="+v)
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
