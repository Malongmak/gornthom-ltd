package handlers

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func getAdminPassword() string {
	if p := os.Getenv("ADMIN_PASSWORD"); p != "" {
		return p
	}
	return "admin123" // default — change via ADMIN_PASSWORD in .env
}

// AdminAuth middleware — checks Bearer token or X-Admin-Token header
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Admin-Token")
		if token == "" {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		expected := getAdminPassword()
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Unauthorized",
			})
			return
		}
		c.Next()
	}
}

// POST /api/admin/login
func AdminLogin(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Password required"})
		return
	}
	expected := getAdminPassword()
	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(expected)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "token": expected})
}
