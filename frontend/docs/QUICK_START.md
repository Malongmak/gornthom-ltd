# Quick Start Guide - Backend Setup

## Fastest Way to Get Started

### Step 1: Install Node.js
Download and install Node.js from https://nodejs.org/ (v14 or higher)

### Step 2: Create Backend Directory

```bash
mkdir gornhom-backend
cd gornhom-backend
npm init -y
```

### Step 3: Install Dependencies

```bash
npm install express cors dotenv axios routeros-api
npm install --save-dev nodemon
```

### Step 4: Create Files

Copy the code from `ROUTER_SETUP_GUIDE.md` into these files:
- `server.js`
- `routes/connection.js`
- `services/routerService.js`
- `config/router.js`
- `.env`

### Step 5: Configure Router

Edit `.env` file:
```env
ROUTER_TYPE=mikrotik
MIKROTIK_HOST=192.168.88.1
MIKROTIK_USER=admin
MIKROTIK_PASSWORD=your_router_password
```

### Step 6: Start Server

```bash
npm run dev
```

### Step 7: Update Frontend

In `packages.html`, change:
```javascript
const API_BASE_URL = "http://localhost:3000/api";
// Or your server IP:
// const API_BASE_URL = "http://192.168.1.100:3000/api";
```

### Step 8: Test

1. Make a test payment
2. Check server logs for connection activation
3. Verify user gets internet access

---

## Router Configuration Examples

### MikroTik RouterOS (Recommended)

**On Router:**
```
/ip service enable api
/ip firewall address-list add list=allowed-users
/ip firewall filter add chain=forward src-address-list=!allowed-users action=drop
```

**In Backend `.env`:**
```env
ROUTER_TYPE=mikrotik
MIKROTIK_HOST=192.168.88.1
MIKROTIK_USER=admin
MIKROTIK_PASSWORD=your_password
```

### OpenWrt Router

**Enable SSH on router, then in backend `.env`:**
```env
ROUTER_TYPE=openwrt
OPENWRT_HOST=192.168.1.1
OPENWRT_USER=root
OPENWRT_PASSWORD=your_password
```

### Generic Router (No API)

For routers without API:
```env
ROUTER_TYPE=generic
```

The system will log IPs for manual whitelisting.

---

## Testing Your Setup

### Test 1: Health Check
```bash
curl http://localhost:3000/health
```
Should return: `{"status":"ok",...}`

### Test 2: Connection Activation
```bash
curl -X POST http://localhost:3000/api/connection/activate \
  -H "Content-Type: application/json" \
  -d '{
    "transactionId": "TEST_123",
    "phoneNumber": "254712345678",
    "packageName": "1 Day",
    "durationMinutes": 1440,
    "userIP": "192.168.1.100"
  }'
```

### Test 3: Real Payment
1. Make a payment on the website
2. Check server console for activation logs
3. Verify user can browse internet

---

## Common Issues & Solutions

### Issue: Cannot connect to router
**Solution:**
- Check router IP address
- Verify username/password
- Ensure API/SSH is enabled on router
- Check firewall rules

### Issue: Connection not activating
**Solution:**
- Check router logs
- Verify firewall address list exists
- Test router API manually
- Check user IP is correct

### Issue: IP address not detected
**Solution:**
- Ensure device sends MAC address or IP
- Check request headers
- Use router's DHCP lease table

---

## Production Deployment

### Using PM2 (Recommended)

```bash
npm install -g pm2
pm2 start server.js --name gornhom-backend
pm2 save
pm2 startup
```

### Using Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;
    
    location / {
        proxy_pass http://localhost:3000;
    }
}
```

### Enable HTTPS

Use Let's Encrypt:
```bash
sudo certbot --nginx -d api.yourdomain.com
```

---

## Support

For detailed instructions, see:
- `ROUTER_SETUP_GUIDE.md` - Complete setup guide
- `CONNECTION_API_REFERENCE.md` - API documentation

For router-specific help:
- MikroTik: https://wiki.mikrotik.com/wiki/Manual:API
- OpenWrt: https://openwrt.org/docs/guide-user/services/ssh
- pfSense: https://docs.netgate.com/pfsense/
