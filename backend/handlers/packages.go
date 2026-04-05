package handlers

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/models"
)

type PackageHandler struct {
	mu       sync.RWMutex
	packages []models.Package
}

func NewPackageHandler() *PackageHandler {
	return &PackageHandler{
		packages: []models.Package{
			{ID: "30min", Name: "30 Minutes", Duration: "30 min", DurationMinutes: 30, Price: 5, Currency: "KES", Speed: "5Mbps", Tier: "Lite", MaxDevices: 1, Active: true},
			{ID: "1hour", Name: "1 Hour", Duration: "1 hour", DurationMinutes: 60, Price: 10, Currency: "KES", Speed: "5Mbps", Tier: "Basic", MaxDevices: 1, Active: true},
			{ID: "1day", Name: "1 Day", Duration: "24 hours", DurationMinutes: 1440, Price: 60, Currency: "KES", Speed: "10Mbps", Tier: "Standard", MaxDevices: 2, Active: true, Popular: true},
			{ID: "1week", Name: "1 Week", Duration: "7 days", DurationMinutes: 10080, Price: 260, Currency: "KES", Speed: "20Mbps", Tier: "Premium", MaxDevices: 3, Active: true},
			{ID: "1month", Name: "1 Month", Duration: "30 days", DurationMinutes: 43200, Price: 500, Currency: "KES", Speed: "Unlimited", Tier: "Ultimate", MaxDevices: 5, Active: true},
			{ID: "enterprise", Name: "Enterprise Plan", Duration: "30 days", DurationMinutes: 43200, Price: 2500, Currency: "KES", Speed: "100Mbps", Tier: "Business", MaxDevices: 25, Active: true, Enterprise: true},
		},
	}
}

func (h *PackageHandler) List(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	active := make([]models.Package, 0)
	for _, p := range h.packages {
		if p.Active {
			active = append(active, p)
		}
	}
	c.JSON(http.StatusOK, active)
}

func (h *PackageHandler) ListAll(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c.JSON(http.StatusOK, h.packages)
}

func (h *PackageHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		MaxDevices *int     `json:"maxDevices"`
		Price      *float64 `json:"price"`
		Speed      *string  `json:"speed"`
		Active     *bool    `json:"active"`
		Name       *string  `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for i, p := range h.packages {
		if p.ID == id {
			if body.MaxDevices != nil {
				h.packages[i].MaxDevices = *body.MaxDevices
			}
			if body.Price != nil {
				h.packages[i].Price = *body.Price
			}
			if body.Speed != nil {
				h.packages[i].Speed = *body.Speed
			}
			if body.Active != nil {
				h.packages[i].Active = *body.Active
			}
			if body.Name != nil {
				h.packages[i].Name = *body.Name
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "package": h.packages[i]})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Package not found"})
}
