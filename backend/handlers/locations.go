package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/db"
	"github.com/gornhom/backend/models"
	"github.com/gornhom/backend/services"
)

type LocationHandler struct {
	rs *services.RouterService
}

var defaultLocations = []models.Location{
	{ID: "GH-NRB-001", Name: "Nairobi Central Hub", Status: "online", Region: "Nairobi"},
	{ID: "GH-MBA-042", Name: "Mombasa Coastal Link", Status: "online", Region: "Mombasa"},
	{ID: "GH-KIS-109", Name: "Kisumu West Station", Status: "offline", Region: "Kisumu"},
	{ID: "GH-ELD-215", Name: "Eldoret Tech Park", Status: "online", Region: "Eldoret"},
}

func NewLocationHandler(rs *services.RouterService) *LocationHandler {
	if exists, _ := db.LocationExists(); !exists {
		for _, l := range defaultLocations {
			db.InsertLocation(l)
		}
	}
	return &LocationHandler{rs: rs}
}

func (h *LocationHandler) List(c *gin.Context) {
	locs, err := db.GetAllLocations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	connections := h.rs.GetActiveConnections()
	now := time.Now()

	onlineCount := 0
	for _, l := range locs {
		if l.Status == "online" {
			onlineCount++
		}
	}

	totalActive := 0
	totalRevenue := 0.0
	for _, conn := range connections {
		expiry, _ := time.Parse(time.RFC3339, conn.ExpiryTime)
		if expiry.After(now) {
			totalActive++
			totalRevenue += conn.PackagePrice
		}
	}

	enriched := make([]map[string]interface{}, 0, len(locs))
	for _, loc := range locs {
		activeUsers := 0
		revenue := 0.0
		if loc.Status == "online" && onlineCount > 0 {
			activeUsers = totalActive / onlineCount
			revenue = totalRevenue / float64(onlineCount)
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

	c.JSON(http.StatusOK, gin.H{
		"locations": enriched,
		"summary": gin.H{
			"total":        len(locs),
			"online":       onlineCount,
			"offline":      len(locs) - onlineCount,
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
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" || body.Region == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name and region are required"})
		return
	}
	region := body.Region
	if len(region) < 3 {
		region = region + strings.Repeat("X", 3-len(region))
	}
	loc := models.Location{
		ID:     fmt.Sprintf("GH-%s-%d", strings.ToUpper(region[:3]), time.Now().UnixMilli()%1000),
		Name:   body.Name,
		Region: body.Region,
		Status: "online",
	}
	if err := db.InsertLocation(loc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "location": loc})
}

func (h *LocationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Status string `json:"status"`
		Name   string `json:"name"`
		Region string `json:"region"`
	}
	c.ShouldBindJSON(&body)
	if err := db.UpdateLocation(id, body.Status, body.Name, body.Region); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	locs, _ := db.GetAllLocations()
	for _, l := range locs {
		if l.ID == id {
			c.JSON(http.StatusOK, gin.H{"success": true, "location": l})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Location not found"})
}

func (h *LocationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := db.DeleteLocation(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
