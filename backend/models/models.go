package models

type Session struct {
	ID            int64   `db:"id"`
	Token         string  `db:"token"`
	Phone         string  `db:"phone"`
	Package       string  `db:"package"`
	Price         float64 `db:"price"`
	Currency      string  `db:"currency"`
	Duration      int     `db:"duration"`
	UserIP        string  `db:"user_ip"`
	MacAddress    string  `db:"mac_address"`
	TxnID         string  `db:"txn_id"`
	PaymentMethod string  `db:"payment_method"`
	StartTime     string  `db:"start_time"`
	ExpiryTime    string  `db:"expiry_time"`
	Active        int     `db:"active"`
	CreatedAt     string  `db:"created_at"`
}

type Transaction struct {
	ID            int64   `db:"id"`
	TxnID         string  `db:"txn_id"`
	Phone         string  `db:"phone"`
	Package       string  `db:"package"`
	Amount        float64 `db:"amount"`
	Currency      string  `db:"currency"`
	Status        string  `db:"status"`
	PaymentMethod string  `db:"payment_method"`
	CreatedAt     string  `db:"created_at"`
}

type Package struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Duration        string  `json:"duration"`
	DurationMinutes int     `json:"durationMinutes"`
	Price           float64 `json:"price"`
	Currency        string  `json:"currency"`
	Speed           string  `json:"speed"`
	Tier            string  `json:"tier"`
	MaxDevices      int     `json:"maxDevices"`
	Active          bool    `json:"active"`
	Popular         bool    `json:"popular,omitempty"`
	Enterprise      bool    `json:"enterprise,omitempty"`
}

type Location struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	Region       string  `json:"region"`
	ActiveUsers  int     `json:"activeUsers,omitempty"`
	DailyRevenue float64 `json:"dailyRevenue,omitempty"`
}

type ActivationData struct {
	TransactionID   string  `json:"transactionId"`
	PhoneNumber     string  `json:"phoneNumber"`
	BusinessPhone   string  `json:"businessPhone"`
	RecipientPhone  string  `json:"recipientPhone"`
	PackageName     string  `json:"packageName"`
	PackagePrice    float64 `json:"packagePrice"`
	PackageCurrency string  `json:"packageCurrency"`
	DurationMinutes int     `json:"durationMinutes"`
	ExpiryTime      string  `json:"expiryTime"`
	UserIP          string  `json:"userIP"`
	MacAddress      string  `json:"macAddress"`
	DeviceID        string  `json:"deviceId"`
	UserEmail       string  `json:"userEmail"`
	PaymentMethod   string  `json:"paymentMethod"`
}

type ActivationResult struct {
	Success   bool   `json:"success"`
	Token     string `json:"token"`
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}
