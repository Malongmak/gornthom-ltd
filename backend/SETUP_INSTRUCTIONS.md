# Complete Backend Setup Instructions

## 🎯 RECOMMENDED ROUTER: **MikroTik RouterOS**

**Why MikroTik is Best for This Project:**
- ✅ Industry standard for WiFi hotspot systems
- ✅ Powerful API for real-time automation
- ✅ Built-in firewall and bandwidth management
- ✅ Time-based access control (perfect for packages)
- ✅ Easy to configure and manage
- ✅ Widely used in commercial deployments
- ✅ Excellent documentation and community support

## 📦 Step 1: Install Backend

```bash
# Navigate to backend directory
cd backend

# Install all dependencies
npm install
```

## 🔧 Step 2: Configure MikroTik Router

### On Your MikroTik Router:

1. **Enable API Service:**
   ```
   /ip service enable api
   /ip service set api port=8728
   ```

2. **Create Firewall Address List:**
   ```
   /ip firewall address-list add list=allowed-users comment="GORNHOM WiFi Users"
   ```

3. **Block Non-Whitelisted Users:**
   ```
   /ip firewall filter add chain=forward src-address-list=!allowed-users action=drop comment="Block non-whitelisted"
   /ip firewall filter add chain=forward dst-address-list=!allowed-users action=drop comment="Block non-whitelisted"
   ```

4. **Verify Configuration:**
   ```
   /ip firewall address-list print
   /ip firewall filter print
   ```

## ⚙️ Step 3: Configure Backend

1. **Create `.env` file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit `.env` file with your router details:**
   ```env
   PORT=3000
   NODE_ENV=development
   ROUTER_TYPE=mikrotik
   
   MIKROTIK_HOST=192.168.88.1
   MIKROTIK_USER=admin
   MIKROTIK_PASSWORD=your_router_password
   MIKROTIK_PORT=8728
   
   PAYSTACK_SECRET_KEY=sk_live_your_secret_key
   ```

   **Important:** Replace:
   - `192.168.88.1` with your router's IP address
   - `your_router_password` with your router admin password
   - `sk_live_your_secret_key` with your Paystack secret key

## 🚀 Step 4: Start Backend Server

```bash
# Development mode (with auto-reload)
npm run dev

# Production mode
npm start
```

You should see:
```
🚀 GORNHOM Backend Server Started
📡 Server running on port 3000
🌐 Router Type: mikrotik
```

## 🔗 Step 5: Update Frontend

In `packages.html`, update the API URL:

```javascript
// Change this line (around line 601):
const API_BASE_URL = "http://your-server-ip:3000/api";

// Replace 'your-server-ip' with:
// - localhost (if testing locally)
// - Your server's IP address (if on network)
// - Your domain (if using domain)
```

**Example:**
```javascript
const API_BASE_URL = "http://192.168.1.100:3000/api";  // Local network
// OR
const API_BASE_URL = "http://localhost:3000/api";     // Same machine
// OR
const API_BASE_URL = "https://api.yourdomain.com/api"; // Production
```

## ✅ Step 6: Test the System

### Test 1: Health Check
```bash
curl http://localhost:3000/health
```

Expected response:
```json
{
  "status": "ok",
  "message": "GORNHOM Backend API is running",
  "routerType": "mikrotik"
}
```

### Test 2: Connection Activation
```bash
curl -X POST http://localhost:3000/api/connection/activate \
  -H "Content-Type: application/json" \
  -d '{
    "transactionId": "TEST_123",
    "phoneNumber": "254712345678",
    "packageName": "1 Day",
    "packagePrice": "60",
    "packageCurrency": "KES",
    "durationMinutes": 1440,
    "userIP": "192.168.1.100"
  }'
```

Expected response:
```json
{
  "success": true,
  "connectionToken": "mikrotik_TEST_123",
  "sessionId": "session_1234567890",
  "message": "Internet connection activated successfully"
}
```

### Test 3: Real Payment Flow
1. Make a payment on the website
2. Check backend console for activation logs
3. Verify user can browse internet
4. Check MikroTik router address list

## 📊 How It Works

1. **User pays** → Paystack processes payment
2. **Payment success** → Frontend calls `/api/connection/activate`
3. **Backend receives request** → Connects to MikroTik router
4. **Router adds IP** → User's IP added to `allowed-users` list
5. **Firewall allows traffic** → User gets internet access
6. **Auto-expiry** → Connection expires after package duration

## 🛡️ Security Checklist

- [ ] Use HTTPS in production
- [ ] Change default router password
- [ ] Use strong API keys
- [ ] Enable firewall on server
- [ ] Restrict API access by IP (optional)
- [ ] Regular security updates

## 🐛 Troubleshooting

### Issue: Cannot connect to router
**Solutions:**
- Check router IP address is correct
- Verify username/password
- Ensure API service is enabled
- Check firewall allows port 8728
- Test connection: `telnet router-ip 8728`

### Issue: IP not being whitelisted
**Solutions:**
- Check router logs
- Verify address list exists
- Test API manually
- Check user IP is correct

### Issue: Connection expires too early
**Solutions:**
- Verify duration calculation
- Check router time settings
- Review timeout configuration

## 📝 Production Deployment

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
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🎉 You're All Set!

Your backend is now ready to activate internet connections in real-time after payment!

For more details, see:
- `ROUTER_SETUP_GUIDE.md` - Complete router setup
- `README.md` - Backend documentation
