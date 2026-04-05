package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/config"
	"github.com/gornhom/backend/models"
	"github.com/gornhom/backend/services"
)

type PendingPayment struct {
	Status        string
	PhoneNumber   string
	Amount        float64
	PackageInfo   map[string]interface{}
	ResultCode    string
	TransactionID string
	Message       string
	CreatedAt     int64
}

type MpesaHandler struct {
	cfg             *config.Config
	rs              *services.RouterService
	mu              sync.RWMutex
	pendingPayments map[string]*PendingPayment
}

func NewMpesaHandler(cfg *config.Config, rs *services.RouterService) *MpesaHandler {
	return &MpesaHandler{
		cfg:             cfg,
		rs:              rs,
		pendingPayments: make(map[string]*PendingPayment),
	}
}

func (h *MpesaHandler) baseURL() string {
	if h.cfg.MpesaEnv == "production" {
		return "https://api.safaricom.co.ke"
	}
	return "https://sandbox.safaricom.co.ke"
}

func (h *MpesaHandler) getToken() (string, error) {
	creds := base64.StdEncoding.EncodeToString([]byte(h.cfg.MpesaConsumerKey + ":" + h.cfg.MpesaConsumerSecret))
	req, _ := http.NewRequest("GET", h.baseURL()+"/oauth/v1/generate?grant_type=client_credentials", nil)
	req.Header.Set("Authorization", "Basic "+creds)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)
	return result.AccessToken, nil
}

// POST /api/mpesa/stk-push
func (h *MpesaHandler) StkPush(c *gin.Context) {
	var body struct {
		PhoneNumber      string                 `json:"phoneNumber"`
		Amount           float64                `json:"amount"`
		AccountReference string                 `json:"accountReference"`
		TransactionDesc  string                 `json:"transactionDesc"`
		PackageInfo      map[string]interface{} `json:"packageInfo"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PhoneNumber == "" || body.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "phoneNumber and amount are required"})
		return
	}

	if h.cfg.MpesaShortcode == "" || h.cfg.MpesaPasskey == "" || h.cfg.MpesaCallbackURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "M-Pesa not configured. Check .env file."})
		return
	}

	token, err := h.getToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	timestamp := time.Now().Format("20060102150405")
	rawPass := h.cfg.MpesaShortcode + h.cfg.MpesaPasskey + timestamp
	password := base64.StdEncoding.EncodeToString([]byte(rawPass))

	payload := map[string]interface{}{
		"BusinessShortCode": h.cfg.MpesaShortcode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            int(math.Ceil(body.Amount)),
		"PartyA":            body.PhoneNumber,
		"PartyB":            h.cfg.MpesaShortcode,
		"PhoneNumber":       body.PhoneNumber,
		"CallBackURL":       h.cfg.MpesaCallbackURL,
		"AccountReference":  orDefault(body.AccountReference, "GORNHOM"),
		"TransactionDesc":   orDefault(body.TransactionDesc, "WiFi Package"),
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", h.baseURL()+"/mpesa/stkpush/v1/processrequest", strings.NewReader(string(payloadBytes)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var result struct {
		CheckoutRequestID   string `json:"CheckoutRequestID"`
		ResponseCode        string `json:"ResponseCode"`
		ResponseDescription string `json:"ResponseDescription"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)

	if result.ResponseCode != "0" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": result.ResponseDescription})
		return
	}

	h.mu.Lock()
	h.pendingPayments[result.CheckoutRequestID] = &PendingPayment{
		Status:      "pending",
		PhoneNumber: body.PhoneNumber,
		Amount:      body.Amount,
		PackageInfo: body.PackageInfo,
		CreatedAt:   time.Now().UnixMilli(),
	}
	h.mu.Unlock()

	log.Printf("📱 STK Push sent to %s | CheckoutRequestID: %s", body.PhoneNumber, result.CheckoutRequestID)
	c.JSON(http.StatusOK, gin.H{"success": true, "checkoutRequestId": result.CheckoutRequestID, "message": "STK Push sent"})
}

// POST /api/mpesa/payment-status
func (h *MpesaHandler) PaymentStatus(c *gin.Context) {
	var body struct {
		CheckoutRequestID string `json:"checkoutRequestId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.CheckoutRequestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "checkoutRequestId is required"})
		return
	}

	h.mu.RLock()
	payment, ok := h.pendingPayments[body.CheckoutRequestID]
	h.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusOK, gin.H{"status": "pending", "message": "Payment not yet confirmed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        payment.Status,
		"resultCode":    payment.ResultCode,
		"transactionId": payment.TransactionID,
		"message":       payment.Message,
	})
}

// POST /api/mpesa/callback
func (h *MpesaHandler) Callback(c *gin.Context) {
	var body struct {
		Body struct {
			StkCallback struct {
				CheckoutRequestID string `json:"CheckoutRequestID"`
				ResultCode        int    `json:"ResultCode"`
				ResultDesc        string `json:"ResultDesc"`
				CallbackMetadata  *struct {
					Item []struct {
						Name  string      `json:"Name"`
						Value interface{} `json:"Value"`
					} `json:"Item"`
				} `json:"CallbackMetadata"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
		return
	}

	cb := body.Body.StkCallback
	log.Printf("📲 M-Pesa Callback | CheckoutRequestID: %s | ResultCode: %d", cb.CheckoutRequestID, cb.ResultCode)

	h.mu.Lock()
	payment := h.pendingPayments[cb.CheckoutRequestID]
	h.mu.Unlock()

	if cb.ResultCode == 0 {
		get := func(name string) interface{} {
			if cb.CallbackMetadata == nil {
				return nil
			}
			for _, item := range cb.CallbackMetadata.Item {
				if item.Name == name {
					return item.Value
				}
			}
			return nil
		}

		txnID := fmt.Sprintf("%v", get("MpesaReceiptNumber"))
		amount, _ := get("Amount").(float64)
		phone := fmt.Sprintf("%v", get("PhoneNumber"))

		log.Printf("✅ M-Pesa confirmed | TXN: %s | Amount: %.2f | Phone: %s", txnID, amount, phone)

		if payment != nil {
			h.mu.Lock()
			payment.Status = "success"
			payment.ResultCode = "0"
			payment.TransactionID = txnID
			payment.Message = "Payment successful"
			h.mu.Unlock()

			if durRaw, ok := payment.PackageInfo["durationMinutes"]; ok {
				var dur int
				switch v := durRaw.(type) {
				case float64:
					dur = int(v)
				case int:
					dur = v
				}
				if dur > 0 {
					userIP, _ := payment.PackageInfo["userIP"].(string)
					pkgName, _ := payment.PackageInfo["name"].(string)
					if userIP != "" {
						h.rs.ActivateConnection(&models.ActivationData{
							UserIP:          userIP,
							PhoneNumber:     payment.PhoneNumber,
							PackageName:     pkgName,
							PackagePrice:    amount,
							PackageCurrency: "KES",
							DurationMinutes: dur,
							TransactionID:   txnID,
							PaymentMethod:   "mpesa",
						})
						log.Printf("🌐 Auto-activated connection for IP: %s", userIP)
					}
				}
			}
		}
	} else {
		log.Printf("❌ M-Pesa failed | ResultCode: %d | %s", cb.ResultCode, cb.ResultDesc)
		if payment != nil {
			h.mu.Lock()
			payment.Status = "failed"
			payment.ResultCode = fmt.Sprintf("%d", cb.ResultCode)
			payment.Message = cb.ResultDesc
			h.mu.Unlock()
		}
	}

	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
