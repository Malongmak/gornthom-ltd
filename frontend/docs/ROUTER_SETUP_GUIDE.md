# Router API Setup Guide - Real-Time Internet Connection Activation

This guide shows you how to set up a backend server that integrates with your router to grant internet access after payment.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Backend Server Setup (Node.js)](#backend-server-setup)
3. [Router Integration Options](#router-integration-options)
4. [Step-by-Step Setup](#step-by-step-setup)
5. [Testing](#testing)

---

## Prerequisites

- A router that supports API access (MikroTik, OpenWrt, pfSense, etc.)
- Node.js installed (v14 or higher)
- Access to router admin panel
- Basic knowledge of networking

---

## Backend Server Setup

### Option 1: Node.js/Express Server

Create a new directory and set up your backend:

```bash
mkdir gornhom-backend
cd gornhom-backend
npm init -y
npm install express cors dotenv axios
npm install --save-dev nodemon
```

### Project Structure

```
gornhom-backend/
├── server.js
├── routes/
│   ├── connection.js
│   └── webhooks.js
├── services/
│   └── routerService.js
├── config/
│   └── router.js
├── .env
└── package.json
```

---

## Complete Backend Server Code

### 1. Create `server.js`

```javascript
const express = require('express');
const cors = require('cors');
require('dotenv').config();

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(cors());
app.use(express.json());

// Routes
const connectionRoutes = require('./routes/connection');
const webhookRoutes = require('./routes/webhooks');

app.use('/api/connection', connectionRoutes);
app.use('/api/webhooks', webhookRoutes);

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', message: 'GORNHOM Backend API is running' });
});

app.listen(PORT, () => {
  console.log(`🚀 Server running on port ${PORT}`);
  console.log(`📡 Router API: ${process.env.ROUTER_TYPE || 'Not configured'}`);
});
```

### 2. Create `routes/connection.js`

```javascript
const express = require('express');
const router = express.Router();
const routerService = require('../services/routerService');

// Activate internet connection
router.post('/activate', async (req, res) => {
  try {
    const {
      transactionId,
      phoneNumber,
      packageName,
      packagePrice,
      durationMinutes,
      expiryTime,
      deviceId,
      userEmail
    } = req.body;

    // Validate required fields
    if (!transactionId || !phoneNumber || !durationMinutes) {
      return res.status(400).json({
        success: false,
        message: 'Missing required fields'
      });
    }

    // Get user's IP address (from deviceId or request)
    const userIP = deviceId || req.ip || req.headers['x-forwarded-for'] || req.connection.remoteAddress;

    console.log(`🌐 Activating connection for: ${phoneNumber}`);
    console.log(`📦 Package: ${packageName}, Duration: ${durationMinutes} minutes`);

    // Activate connection via router
    const result = await routerService.activateConnection({
      userIP: userIP,
      phoneNumber: phoneNumber,
      packageName: packageName,
      durationMinutes: durationMinutes,
      expiryTime: expiryTime,
      transactionId: transactionId
    });

    if (result.success) {
      res.json({
        success: true,
        connectionToken: result.token || `conn_${transactionId}`,
        sessionId: result.sessionId || `session_${Date.now()}`,
        message: 'Internet connection activated successfully',
        expiresAt: expiryTime
      });
    } else {
      res.status(500).json({
        success: false,
        message: result.message || 'Failed to activate connection'
      });
    }
  } catch (error) {
    console.error('Connection activation error:', error);
    res.status(500).json({
      success: false,
      message: 'Internal server error'
    });
  }
});

// Check connection status
router.get('/status', async (req, res) => {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '');
    if (!token) {
      return res.status(401).json({ active: false, message: 'No token provided' });
    }

    const status = await routerService.checkConnectionStatus(token);
    res.json(status);
  } catch (error) {
    console.error('Status check error:', error);
    res.status(500).json({ active: false, message: 'Error checking status' });
  }
});

module.exports = router;
```

### 3. Create `services/routerService.js`

This file contains router-specific implementations. Choose the one that matches your router:

```javascript
const routerConfig = require('../config/router');
const axios = require('axios');

class RouterService {
  constructor() {
    this.routerType = process.env.ROUTER_TYPE || 'mikrotik';
    this.config = routerConfig[this.routerType];
  }

  async activateConnection(data) {
    switch (this.routerType) {
      case 'mikrotik':
        return await this.activateMikroTik(data);
      case 'openwrt':
        return await this.activateOpenWrt(data);
      case 'pfsense':
        return await this.activatePfSense(data);
      case 'generic':
        return await this.activateGeneric(data);
      default:
        return { success: false, message: 'Unsupported router type' };
    }
  }

  // MikroTik RouterOS Integration
  async activateMikroTik(data) {
    try {
      const RouterOSAPI = require('routeros-api');
      const connection = RouterOSAPI.connect({
        host: this.config.host,
        user: this.config.username,
        password: this.config.password,
        port: this.config.port || 8728
      });

      return new Promise((resolve, reject) => {
        connection.on('error', (err) => {
          console.error('MikroTik connection error:', err);
          reject({ success: false, message: err.message });
        });

        connection.on('connected', () => {
          // Add user to firewall address list
          const timeout = data.durationMinutes || 60;
          
          connection.write('/ip/firewall/address-list/add', {
            list: 'allowed-users',
            address: data.userIP,
            timeout: `${timeout}m`,
            comment: `Package: ${data.packageName} | Phone: ${data.phoneNumber} | TXN: ${data.transactionId}`
          }, (err, result) => {
            connection.close();
            
            if (err) {
              console.error('MikroTik API error:', err);
              resolve({ success: false, message: err.message });
            } else {
              console.log('✅ MikroTik: User added to whitelist');
              resolve({
                success: true,
                token: `mikrotik_${data.transactionId}`,
                sessionId: result.id || `session_${Date.now()}`
              });
            }
          });
        });
      });
    } catch (error) {
      console.error('MikroTik activation error:', error);
      return { success: false, message: error.message };
    }
  }

  // OpenWrt/LuCI Integration (via SSH)
  async activateOpenWrt(data) {
    try {
      const { exec } = require('child_process');
      const { promisify } = require('util');
      const execAsync = promisify(exec);

      // SSH command to add firewall rule
      const sshCommand = `ssh -o StrictHostKeyChecking=no ${this.config.username}@${this.config.host} \
        "iptables -I FORWARD -s ${data.userIP} -j ACCEPT && \
         iptables -I FORWARD -d ${data.userIP} -j ACCEPT && \
         echo 'Connection activated for ${data.userIP}'"`;

      const { stdout, stderr } = await execAsync(sshCommand);

      if (stderr && !stderr.includes('Warning')) {
        throw new Error(stderr);
      }

      console.log('✅ OpenWrt: Firewall rules added');
      return {
        success: true,
        token: `openwrt_${data.transactionId}`,
        sessionId: `session_${Date.now()}`
      };
    } catch (error) {
      console.error('OpenWrt activation error:', error);
      return { success: false, message: error.message };
    }
  }

  // pfSense Integration (via API)
  async activatePfSense(data) {
    try {
      // pfSense API call to add firewall rule
      const response = await axios.post(
        `https://${this.config.host}/api/v1/firewall/rule`,
        {
          interface: 'lan',
          type: 'pass',
          source: data.userIP,
          destination: 'any',
          description: `GORNHOM: ${data.packageName} - ${data.phoneNumber}`
        },
        {
          auth: {
            username: this.config.username,
            password: this.config.password
          },
          headers: {
            'Content-Type': 'application/json'
          }
        }
      );

      if (response.data.success) {
        console.log('✅ pfSense: Firewall rule added');
        return {
          success: true,
          token: `pfsense_${data.transactionId}`,
          sessionId: response.data.id || `session_${Date.now()}`
        };
      } else {
        throw new Error(response.data.message || 'Failed to add rule');
      }
    } catch (error) {
      console.error('pfSense activation error:', error);
      return { success: false, message: error.message };
    }
  }

  // Generic Router (Manual IP Whitelist)
  async activateGeneric(data) {
    // For routers without API, you can:
    // 1. Log the IP for manual whitelisting
    // 2. Send email notification
    // 3. Store in database for cron job to process
    
    console.log(`📝 Generic Router: Whitelist IP ${data.userIP} for ${data.durationMinutes} minutes`);
    console.log(`   Package: ${data.packageName}, Phone: ${data.phoneNumber}`);
    
    // Store in database (implement your database logic here)
    // await db.saveConnection(data);
    
    return {
      success: true,
      token: `generic_${data.transactionId}`,
      sessionId: `session_${Date.now()}`,
      message: 'Connection request logged. Please whitelist manually or configure automated system.'
    };
  }

  async checkConnectionStatus(token) {
    // Implement status check based on your router type
    // This is a simplified version
    return {
      active: true,
      remainingMinutes: 60,
      expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString()
    };
  }
}

