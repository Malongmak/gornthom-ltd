const Database = require('better-sqlite3');
const path = require('path');
const fs = require('fs');

const DB_DIR = path.join(__dirname, 'data');
if (!fs.existsSync(DB_DIR)) fs.mkdirSync(DB_DIR);

const db = new Database(path.join(DB_DIR, 'gornhom.db'));

// Enable WAL mode for better concurrent read performance
db.pragma('journal_mode = WAL');

// ── Schema ────────────────────────────────────────────────────────────────────
db.exec(`
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
`);

// ── Session helpers ───────────────────────────────────────────────────────────
const saveSession = db.prepare(`
  INSERT OR REPLACE INTO sessions
    (token, phone, package, price, currency, duration, user_ip, mac_address, txn_id, payment_method, start_time, expiry_time, active)
  VALUES
    (@token, @phone, @package, @price, @currency, @duration, @userIP, @macAddress, @txnId, @paymentMethod, @startTime, @expiryTime, 1)
`);

const getSessionByToken = db.prepare(`SELECT * FROM sessions WHERE token = ? AND active = 1`);
const getSessionByPhone = db.prepare(`SELECT * FROM sessions WHERE phone = ? AND active = 1 AND expiry_time > datetime('now') ORDER BY expiry_time DESC LIMIT 1`);
const expireSession     = db.prepare(`UPDATE sessions SET active = 0 WHERE token = ?`);
const expireByPhone     = db.prepare(`UPDATE sessions SET active = 0 WHERE phone = ? AND active = 1`);
const getAllActive       = db.prepare(`SELECT * FROM sessions WHERE active = 1 AND expiry_time > datetime('now')`);
const getAllSessions     = db.prepare(`SELECT * FROM sessions ORDER BY created_at DESC`);

// ── Transaction helpers ───────────────────────────────────────────────────────
const saveTransaction = db.prepare(`
  INSERT OR IGNORE INTO transactions (txn_id, phone, package, amount, currency, status, payment_method)
  VALUES (@txnId, @phone, @package, @amount, @currency, @status, @paymentMethod)
`);
const getAllTransactions = db.prepare(`SELECT * FROM transactions ORDER BY created_at DESC`);

module.exports = {
  db,
  saveSession,
  getSessionByToken,
  getSessionByPhone,
  expireSession,
  expireByPhone,
  getAllActive,
  getAllSessions,
  saveTransaction,
  getAllTransactions,
};
