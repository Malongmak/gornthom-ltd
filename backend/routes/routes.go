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

	// Locations (public read)
	locH := handlers.NewLocationHandler(rs)

	// Packages (public read)
	pkgH := handlers.NewPackageHandler()

	// Admin login (public)
	r.POST("/api/admin/login", handlers.AdminLogin)

	// Protected admin routes
	admin := r.Group("/api/admin", handlers.AdminAuth())
	{
		adminH := handlers.NewAdminHandler(rs)
		admin.GET("/stats", adminH.Stats)
		admin.GET("/transactions", adminH.Transactions)
	}

	// Protected settings
	settingsH := handlers.NewSettingsHandler(cfg)
	settings := r.Group("/api/settings", handlers.AdminAuth())
	{
		settings.GET("", settingsH.Get)
		settings.PATCH("", settingsH.Update)
	}

	// Protected package/location mutations
	r.GET("/api/packages", pkgH.List)
	r.GET("/api/packages/all", pkgH.ListAll)
	r.PATCH("/api/packages/:id", handlers.AdminAuth(), pkgH.Update)

	r.GET("/api/locations", locH.List)
	r.POST("/api/locations", handlers.AdminAuth(), locH.Add)
	r.PATCH("/api/locations/:id", handlers.AdminAuth(), locH.Update)
	r.DELETE("/api/locations/:id", handlers.AdminAuth(), locH.Delete)

	r.POST("/api/connection/revoke", handlers.AdminAuth(), connH.Revoke)

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