module.exports = new RouterService();
```

### 4. Create `config/router.js`

```javascript
module.exports = {
  mikrotik: {
    host: process.env.MIKROTIK_HOST || '192.168.88.1',
    username: process.env.MIKROTIK_USER || 'admin',
    password: process.env.MIKROTIK_PASSWORD || '',
    port: process.env.MIKROTIK_PORT || 8728
  },
  openwrt: {
    host: process.env.OPENWRT_HOST || '192.168.1.1',
    username: process.env.OPENWRT_USER || 'root',
    password: process.env.OPENWRT_PASSWORD || '',
    sshPort: process.env.OPENWRT_SSH_PORT || 22
  },
  pfsense: {
    host: process.env.PFSENSE_HOST || '192.168.1.1',
    username: process.env.PFSENSE_USER || 'admin',
    password: process.env.PFSENSE_PASSWORD || '',
    apiKey: process.env.PFSENSE_API_KEY || ''
  },
  generic: {
    // For routers without API support
    logPath: process.env.LOG_PATH || './logs/connections.log'
  }
};
```

### 5. Create `.env` file

```env
# Server Configuration
PORT=3000
NODE_ENV=production

# Router Configuration
ROUTER_TYPE=mikrotik

# MikroTik Settings
MIKROTIK_HOST=192.168.88.1
MIKROTIK_USER=admin
MIKROTIK_PASSWORD=your_router_password
MIKROTIK_PORT=8728

