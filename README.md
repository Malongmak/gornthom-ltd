# GORNHOM WiFi Billing System

A WiFi hotspot billing system with Paystack payment integration and MikroTik router support.

## Project Structure

```
gornhom-ltd/
├── frontend/
│   ├── public/          # Customer-facing pages
│   │   ├── index.html       # Captive portal login
│   │   ├── packages.html    # Package selection & payment
│   │   └── session.html     # Active session status
│   ├── admin/           # Admin dashboard pages
│   │   ├── admin.html
│   │   ├── users.html
│   │   ├── admin-packages.html
│   │   ├── locations.html
│   │   ├── analytics.html
│   │   ├── settings.html
│   │   └── support.html
│   ├── assets/
│   │   └── images/
│   └── docs/            # Setup guides & API references
└── backend/             # Node.js/Express API server
    ├── config/
    ├── routes/
    ├── services/
    ├── tests/
    └── server.js
```

## Quick Start

```bash
cd backend
npm install
cp .env.example .env   # fill in your credentials
npm start
```

Then open `frontend/public/index.html` in a browser.

## Deployment

- **Backend**: deploy the `backend/` folder to any Node.js host (Railway, Render, VPS)
- **Frontend**: serve the `frontend/` folder as static files (Nginx, Apache, Netlify, or same VPS)
- Update `API_BASE_URL` in `frontend/public/packages.html` to your production backend URL
