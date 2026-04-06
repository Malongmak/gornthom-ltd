package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gornhom/backend/models"
)

var DB *sql.DB

func Init() {
	dir := "data"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	var err error
	DB, err = sql.Open("sqlite3", filepath.Join(dir, "gornhom.db")+"?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	migrate()
}

func migrate() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			token       TEXT    UNIQUE NOT NULL,
			phone       TEXT    NOT NULL,
			package     TEXT    NOT NULL,
			price       REAL    DEFAULT 0,
			currency    TEXT    DEFAULT 'KES',
			duration    INTEGER NOT NULL,
			user_ip     TEXT,
			mac_address TEXT,
			txn_id      TEXT    UNIQUE,
			payment_method TEXT DEFAULT 'paystack',
			start_time  TEXT    NOT NULL,
			expiry_time TEXT    NOT NULL,
			active      INTEGER DEFAULT 1,
			created_at  TEXT    DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			txn_id      TEXT    UNIQUE NOT NULL,
			phone       TEXT    NOT NULL,
			package     TEXT    NOT NULL,
			amount      REAL    NOT NULL,
			currency    TEXT    DEFAULT 'KES',
			status      TEXT    DEFAULT 'success',
			payment_method TEXT DEFAULT 'paystack',
			created_at  TEXT    DEFAULT (datetime('now'))
		);
		CREATE TABLE IF NOT EXISTS packages (
			id               TEXT PRIMARY KEY,
			name             TEXT NOT NULL,
			duration         TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			price            REAL NOT NULL,
			currency         TEXT DEFAULT 'KES',
			speed            TEXT NOT NULL,
			tier             TEXT NOT NULL,
			max_devices      INTEGER DEFAULT 1,
			active           INTEGER DEFAULT 1,
			popular          INTEGER DEFAULT 0,
			enterprise       INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS locations (
			id     TEXT PRIMARY KEY,
			name   TEXT NOT NULL,
			status TEXT DEFAULT 'online',
			region TEXT NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}
}

func SaveSession(s *models.Session) error {
	_, err := DB.Exec(`
		INSERT OR REPLACE INTO sessions
			(token, phone, package, price, currency, duration, user_ip, mac_address, txn_id, payment_method, start_time, expiry_time, active)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,1)`,
		s.Token, s.Phone, s.Package, s.Price, s.Currency, s.Duration,
		s.UserIP, s.MacAddress, s.TxnID, s.PaymentMethod, s.StartTime, s.ExpiryTime,
	)
	return err
}

func GetSessionByToken(token string) (*models.Session, error) {
	row := DB.QueryRow(`SELECT id,token,phone,package,price,currency,duration,user_ip,mac_address,txn_id,payment_method,start_time,expiry_time,active,created_at FROM sessions WHERE token=? AND active=1`, token)
	return scanSession(row)
}

func GetSessionByPhone(phone string) (*models.Session, error) {
	row := DB.QueryRow(`SELECT id,token,phone,package,price,currency,duration,user_ip,mac_address,txn_id,payment_method,start_time,expiry_time,active,created_at FROM sessions WHERE phone=? AND active=1 AND expiry_time > datetime('now') ORDER BY expiry_time DESC LIMIT 1`, phone)
	return scanSession(row)
}

func GetSessionByTxn(txnID string) (*models.Session, error) {
	row := DB.QueryRow(`SELECT id,token,phone,package,price,currency,duration,user_ip,mac_address,txn_id,payment_method,start_time,expiry_time,active,created_at FROM sessions WHERE txn_id=? ORDER BY created_at DESC LIMIT 1`, txnID)
	return scanSession(row)
}

func ExpireSession(token string) error {
	_, err := DB.Exec(`UPDATE sessions SET active=0 WHERE token=?`, token)
	return err
}

func ExpireByPhone(phone string) error {
	_, err := DB.Exec(`UPDATE sessions SET active=0 WHERE phone=? AND active=1`, phone)
	return err
}