# OpenWrt Settings (if using OpenWrt)
# OPENWRT_HOST=192.168.1.1
# OPENWRT_USER=root
# OPENWRT_PASSWORD=your_password
# OPENWRT_SSH_PORT=22

# pfSense Settings (if using pfSense)
# PFSENSE_HOST=192.168.1.1
# PFSENSE_USER=admin
# PFSENSE_PASSWORD=your_password
# PFSENSE_API_KEY=your_api_key

# Paystack Webhook Secret (for verifying webhooks)
PAYSTACK_SECRET_KEY=sk_live_your_secret_key
```

### 6. Create `routes/webhooks.js` (Optional - for Paystack webhooks)

```javascript
const express = require('express');
const router = express.Router();
const crypto = require('crypto');

// Verify Paystack webhook
router.post('/paystack', async (req, res) => {
  const hash = crypto.createHmac('sha512', process.env.PAYSTACK_SECRET_KEY)
    .update(JSON.stringify(req.body))
    .digest('hex');

  if (hash !== req.headers['x-paystack-signature']) {
    return res.status(400).send('Invalid signature');
  }

  const event = req.body;
  
  if (event.event === 'charge.success') {
    const { reference, amount, customer, metadata } = event.data;
    
    console.log('✅ Payment verified:', reference);
    console.log('   Amount:', amount / 100, 'KES');
    console.log('   Customer:', customer.email);
    
    // Here you can:
    // 1. Verify the payment in your database
    // 2. Activate connection if not already activated
    // 3. Send confirmation email
    
    // The frontend already handles activation, but this is a backup
  }

  res.status(200).send('Webhook received');
});

