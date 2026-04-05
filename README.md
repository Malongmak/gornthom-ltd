# GORNHOM WiFi Billing System

A self-hosted WiFi hotspot billing platform. Users connect to your WiFi, get redirected to a payment page, pay via Paystack, and get internet access for the duration they paid for. Built with Node.js, SQLite, and MikroTik RouterOS.

---

## How It Works

```
User connects to WiFi
        ↓
Phone shows "Sign in to network"
        ↓
Browser opens captive portal (index.html)
        ↓
User selects a package and pays via Paystack
        ↓
Backend verifies payment → saves to database → whitelists user's IP/MAC on router
        ↓
User browses the internet for the paid duration
        ↓
Session expires → router blocks access → user pays again
```

If the user disconnects and reconnects before their session expires, the system automatically restores their access using their phone number from the database.

---

## Project Structure

```
gornhom-ltd/
├── frontend/
│   ├── public/          # Customer-facing pages
│   │   ├── index.html       # Captive portal login
│   │   ├── packages.html    # Package selection & payment
│   │   ├── session.html     # Active session countdown
│   │   └── config.js        # API URL config (update for production)
│   ├── admin/           # Admin dashboard
│   │   ├── admin.html       # Dashboard overview
│   │   ├── users.html       # Active/expired sessions
│   │   ├── admin-packages.html  # Edit packages & device limits
│   │   ├── locations.html   # Hotspot locations
│   │   ├── analytics.html   # Revenue & usage charts
│   │   ├── settings.html    # Server config & health
│   │   ├── support.html     # Diagnostics & troubleshooting
│   │   └── config.js        # API URL config (update for production)
│   ├── assets/images/   # Logo and images
│   └── docs/            # Setup guides
└── backend/
    ├── routes/          # API endpoints
    ├── services/        # Router integration (MikroTik, OpenWrt, pfSense)
    ├── db.js            # SQLite database (sessions & transactions)
    ├── server.js        # Express app
    └── tests/           # API test suite (22 tests)
```

---

## Quick Start (Local Development)

**1. Clone the repo**
```bash
git clone https://github.com/Malongmak/gornthom-ltd.git
cd gornthom-ltd
```

**2. Configure the backend**
```bash
cd backend
cp .env.example .env   # or create .env manually
```

Edit `backend/.env`:
```env
PORT=3000
NODE_ENV=development
SERVER_IP=localhost
ROUTER_TYPE=generic          # use 'generic' for testing without a router
PAYSTACK_SECRET_KEY=sk_live_your_key_here
PAYSTACK_PUBLIC_KEY=pk_live_your_key_here
BUSINESS_PHONE=+254116465399
BUSINESS_EMAIL=your@email.com
MAX_DEVICES_PER_SESSION=1
```

**3. Install and start the backend**
```bash
npm install
npm start
```

**4. Open the frontend**

Open `frontend/public/index.html` in a browser, or visit `http://localhost:3000/portal`.

---

## Deploy with Docker

Docker runs both the frontend (nginx) and backend (Node.js) as containers.

**1. Copy and fill in your credentials**
```bash
cp .env.example .env
# Edit .env with your real values
```

**2. Build and start**
```bash
docker compose up --build
```

- Frontend → `http://localhost:80`
- Backend → `http://localhost:3000`

**Run in background**
```bash
docker compose up --build -d
```

**Stop**
```bash
docker compose down
```

**View logs**
```bash
docker compose logs -f backend
docker compose logs -f frontend
```

The SQLite database is stored in a Docker volume (`backend_data`) so sessions and transactions persist across container restarts.

---

## Deploy to Production (Railway + Netlify)

### Backend → Railway

