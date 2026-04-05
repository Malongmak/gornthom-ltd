package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/config"
	"github.com/gornhom/backend/handlers"
	"github.com/gornhom/backend/services"
)

func Register(r *gin.Engine, cfg *config.Config, rs *services.RouterService) {
	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Connection
	connH := handlers.NewConnectionHandler(rs)
	conn := r.Group("/api/connection")
	{
		conn.POST("/activate", connH.Activate)
		conn.GET("/status", connH.Status)
		conn.POST("/revoke", connH.Revoke)
		conn.POST("/reconnect", connH.Reconnect)
		conn.POST("/device-check", connH.DeviceCheck)
	}

	// Webhooks & Paystack
	webhookH := handlers.NewWebhookHandler(cfg, rs)
	r.POST("/api/webhooks/paystack", webhookH.Paystack)
	r.GET("/api/paystack/verify/:reference", webhookH.VerifyPaystack)

	// M-Pesa
	mpesaH := handlers.NewMpesaHandler(cfg, rs)
	mpesa := r.Group("/api/mpesa")
	{
		mpesa.POST("/stk-push", mpesaH.StkPush)
		mpesa.POST("/payment-status", mpesaH.PaymentStatus)
		mpesa.POST("/callback", mpesaH.Callback)
	}

	// Locations
	locH := handlers.NewLocationHandler(rs)
	locs := r.Group("/api/locations")
	{
		locs.GET("", locH.List)
		locs.POST("", locH.Add)
		locs.PATCH("/:id", locH.Update)
		locs.DELETE("/:id", locH.Delete)
	}

	// Packages
	pkgH := handlers.NewPackageHandler()
	pkgs := r.Group("/api/packages")
	{
		pkgs.GET("", pkgH.List)
		pkgs.GET("/all", pkgH.ListAll)
		pkgs.PATCH("/:id", pkgH.Update)
	}

	// Admin
	r.GET("/api/admin/stats", handlers.NewAdminHandler(rs).Stats)

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"message":    "GORNHOM Backend API is running",
			"routerType": cfg.RouterType,
		})
	})

	// Frontend — portal, admin pages, assets, captive portal probes
	handlers.RegisterFrontend(r, cfg)

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Endpoint not found"})
	})
}
