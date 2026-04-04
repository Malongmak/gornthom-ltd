# GORNHOM WiFi Backend API

Complete backend server for real-time internet connection activation after payment.

## 🎯 Recommended Router: **MikroTik RouterOS**

**Why MikroTik?**
- ✅ Most popular for WiFi hotspot systems
- ✅ Powerful API for automation
- ✅ Excellent firewall and bandwidth management
- ✅ Time-based access control
- ✅ Easy to configure
- ✅ Widely used in commercial WiFi deployments

## 🚀 Quick Start

### 1. Install Dependencies

```bash
cd backend
npm install
```

### 2. Configure Router

**For MikroTik RouterOS (Recommended):**

1. Enable API on router:
   ```
   /ip service enable api
   /ip service set api port=8728
   ```

2. Create firewall address list:
   ```
   /ip firewall address-list add list=allowed-users comment="GORNHOM WiFi Users"
   ```

3. Configure firewall to block non-whitelisted users:
   ```
   /ip firewall filter add chain=forward src-address-list=!allowed-users action=drop comment="Block non-whitelisted"
   /ip firewall filter add chain=forward dst-address-list=!allowed-users action=drop comment="Block non-whitelisted"
   ```

### 3. Configure Backend

Copy `.env.example` to `.env` and update:

```env
ROUTER_TYPE=mikrotik
MIKROTIK_HOST=192.168.88.1
MIKROTIK_USER=admin
MIKROTIK_PASSWORD=your_router_password
```

### 4. Start Server

```bash
npm run dev    # Development mode
npm start      # Production mode
```

### 5. Update Frontend

In `packages.html`, update:

```javascript
const API_BASE_URL = "http://your-server-ip:3000/api";
```

## 📋 API Endpoints

### Activate Connection
```
POST /api/connection/activate
Content-Type: application/json

{
  "transactionId": "GORNHOM_1234567890",
  "phoneNumber": "254712345678",
  "packageName": "1 Day",
  "packagePrice": "60",
  "packageCurrency": "KES",
  "durationMinutes": 1440,
  "userIP": "192.168.1.100"
}
```

### Check Status
```
GET /api/connection/status?token=conn_token_123
```

### Health Check
```
GET /health
```

## 🔧 Router Types Supported

1. **MikroTik RouterOS** ⭐ RECOMMENDED
   - Best for WiFi hotspots
   - Full API support
   - Automatic whitelisting

2. **OpenWrt**
   - SSH-based configuration
   - Good for custom setups

3. **pfSense**
   - Enterprise-grade
   - API support

4. **Generic**
   - Logs IPs for manual whitelisting
   - Works with any router

## 🛡️ Security

- Input validation
- Error handling
- Request logging
- Webhook signature verification

## 📝 Production Deployment

### Using PM2

```bash
npm install -g pm2
pm2 start server.js --name gornhom-backend
pm2 save
pm2 startup
```

### Using Docker (Optional)

```dockerfile
FROM node:18
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["node", "server.js"]
```

## 🐛 Troubleshooting

**Cannot connect to router:**
- Check router IP address
- Verify username/password
- Ensure API is enabled
- Check firewall rules

**Connection not activating:**
- Check router logs
- Verify address list exists
- Test API manually

## 📞 Support

See `ROUTER_SETUP_GUIDE.md` for detailed instructions.
