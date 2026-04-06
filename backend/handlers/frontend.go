package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/config"
)

// RegisterFrontend registers all static/portal/admin/captive-portal routes.
func RegisterFrontend(r *gin.Engine, cfg *config.Config) {
	fe := cfg.FrontendDir

	// Static assets
	r.Static("/assets", fe+"/assets")
	r.Static("/static", fe+"/assets")

	// Dynamic config.js — injects live API_BASE_URL for every page context
	configJS := func(c *gin.Context) {
		c.Header("Content-Type", "application/javascript")
		c.String(http.StatusOK, `window.API_BASE_URL = "http://%s:%s/api";`, cfg.ServerIP, cfg.Port)
	}
	r.GET("/config.js", configJS)
	r.GET("/portal/config.js", configJS)
	r.GET("/admin/config.js", configJS)

	// Portal pages
	r.GET("/portal", func(c *gin.Context) { c.File(fe + "/public/index.html") })
	r.GET("/portal/packages", func(c *gin.Context) { c.File(fe + "/public/packages.html") })
	r.GET("/portal/session", func(c *gin.Context) { c.File(fe + "/public/session.html") })
	// Admin pages
	r.GET("/admin", func(c *gin.Context) { c.File(fe + "/admin/admin.html") })
	r.GET("/admin/packages", func(c *gin.Context) { c.File(fe + "/admin/admin-packages.html") })
	r.GET("/admin/analytics", func(c *gin.Context) { c.File(fe + "/admin/analytics.html") })
	r.GET("/admin/locations", func(c *gin.Context) { c.File(fe + "/admin/locations.html") })
	r.GET("/admin/settings", func(c *gin.Context) { c.File(fe + "/admin/settings.html") })
	r.GET("/admin/support", func(c *gin.Context) { c.File(fe + "/admin/support.html") })
	r.GET("/admin/users", func(c *gin.Context) { c.File(fe + "/admin/users.html") })

	// Captive portal OS probes — redirect to portal
	portalRedirect := fmt.Sprintf("http://%s:%s/portal", cfg.ServerIP, cfg.Port)
	for _, probe := range []string{
		"/library/test/success.html",
		"/connecttest.txt", "/redirect", "/ncsi.txt",
		"/generate_204", "/gen_204", "/mobile/status.php",
	} {
		probe := probe
		r.GET(probe, func(c *gin.Context) { c.Redirect(http.StatusFound, portalRedirect) })
	}
	// Apple captive portal detection — must return non-success HTML
	r.GET("/hotspot-detect.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", []byte(
			"<HTML><HEAD><TITLE>GORNHOM WiFi</TITLE></HEAD><BODY>GORNHOM WiFi — Sign in required</BODY></HTML>",
		))
	})

	// Root → portal
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/portal") })
}