func GetAllActive() ([]models.Session, error) {
	rows, err := DB.Query(`SELECT id,token,phone,package,price,currency,duration,user_ip,mac_address,txn_id,payment_method,start_time,expiry_time,active,created_at FROM sessions WHERE active=1 AND expiry_time > datetime('now')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func GetAllSessions() ([]models.Session, error) {
	rows, err := DB.Query(`SELECT id,token,phone,package,price,currency,duration,user_ip,mac_address,txn_id,payment_method,start_time,expiry_time,active,created_at FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func SaveTransaction(t *models.Transaction) error {
	_, err := DB.Exec(`INSERT OR IGNORE INTO transactions (txn_id,phone,package,amount,currency,status,payment_method) VALUES (?,?,?,?,?,?,?)`,
		t.TxnID, t.Phone, t.Package, t.Amount, t.Currency, t.Status, t.PaymentMethod)
	return err
}

func GetAllTransactions() ([]models.Transaction, error) {
	rows, err := DB.Query(`SELECT id,txn_id,phone,package,amount,currency,status,payment_method,created_at FROM transactions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var txns []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.TxnID, &t.Phone, &t.Package, &t.Amount, &t.Currency, &t.Status, &t.PaymentMethod, &t.CreatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, nil
}

func scanSession(row *sql.Row) (*models.Session, error) {
	var s models.Session
	var macAddress, txnID, userIP sql.NullString
	err := row.Scan(&s.ID, &s.Token, &s.Phone, &s.Package, &s.Price, &s.Currency, &s.Duration,
		&userIP, &macAddress, &txnID, &s.PaymentMethod, &s.StartTime, &s.ExpiryTime, &s.Active, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	s.UserIP = userIP.String
	s.MacAddress = macAddress.String
	s.TxnID = txnID.String
	return &s, err
}

func scanSessions(rows *sql.Rows) ([]models.Session, error) {
	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		var macAddress, txnID, userIP sql.NullString
		if err := rows.Scan(&s.ID, &s.Token, &s.Phone, &s.Package, &s.Price, &s.Currency, &s.Duration,
			&userIP, &macAddress, &txnID, &s.PaymentMethod, &s.StartTime, &s.ExpiryTime, &s.Active, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.UserIP = userIP.String
		s.MacAddress = macAddress.String
		s.TxnID = txnID.String
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// ── Packages ──────────────────────────────────────────────────────────────

func GetAllPackages() ([]models.Package, error) {
	rows, err := DB.Query(`SELECT id,name,duration,duration_minutes,price,currency,speed,tier,max_devices,active,popular,enterprise FROM packages ORDER BY price ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pkgs []models.Package
	for rows.Next() {
		var p models.Package
		var active, popular, enterprise int
		if err := rows.Scan(&p.ID, &p.Name, &p.Duration, &p.DurationMinutes, &p.Price, &p.Currency, &p.Speed, &p.Tier, &p.MaxDevices, &active, &popular, &enterprise); err != nil {
			return nil, err
		}
		p.Active = active == 1
		p.Popular = popular == 1
		p.Enterprise = enterprise == 1
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func UpsertPackage(p models.Package) error {
	active, popular, enterprise := 0, 0, 0
	if p.Active {
		active = 1
	}
	if p.Popular {
		popular = 1
	}
	if p.Enterprise {
		enterprise = 1
	}
	_, err := DB.Exec(`INSERT OR REPLACE INTO packages (id,name,duration,duration_minutes,price,currency,speed,tier,max_devices,active,popular,enterprise)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Duration, p.DurationMinutes, p.Price, p.Currency, p.Speed, p.Tier, p.MaxDevices, active, popular, enterprise)
	return err
}

func UpdatePackage(id string, name *string, price *float64, speed *string, maxDevices *int, active *bool) error {
	if name != nil {
		if _, err := DB.Exec(`UPDATE packages SET name=? WHERE id=?`, *name, id); err != nil {
			return err
		}
	}
	if price != nil {
		if _, err := DB.Exec(`UPDATE packages SET price=? WHERE id=?`, *price, id); err != nil {
			return err
		}
	}
	if speed != nil {
		if _, err := DB.Exec(`UPDATE packages SET speed=? WHERE id=? `, *speed, id); err != nil {
			return err
		}
	}
	if maxDevices != nil {
		if _, err := DB.Exec(`UPDATE packages SET max_devices=? WHERE id=?`, *maxDevices, id); err != nil {
			return err
		}
	}
	if active != nil {
		v := 0
		if *active {
			v = 1
		}
		if _, err := DB.Exec(`UPDATE packages SET active=? WHERE id=?`, v, id); err != nil {
			return err
		}
	}
	return nil
}

func PackageExists() (bool, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM packages`).Scan(&count)
	return count > 0, err
}

// ── Locations ─────────────────────────────────────────────────────────────

func GetAllLocations() ([]models.Location, error) {
	rows, err := DB.Query(`SELECT id,name,status,region FROM locations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locs []models.Location
	for rows.Next() {
		var l models.Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Status, &l.Region); err != nil {
			return nil, err
		}
		locs = append(locs, l)
	}
	return locs, nil
}

func InsertLocation(l models.Location) error {
	_, err := DB.Exec(`INSERT INTO locations (id,name,status,region) VALUES (?,?,?,?)`, l.ID, l.Name, l.Status, l.Region)
	return err
}

func UpdateLocation(id, status, name string) error {
	if status != "" {
		if _, err := DB.Exec(`UPDATE locations SET status=? WHERE id=?`, status, id); err != nil {
			return err
		}
	}
	if name != "" {
		if _, err := DB.Exec(`UPDATE locations SET name=? WHERE id=?`, name, id); err != nil {
			return err
		}
	}
	return nil
}

func DeleteLocation(id string) error {
	_, err := DB.Exec(`DELETE FROM locations WHERE id=?`, id)
	return err
}

func LocationExists() (bool, error) {
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM locations`).Scan(&count)
	return count > 0, err
}