module.exports = router;
```

### 7. Update `package.json`

```json
{
  "name": "gornhom-backend",
  "version": "1.0.0",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "cors": "^2.8.5",
    "dotenv": "^16.0.3",
    "axios": "^1.4.0",
    "routeros-api": "^0.7.0"
  },
  "devDependencies": {
    "nodemon": "^2.0.22"
  }
}
```

---

## Router-Specific Setup Instructions

### MikroTik RouterOS

1. **Enable API on Router:**
   ```
   /ip service enable api
   /ip service set api port=8728
   ```

2. **Create Firewall Address List:**
   ```
   /ip firewall address-list add list=allowed-users comment="GORNHOM WiFi Users"
   ```

3. **Configure Firewall Rules:**
   ```
   /ip firewall filter add chain=forward src-address-list=!allowed-users action=drop
   /ip firewall filter add chain=forward dst-address-list=!allowed-users action=drop
   ```

4. **Install RouterOS API package:**
   ```bash
   npm install routeros-api
   ```

### OpenWrt Router

1. **Enable SSH on Router:**
   - Go to System → Administration → SSH Access
   - Enable SSH server

2. **Set up SSH Key Authentication:**
   ```bash
   ssh-keygen -t rsa
   ssh-copy-id root@192.168.1.1
   ```

3. **Configure Firewall Rules:**
   The script will automatically add iptables rules via SSH

### pfSense Router

1. **Enable API:**
   - System → API → Enable API
   - Create API user with appropriate permissions

2. **Install pfSense API client:**
   ```bash
   npm install pfsense-api-client
   ```

### Generic Router (No API)

For routers without API support:

1. **Manual Whitelist Method:**
   - Log IP addresses to a file
   - Manually add to router admin panel
   - Or use a cron job to process logs

2. **Email Notification:**
   - Send email with IP to whitelist
   - Admin manually adds to router

---

## Step-by-Step Setup

### Step 1: Install Dependencies

```bash
cd gornhom-backend
npm install
```

### Step 2: Configure Router

1. Choose your router type
2. Update `.env` file with router credentials
3. Set `ROUTER_TYPE` in `.env`

### Step 3: Start Server

```bash
# Development mode (with auto-reload)
npm run dev

# Production mode
npm start
```

### Step 4: Update Frontend

In `packages.html`, update the API URL:

```javascript
const API_BASE_URL = "http://your-server-ip:3000/api";
// Or if using domain:
// const API_BASE_URL = "https://api.yourdomain.com/api";
```

### Step 5: Test Connection

```bash
# Test health endpoint
curl http://localhost:3000/health

# Test activation endpoint
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

---

## Security Best Practices

1. **Use HTTPS in Production:**
   ```bash
   npm install express-sslify
   # Configure SSL certificate
   ```

2. **Add Authentication:**
   ```javascript
   // Add API key authentication
   const apiKey = process.env.API_KEY;
   app.use('/api', (req, res, next) => {
     if (req.headers['x-api-key'] !== apiKey) {
       return res.status(401).json({ error: 'Unauthorized' });
     }
     next();
   });
   ```

3. **Rate Limiting:**
   ```bash
   npm install express-rate-limit
   ```

4. **Input Validation:**
   ```bash
   npm install express-validator
   ```

---

## Troubleshooting

### Common Issues

1. **Cannot connect to router:**
   - Check router IP address
   - Verify credentials
   - Check firewall rules
   - Ensure API/SSH is enabled

2. **Connection not activating:**
   - Check router logs
   - Verify firewall rules are correct
   - Test router API manually

3. **IP address not detected:**
   - Ensure device ID is being sent
   - Check request headers
   - Use MAC address instead of IP

---

## Production Deployment

1. **Use PM2 for process management:**
   ```bash
   npm install -g pm2
   pm2 start server.js --name gornhom-backend
   pm2 save
   pm2 startup
   ```

2. **Set up reverse proxy (Nginx):**
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

3. **Enable HTTPS:**
   - Use Let's Encrypt for free SSL
   - Update frontend to use HTTPS API URL

---

## Next Steps

1. Set up your backend server
2. Configure router API access
3. Update `API_BASE_URL` in `packages.html`
4. Test with a real payment
5. Monitor connection activations

For questions or issues, check the router-specific documentation or contact support.
