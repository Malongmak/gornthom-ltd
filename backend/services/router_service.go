package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gornhom/backend/config"
	"github.com/gornhom/backend/db"
	"github.com/gornhom/backend/models"
)

type ConnectionEntry struct {
	Token            string
	PhoneNumber      string
	PackageName      string
	PackagePrice     float64
	UserIP           string
	MacAddress       string
	StartTime        string
	ExpiryTime       string
	DurationMinutes  int
	TransactionID    string
	ConnectedDevices []string
	MaxDevices       int
}

// packageMaxDevices maps package names to their device limits
var packageMaxDevices = map[string]int{
	"30 Minutes":      1,
	"1 Hour":          1,
	"1 Day":           2,
	"1 Week":          3,
	"1 Month":         5,
	"Enterprise Plan": 25,
}

func maxDevicesForPackage(name string) int {
	if n, ok := packageMaxDevices[name]; ok {
		return n
	}
	return 1
}

type RouterService struct {
	cfg               *config.Config
	mu                sync.RWMutex
	activeConnections map[string]*ConnectionEntry // keyed by token
}

func NewRouterService(cfg *config.Config) *RouterService {
	rs := &RouterService{
		cfg:               cfg,
		activeConnections: make(map[string]*ConnectionEntry),
	}
	rs.loadFromDB()
	return rs
}

func (rs *RouterService) loadFromDB() {
	sessions, err := db.GetAllActive()
	if err != nil {
		log.Printf("Could not load sessions from DB: %v", err)
		return
	}
	for _, s := range sessions {
		entry := &ConnectionEntry{
			Token:           s.Token,
			PhoneNumber:     s.Phone,
			PackageName:     s.Package,
			PackagePrice:    s.Price,
			UserIP:          s.UserIP,
			MacAddress:      s.MacAddress,
			StartTime:       s.StartTime,
			ExpiryTime:      s.ExpiryTime,
			DurationMinutes: s.Duration,
			TransactionID:   s.TxnID,
			MaxDevices:      1,
		}
		rs.activeConnections[s.Token] = entry
		rs.scheduleExpiry(s.Token, s.ExpiryTime)
	}
	if len(sessions) > 0 {
		log.Printf("📦 Restored %d active session(s) from database", len(sessions))
	}
}

func (rs *RouterService) ActivateConnection(data *models.ActivationData) *models.ActivationResult {
	switch rs.cfg.RouterType {
	case "mikrotik":
		return rs.activateMikroTik(data)
	default:
		return rs.activateGeneric(data)
	}
}

func (rs *RouterService) activateMikroTik(data *models.ActivationData) *models.ActivationResult {
	token := fmt.Sprintf("mikrotik_%s", data.TransactionID)
	log.Printf("📡 MikroTik: would whitelist MAC:%s IP:%s for %dm", data.MacAddress, data.UserIP, data.DurationMinutes)
	// In production: use a MikroTik API library or SSH to add firewall address-list entries
	rs.storeConnection(data, token)
	return &models.ActivationResult{
		Success:   true,
		Token:     token,
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixMilli()),
		Message:   "Connection activated",
	}
}

func (rs *RouterService) activateGeneric(data *models.ActivationData) *models.ActivationResult {
	token := fmt.Sprintf("generic_%s", data.TransactionID)
	log.Printf("📝 Generic: whitelist IP:%s MAC:%s for %dm", data.UserIP, data.MacAddress, data.DurationMinutes)
	rs.storeConnection(data, token)
	return &models.ActivationResult{
		Success:   true,
		Token:     token,
		SessionID: fmt.Sprintf("session_%d", time.Now().UnixMilli()),
		Message:   "Connection activated (generic)",
	}
}

func (rs *RouterService) storeConnection(data *models.ActivationData, token string) {
	now := time.Now().UTC()
	expiry := now.Add(time.Duration(data.DurationMinutes) * time.Minute)

	session := &models.Session{
		Token:         token,
		Phone:         data.PhoneNumber,
		Package:       data.PackageName,
		Price:         data.PackagePrice,
		Currency:      data.PackageCurrency,
		Duration:      data.DurationMinutes,
		UserIP:        data.UserIP,
		MacAddress:    data.MacAddress,
		TxnID:         data.TransactionID,
		PaymentMethod: data.PaymentMethod,
		StartTime:     now.Format(time.RFC3339),
		ExpiryTime:    expiry.Format(time.RFC3339),
	}
	if session.Currency == "" {
		session.Currency = "KES"
	}
	if session.PaymentMethod == "" {
		session.PaymentMethod = "paystack"
	}

	if err := db.SaveSession(session); err != nil {
		log.Printf("Failed to save session: %v", err)
	}

	entry := &ConnectionEntry{
		Token:           token,
		PhoneNumber:     data.PhoneNumber,
		PackageName:     data.PackageName,
		PackagePrice:    data.PackagePrice,
		UserIP:          data.UserIP,
		MacAddress:      data.MacAddress,
		StartTime:       session.StartTime,
		ExpiryTime:      session.ExpiryTime,
		DurationMinutes: data.DurationMinutes,
		TransactionID:   data.TransactionID,
		MaxDevices:      maxDevicesForPackage(data.PackageName),
	}

	rs.mu.Lock()
	rs.activeConnections[token] = entry
	rs.mu.Unlock()

	rs.scheduleExpiry(token, session.ExpiryTime)
}

