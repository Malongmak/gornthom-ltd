package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/models"
	"github.com/gornhom/backend/services"
)

type ConnectionHandler struct {
	rs *services.RouterService
}

func NewConnectionHandler(rs *services.RouterService) *ConnectionHandler {
	return &ConnectionHandler{rs: rs}
}

func (h *ConnectionHandler) Activate(c *gin.Context) {
	var data models.ActivationData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	if data.TransactionID == "" || data.PhoneNumber == "" || data.PackageName == "" || data.DurationMinutes < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "transactionId, phoneNumber, packageName, and durationMinutes are required"})
		return
	}

	if data.UserIP == "" {
		data.UserIP = clientIP(c)
	}
	if data.PaymentMethod == "" {
		data.PaymentMethod = "paystack"
	}

	result := h.rs.ActivateConnection(&data)
	if !result.Success {
		c.JSON(http.StatusInternalServerError, result)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"connectionToken": result.Token,
		"sessionId":       result.SessionID,
		"message":         result.Message,
		"userIP":          data.UserIP,
	})
}

func (h *ConnectionHandler) Status(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"active": false, "message": "No token provided"})
		return
	}
	c.JSON(http.StatusOK, h.rs.CheckConnectionStatus(token))
}

func (h *ConnectionHandler) Revoke(c *gin.Context) {
	var body struct {
		UserIP string `json:"userIP"`
		Token  string `json:"token"`
	}
	c.ShouldBindJSON(&body)
	identifier := body.UserIP
	if identifier == "" {
		identifier = body.Token
	}
	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "IP address or token required"})
		return
	}
	c.JSON(http.StatusOK, h.rs.RevokeConnection(identifier))
}

func (h *ConnectionHandler) Reconnect(c *gin.Context) {
	var body struct {
		PhoneNumber string `json:"phoneNumber"`
		UserIP      string `json:"userIP"`
		MacAddress  string `json:"macAddress"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "phoneNumber is required"})
		return
	}
	ip := body.UserIP
	if ip == "" {
		ip = clientIP(c)
	}
	result := h.rs.ReconnectByPhone(body.PhoneNumber, ip, body.MacAddress)
	status := http.StatusOK
	if result["success"] == false {
		status = http.StatusNotFound
	}
	c.JSON(status, result)
}

func (h *ConnectionHandler) DeviceCheck(c *gin.Context) {
	var body struct {
		Token      string `json:"token"`
		MacAddress string `json:"macAddress"`
		UserIP     string `json:"userIP"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"allowed": false, "reason": "token is required"})
		return
	}
	deviceID := body.MacAddress
	if deviceID == "" {
		deviceID = body.UserIP
	}
	if deviceID == "" {
		deviceID = c.ClientIP()
	}
	c.JSON(http.StatusOK, h.rs.CanAddDevice(body.Token, deviceID))
}

func clientIP(c *gin.Context) string {
	if fwd := c.GetHeader("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return c.ClientIP()
}
