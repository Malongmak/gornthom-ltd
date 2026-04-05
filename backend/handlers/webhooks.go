package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gornhom/backend/config"
	"github.com/gornhom/backend/db"
	"github.com/gornhom/backend/models"
	"github.com/gornhom/backend/services"
)

var packageDurations = map[string]int{
	"30 Minutes":      30,
	"1 Hour":          60,
	"1 Day":           1440,
	"1 Week":          10080,
	"1 Month":         43200,
	"Enterprise Plan": 43200,
}

type WebhookHandler struct {
	cfg *config.Config
	rs  *services.RouterService
}

func NewWebhookHandler(cfg *config.Config, rs *services.RouterService) *WebhookHandler {
	return &WebhookHandler{cfg: cfg, rs: rs}
}

func (h *WebhookHandler) Paystack(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	// Verify HMAC-SHA512 signature
	mac := hmac.New(sha512.New, []byte(h.cfg.PaystackKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if expected != c.GetHeader("x-paystack-signature") {
		log.Println("❌ Invalid Paystack webhook signature")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string  `json:"reference"`
			Amount    float64 `json:"amount"`
			Customer  struct {
				Email string `json:"email"`
			} `json:"customer"`
			Metadata struct {
				CustomFields []struct {
					VariableName string `json:"variable_name"`
					Value        string `json:"value"`
				} `json:"custom_fields"`
			} `json:"metadata"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	log.Printf("📨 Paystack Webhook | Event: %s | Ref: %s", event.Event, event.Data.Reference)

	if event.Event == "charge.success" {
		ref := event.Data.Reference
		amount := event.Data.Amount / 100

		// Verify with Paystack
		verified, err := verifyPaystackPayment(ref, h.cfg.PaystackKey)
		if err != nil || verified != "success" {
			log.Printf("❌ Payment %s not verified", ref)
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}

		log.Printf("✅ Payment verified | Ref: %s | Amount: KES %.2f", ref, amount)

		get := func(name string) string {
			for _, f := range event.Data.Metadata.CustomFields {
				if f.VariableName == name {
					return f.Value
				}
			}
			return ""
		}

		packageName := get("package")
		customerPhone := get("customer_phone")
		durationMinutes := packageDurations[packageName]

		if packageName != "" && durationMinutes > 0 && customerPhone != "" {
			db.SaveTransaction(&models.Transaction{
				TxnID:         ref,
				Phone:         customerPhone,
				Package:       packageName,
				Amount:        amount,
				Currency:      "KES",
				Status:        "success",
				PaymentMethod: "paystack",
			})

			result := h.rs.ActivateConnection(&models.ActivationData{
				TransactionID:   ref,
				PhoneNumber:     customerPhone,
				PackageName:     packageName,
				PackagePrice:    amount,
				PackageCurrency: "KES",
				DurationMinutes: durationMinutes,
				PaymentMethod:   "paystack",
			})

			if result.Success {
				log.Printf("🌐 Connection activated via webhook | Package: %s | Phone: %s", packageName, customerPhone)
			} else {
				log.Printf("❌ Webhook activation failed: %s", result.Message)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

func verifyPaystackPayment(reference, secretKey string) (string, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference), nil)
	req.Header.Set("Authorization", "Bearer "+secretKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)
	return result.Data.Status, nil
}

// VerifyPaystack is the GET /api/paystack/verify/:reference endpoint
func (h *WebhookHandler) VerifyPaystack(c *gin.Context) {
	ref := c.Param("reference")
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", ref), nil)
	req.Header.Set("Authorization", "Bearer "+h.cfg.PaystackKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, _ := result["data"].(map[string]interface{})
	status, _ := data["status"].(string)
	c.JSON(http.StatusOK, gin.H{"success": status == "success", "status": status, "data": data})
}

// rawBodyMiddleware reads the body and stores it so Gin can still bind it
func RawBodyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		c.Set("rawBody", body)
		c.Next()
	}
}
