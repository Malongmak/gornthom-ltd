package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/models"
	"github.com/gornhom/backend/services"
)

type LocationHandler struct {
	mu        sync.RWMutex
	locations []models.Location
	rs        *services.RouterService
}

func NewLocationHandler(rs *services.RouterService) *LocationHandler {
	return &LocationHandler{
		rs: rs,
		locations: []models.Location{
			{ID: "GH-NRB-001", Name: "Nairobi Central Hub", Status: "online", Region: "Nairobi"},
			{ID: "GH-MBA-042", Name: "Mombasa Coastal Link", Status: "online", Region: "Mombasa"},
			{ID: "GH-KIS-109", Name: "Kisumu West Station", Status: "offline", Region: "Kisumu"},
			{ID: "GH-ELD-215", Name: "Eldoret Tech Park", Status: "online", Region: "Eldoret"},
		},
	}
}

func (h *LocationHandler) List(c *gin.Context) {
	connections := h.rs.GetActiveConnections()
	now := time.Now()

	onlineCount := 0
	h.mu.RLock()
	for _, l := range h.locations {
		if l.Status == "online" {
			onlineCount++
		}
	}

	enriched := make([]map[string]interface{}, 0, len(h.locations))
	for _, loc := range h.locations {
		activeUsers := 0
		revenue := 0.0
		if loc.Status == "online" {
			for _, conn := range connections {
				expiry, _ := time.Parse(time.RFC3339, conn.ExpiryTime)
				if expiry.After(now) {
					activeUsers++
					revenue += conn.PackagePrice
				}
			}
			if onlineCount > 0 {
				revenue = revenue / float64(onlineCount)
			}
		}
		enriched = append(enriched, map[string]interface{}{
			"id":           loc.ID,
			"name":         loc.Name,
			"status":       loc.Status,
			"region":       loc.Region,
			"activeUsers":  activeUsers,
			"dailyRevenue": revenue,
		})
	}
	h.mu.RUnlock()

	totalActive := 0
	totalRevenue := 0.0
	for _, conn := range connections {
		expiry, _ := time.Parse(time.RFC3339, conn.ExpiryTime)
		if expiry.After(now) {
			totalActive++
			totalRevenue += conn.PackagePrice
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"locations": enriched,
		"summary": gin.H{
			"total":        len(h.locations),
			"online":       onlineCount,
			"offline":      len(h.locations) - onlineCount,
			"activeUsers":  totalActive,
			"totalRevenue": totalRevenue,
		},
	})
}

func (h *LocationHandler) Add(c *gin.Context) {
	var body struct {
		Name   string `json:"name"`
		Region string `json:"region"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name is required"})
		return
	}
	loc := models.Location{
		ID:     "GH-" + body.Region[:3] + "-" + time.Now().Format("999"),
		Name:   body.Name,
		Region: body.Region,
		Status: "online",
	}
	h.mu.Lock()
	h.locations = append(h.locations, loc)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, gin.H{"success": true, "location": loc})
}

func (h *LocationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Status string `json:"status"`
		Name   string `json:"name"`
	}
	c.ShouldBindJSON(&body)

	h.mu.Lock()
	defer h.mu.Unlock()
	for i, l := range h.locations {
		if l.ID == id {
			if body.Status != "" {
				h.locations[i].Status = body.Status
			}
			if body.Name != "" {
				h.locations[i].Name = body.Name
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "location": h.locations[i]})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Location not found"})
}

func (h *LocationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, l := range h.locations {
		if l.ID == id {
			h.locations = append(h.locations[:i], h.locations[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"success": true})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Location not found"})
}
