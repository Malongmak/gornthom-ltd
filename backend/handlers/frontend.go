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
	r.GET("/packages/config.js", configJS)
	r.GET("/admin/config.js", configJS)

	// Pages — packages is the landing page
	r.GET("/packages", func(c *gin.Context) { c.File(fe + "/public/packages.html") })
	r.GET("/session", func(c *gin.Context) { c.File(fe + "/public/session.html") })

	// Admin pages
	r.GET("/admin", func(c *gin.Context) { c.File(fe + "/admin/admin.html") })
	r.GET("/admin/login", func(c *gin.Context) { c.File(fe + "/admin/login.html") })

	// Captive portal OS probes — redirect straight to packages
	packagesURL := fmt.Sprintf("http://%s:%s/packages", cfg.ServerIP, cfg.Port)
	for _, probe := range []string{
		"/library/test/success.html",
		"/connecttest.txt", "/redirect", "/ncsi.txt",
		"/generate_204", "/gen_204", "/mobile/status.php",
		"/portal", "/portal/", "/portal/packages",
	} {
		probe := probe
		r.GET(probe, func(c *gin.Context) { c.Redirect(http.StatusFound, packagesURL) })
	}
	// Apple captive portal detection
	r.GET("/hotspot-detect.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", []byte(
			"<HTML><HEAD><TITLE>GORNHOM WiFi</TITLE></HEAD><BODY>GORNHOM WiFi — Sign in required</BODY></HTML>",
		))
	})

	// Root → packages
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/packages") })
}