func (rs *RouterService) scheduleExpiry(token, expiryTime string) {
	t, err := time.Parse(time.RFC3339, expiryTime)
	if err != nil {
		return
	}
	d := time.Until(t)
	if d <= 0 {
		return
	}
	time.AfterFunc(d, func() {
		rs.mu.Lock()
		delete(rs.activeConnections, token)
		rs.mu.Unlock()
		db.ExpireSession(token)
		log.Printf("⏰ Session expired: %s", token)
	})
}

func (rs *RouterService) CheckConnectionStatus(token string) map[string]interface{} {
	rs.mu.RLock()
	entry, ok := rs.activeConnections[token]
	rs.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"active": false, "message": "Session not found"}
	}

	expiry, _ := time.Parse(time.RFC3339, entry.ExpiryTime)
	remaining := time.Until(expiry)
	if remaining <= 0 {
		return map[string]interface{}{"active": false, "message": "Session expired"}
	}

	return map[string]interface{}{
		"active":           true,
		"remainingMinutes": int(remaining.Minutes()),
		"expiresAt":        entry.ExpiryTime,
		"packageName":      entry.PackageName,
		"userIP":           entry.UserIP,
		"connectedDevices": len(entry.ConnectedDevices),
		"maxDevices":       entry.MaxDevices,
	}
}

func (rs *RouterService) RevokeConnection(identifier string) map[string]interface{} {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Try by token first
	if _, ok := rs.activeConnections[identifier]; ok {
		delete(rs.activeConnections, identifier)
		db.ExpireSession(identifier)
		return map[string]interface{}{"success": true, "message": "Connection revoked"}
	}

	// Try by IP
	for token, entry := range rs.activeConnections {
		if entry.UserIP == identifier {
			delete(rs.activeConnections, token)
			db.ExpireSession(token)
			return map[string]interface{}{"success": true, "message": "Connection revoked", "userIP": identifier}
		}
	}

	return map[string]interface{}{"success": false, "message": "Connection not found"}
}

func (rs *RouterService) ReconnectByPhone(phone, newIP, newMAC string) map[string]interface{} {
	session, err := db.GetSessionByPhone(phone)
	if err != nil || session == nil {
		return map[string]interface{}{"success": false, "message": "No active session found for this phone number"}
	}

	expiry, _ := time.Parse(time.RFC3339, session.ExpiryTime)
	remaining := time.Until(expiry)
	if remaining <= 0 {
		return map[string]interface{}{"success": false, "message": "Session has expired"}
	}

	// Re-whitelist with new IP/MAC
	log.Printf("🔄 Reconnect: phone=%s newIP=%s newMAC=%s remaining=%.0fm", phone, newIP, newMAC, remaining.Minutes())

	return map[string]interface{}{
		"success":          true,
		"connectionToken":  session.Token,
		"remainingMinutes": int(remaining.Minutes()),
		"expiresAt":        session.ExpiryTime,
		"packageName":      session.Package,
		"message":          "Reconnected successfully",
	}
}

func (rs *RouterService) CanAddDevice(token, deviceID string) map[string]interface{} {
	rs.mu.RLock()
	entry, ok := rs.activeConnections[token]
	rs.mu.RUnlock()

	if !ok {
		return map[string]interface{}{"allowed": false, "reason": "Session not found"}
	}

	for _, d := range entry.ConnectedDevices {
		if d == deviceID {
			return map[string]interface{}{"allowed": true, "reason": "Device already in session"}
		}
	}

	if len(entry.ConnectedDevices) >= entry.MaxDevices {
		return map[string]interface{}{"allowed": false, "reason": "Device limit reached"}
	}

	rs.mu.Lock()
	entry.ConnectedDevices = append(entry.ConnectedDevices, deviceID)
	rs.mu.Unlock()

	return map[string]interface{}{"allowed": true, "reason": "Device added to session"}
}

func (rs *RouterService) GetActiveConnections() []*ConnectionEntry {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	entries := make([]*ConnectionEntry, 0, len(rs.activeConnections))
	for _, e := range rs.activeConnections {
		entries = append(entries, e)
	}
	return entries
}