1. Go to [railway.app](https://railway.app) → New Project → Deploy from GitHub
2. Set root directory to `backend`
3. Add environment variables (same as `.env`)
4. Copy your Railway URL (e.g. `https://gornhom-xyz.up.railway.app`)

### Frontend → Netlify

1. Update `frontend/public/config.js` and `frontend/admin/config.js`:
```js
window.API_BASE_URL = "https://gornhom-xyz.up.railway.app/api";
```
2. Go to [netlify.com](https://netlify.com) → New site → Import from GitHub
3. Set publish directory to `frontend`
4. Deploy

---

## MikroTik Router Setup

When you have the router, run these commands in Winbox terminal or SSH:

```
# Allow DNS
/ip firewall filter add chain=forward action=accept protocol=udp dst-port=53
/ip firewall filter add chain=forward action=accept protocol=tcp dst-port=53

# Allow access to your backend server
/ip firewall filter add chain=forward action=accept dst-address=YOUR_SERVER_IP

# Allow paid users through
/ip firewall filter add chain=forward action=accept src-address-list=allowed-users

# Block everyone else
/ip firewall filter add chain=forward action=drop in-interface=wlan1

# Redirect HTTP to captive portal
/ip firewall nat add chain=dstnat protocol=tcp dst-port=80 \
  src-address-list=!allowed-users \
  action=dst-nat to-addresses=YOUR_SERVER_IP to-ports=3000

# DNS entries so phones show "Sign in to network" automatically
/ip dns static add name=connectivitycheck.gstatic.com address=YOUR_SERVER_IP
/ip dns static add name=captive.apple.com address=YOUR_SERVER_IP
/ip dns static add name=www.msftconnecttest.com address=YOUR_SERVER_IP
```

Replace `YOUR_SERVER_IP` with the IP of the machine running the backend, and `wlan1` with your WiFi interface name.

Then update `backend/.env`:
```env
ROUTER_TYPE=mikrotik
MIKROTIK_HOST=192.168.88.1
MIKROTIK_USER=admin
MIKROTIK_PASSWORD=your_password
SERVER_IP=192.168.10.2
```

---

## Database

The backend uses **SQLite** via `better-sqlite3`. The database file is stored at `backend/data/gornhom.db`.

**Tables:**

| Table | What it stores |
|-------|---------------|
| `sessions` | Every paid session — phone, package, IP, MAC, expiry time, token |
| `transactions` | Every payment — transaction ID, amount, phone, package, status |

Sessions survive server restarts. When the backend starts, it loads all active sessions from the database back into memory so reconnects work immediately.

**Backup the database:**
```bash
cp backend/data/gornhom.db backup/gornhom-$(date +%Y%m%d).db
```

---

## Admin Dashboard

Open `frontend/admin/admin.html` in a browser.

| Page | What you can do |
|------|----------------|
| Dashboard | Live revenue, active users, session chart |
| Users | View all sessions, revoke access |
| Packages | Edit price, speed, max devices per package |
| Locations | Add/edit/toggle hotspot locations |
| Analytics | Revenue by package, hourly activity |
| Settings | Update API URL, test backend connection |
| Support | Run diagnostics, troubleshooting FAQ |

---

## Running Tests

```bash
cd backend
npm test
```

22 tests covering all API endpoints — connection activate/status/revoke, Paystack webhook, M-Pesa callbacks, and 404 handling.

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Backend server port | `3000` |
| `SERVER_IP` | IP of this machine on the hotspot network | `localhost` |
| `ROUTER_TYPE` | `mikrotik` / `openwrt` / `pfsense` / `generic` | `generic` |
| `MIKROTIK_HOST` | Router IP address | `192.168.88.1` |
| `MIKROTIK_USER` | Router admin username | `admin` |
| `MIKROTIK_PASSWORD` | Router admin password | — |
| `PAYSTACK_SECRET_KEY` | From Paystack dashboard | — |
| `PAYSTACK_PUBLIC_KEY` | From Paystack dashboard | — |
| `MPESA_CONSUMER_KEY` | From Safaricom Daraja | — |
| `MPESA_CONSUMER_SECRET` | From Safaricom Daraja | — |
| `MPESA_SHORTCODE` | M-Pesa business shortcode | — |
| `MPESA_PASSKEY` | M-Pesa passkey | — |
| `MPESA_CALLBACK_URL` | Public URL for M-Pesa callbacks | — |
| `BUSINESS_PHONE` | Phone number receiving payments | `+254116465399` |
| `MAX_DEVICES_PER_SESSION` | Max devices per paid session | `1` |

---

## Tech Stack

- **Backend** — Node.js, Express, SQLite (better-sqlite3)
- **Frontend** — HTML, Tailwind CSS (local build), vanilla JS
- **Payments** — Paystack (card/mobile money), M-Pesa STK Push
- **Router** — MikroTik RouterOS API, OpenWrt SSH, pfSense REST
- **Deployment** — Docker + nginx, Railway, Netlify
