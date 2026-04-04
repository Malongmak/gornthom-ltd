# Captive Portal Setup Guide

This explains how to configure MikroTik to redirect users to the GORNHOM payment page automatically when they connect to WiFi.

## How It Works

```
User connects to WiFi
        ↓
Opens any website (e.g. google.com)
        ↓
MikroTik intercepts HTTP traffic (port 80)
        ↓
Redirects to: http://YOUR_SERVER_IP:3000/portal
        ↓
User sees login/payment page
        ↓
User pays → backend whitelists their IP/MAC
        ↓
User can now access the internet freely
```

## Step 1 — MikroTik Firewall Rules

Connect to your MikroTik via Winbox or SSH and run these commands:

```
# Allow DNS so devices can resolve domains
/ip firewall filter add chain=forward action=accept protocol=udp dst-port=53 comment="Allow DNS"
/ip firewall filter add chain=forward action=accept protocol=tcp dst-port=53 comment="Allow DNS TCP"

# Allow access to your backend server (so the portal page loads)
/ip firewall filter add chain=forward action=accept dst-address=YOUR_SERVER_IP comment="Allow portal server"

# Allow already-whitelisted users through
/ip firewall filter add chain=forward action=accept src-address-list=allowed-users comment="Allow paid users"

# Block everything else (unauthenticated users)
/ip firewall filter add chain=forward action=drop in-interface=wlan1 comment="Block unpaid users"
```

Replace `YOUR_SERVER_IP` with the IP of the machine running the backend.
Replace `wlan1` with your actual WiFi interface name.

## Step 2 — Redirect HTTP to Captive Portal

This redirects all HTTP traffic from unauthenticated users to your portal:

```
/ip firewall nat add chain=dstnat protocol=tcp dst-port=80 \
  src-address-list=!allowed-users \
  action=dst-nat to-addresses=YOUR_SERVER_IP to-ports=3000 \
  comment="Captive portal redirect"
```

## Step 3 — Serve the Portal Page from the Backend

The backend needs to serve `index.html` at the root so the redirect lands on the payment page.
This is already configured — the backend serves `GET /portal` which returns the login page.

## Step 4 — DHCP Setup (so devices get an IP)

```
/ip pool add name=hotspot-pool ranges=192.168.10.2-192.168.10.254
/ip dhcp-server add name=hotspot interface=wlan1 address-pool=hotspot-pool
/ip dhcp-server network add address=192.168.10.0/24 gateway=192.168.10.1 dns-server=8.8.8.8
/ip address add address=192.168.10.1/24 interface=wlan1
```

## Step 5 — DNS Redirect (for HTTPS captive portal detection)

Modern phones detect captive portals by making HTTP requests to known URLs.
Add this so phones show the "Sign in to network" prompt automatically:

```
/ip dns static add name=connectivitycheck.gstatic.com address=YOUR_SERVER_IP
/ip dns static add name=captive.apple.com address=YOUR_SERVER_IP
/ip dns static add name=www.msftconnecttest.com address=YOUR_SERVER_IP
```

This makes Android, iOS, and Windows automatically show the "Sign in to network" notification.

## Step 6 — Update Backend URL in packages.html

In `frontend/public/packages.html`, update:
```js
const API_BASE_URL = "http://YOUR_SERVER_IP:3000/api";
```

## Summary

| What                        | Where                        |
|-----------------------------|------------------------------|
| WiFi redirect               | MikroTik firewall NAT rule   |
| "Sign in to network" prompt | MikroTik DNS static entries  |
| Payment page                | frontend/public/index.html   |
| Payment processing          | backend/server.js            |
| IP whitelisting             | MikroTik address-list        |
| Session expiry              | Backend auto-removes after duration |
