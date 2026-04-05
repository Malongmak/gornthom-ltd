package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/db"
	"github.com/gornhom/backend/services"
)

type AdminHandler struct {
	rs        *services.RouterService
	startTime time.Time
}

func NewAdminHandler(rs *services.RouterService) *AdminHandler {
	return &AdminHandler{rs: rs, startTime: time.Now()}
}

func (h *AdminHandler) Stats(c *gin.Context) {
	sessions, _ := db.GetAllSessions()
	transactions, _ := db.GetAllTransactions()
	now := time.Now()

	totalRevenue := 0.0
	for _, t := range transactions {
		totalRevenue += t.Amount
	}

	packageCounts := map[string]int{}
	activeCount := 0
	expiredCount := 0

	type connOut struct {
		Phone         string  `json:"phone"`
		Package       string  `json:"package"`
		PackagePrice  float64 `json:"packagePrice"`
		StartTime     string  `json:"startTime"`
		ExpiresAt     string  `json:"expiresAt"`
		TransactionID string  `json:"transactionId"`
		Active        bool    `json:"active"`
	}
	var conns []connOut

	for _, s := range sessions {
		expiry, _ := time.Parse(time.RFC3339, s.ExpiryTime)
		active := expiry.After(now) && s.Active == 1
		if active {
			activeCount++
		} else {
			expiredCount++
		}
		packageCounts[s.Package]++
		conns = append(conns, connOut{
			Phone:         s.Phone,
			Package:       s.Package,
			PackagePrice:  s.Price,
			StartTime:     s.StartTime,
			ExpiresAt:     s.ExpiryTime,
			TransactionID: s.TxnID,
			Active:        active,
		})
	}

	type pkgCount struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	topPackages := make([]pkgCount, 0)
	for name, count := range packageCounts {
		topPackages = append(topPackages, pkgCount{Name: name, Count: count})
	}

	c.JSON(http.StatusOK, gin.H{
		"activeConnections":  activeCount,
		"totalConnections":   len(sessions),
		"expiredConnections": expiredCount,
		"totalRevenue":       totalRevenue,
		"recentRevenue":      totalRevenue,
		"topPackages":        topPackages,
		"serverUptime":       int(time.Since(h.startTime).Seconds()),
		"connections":        conns,
	})
}
