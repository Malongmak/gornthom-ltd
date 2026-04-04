# Complete MikroTik RouterOS Configuration Guide

This guide will help you configure your MikroTik router to work with the GORNHOM WiFi backend for real-time internet connection activation.

## 📋 Prerequisites

- MikroTik router with RouterOS installed
- Access to router admin panel (Winbox, WebFig, or SSH)
- Router IP address (usually 192.168.88.1)
- Admin username and password

---

## 🚀 Step-by-Step Configuration

### Step 1: Access Your Router

**Option A: Using Winbox (Recommended)**
1. Download Winbox from: https://mikrotik.com/download
2. Open Winbox
3. Connect to your router IP (e.g., 192.168.88.1)
4. Login with admin credentials

**Option B: Using Web Browser**
1. Open browser
2. Go to: `http://192.168.88.1` (or your router IP)
3. Login with admin credentials

**Option C: Using SSH**
```bash
ssh admin@192.168.88.1
```

---

### Step 2: Enable API Service

The API service allows the backend to communicate with your router.

**Using Winbox/WebFig:**
1. Go to **IP → Services**
2. Find **api** service
3. Double-click to edit
4. Check **Enabled**
5. Set **Port** to `8728` (default)
6. Click **OK**

**Using Terminal/SSH:**
```bash
/ip service enable api
/ip service set api port=8728
```

**Verify:**
```bash
/ip service print
```
You should see `api` with `enabled` status.

---

### Step 3: Create Firewall Address List

This list will store all users who have paid and should have internet access.

**Using Winbox/WebFig:**
1. Go to **IP → Firewall → Address Lists**
2. Click **+** (Add New)
3. Set:
   - **List**: `allowed-users`
   - **Comment**: `GORNHOM WiFi Users`
4. Leave **Address** empty (will be added by API)
5. Click **OK**

**Using Terminal/SSH:**
```bash
/ip firewall address-list add list=allowed-users comment="GORNHOM WiFi Users"
```

**Verify:**
```bash
/ip firewall address-list print
```
You should see the `allowed-users` list.

---

### Step 4: Configure Firewall Rules

These rules will block all users except those in the `allowed-users` list.

#### Rule 1: Block Outgoing Traffic (Non-Whitelisted Users)

**Using Winbox/WebFig:**
1. Go to **IP → Firewall → Filter Rules**
2. Click **+** (Add New)
3. Configure:
   - **Chain**: `forward`
   - **Src. Address List**: `!allowed-users` (note the `!` means NOT in list)
   - **Action**: `drop`
   - **Comment**: `Block non-whitelisted users - Outgoing`
4. Click **OK**

**Using Terminal/SSH:**
```bash
/ip firewall filter add \
  chain=forward \
  src-address-list=!allowed-users \
  action=drop \
  comment="Block non-whitelisted users - Outgoing"
```

#### Rule 2: Block Incoming Traffic (Non-Whitelisted Users)

**Using Winbox/WebFig:**
1. Click **+** (Add New) again
2. Configure:
   - **Chain**: `forward`
   - **Dst. Address List**: `!allowed-users`
   - **Action**: `drop`
   - **Comment**: `Block non-whitelisted users - Incoming`
3. Click **OK**

**Using Terminal/SSH:**
```bash
/ip firewall filter add \
  chain=forward \
  dst-address-list=!allowed-users \
  action=drop \
  comment="Block non-whitelisted users - Incoming"
```

**Important:** Make sure these rules are placed **BEFORE** any rules that allow traffic.

**Verify:**
```bash
/ip firewall filter print
```
You should see both drop rules.

---

### Step 5: Configure DHCP (Optional but Recommended)

To automatically assign IP addresses and track users:

**Using Winbox/WebFig:**
1. Go to **IP → DHCP Server**
2. If not configured, create a DHCP server:
   - **Interface**: Your LAN interface (usually `ether1` or `bridge`)
   - **Address Pool**: Create a pool (e.g., `192.168.88.100-192.168.88.200`)
   - **Lease Time**: `10m` (10 minutes)

**Using Terminal/SSH:**
```bash
# Create address pool
/ip pool add name=dhcp-pool ranges=192.168.88.100-192.168.88.200

# Create DHCP server
/ip dhcp-server add \
  interface=bridge \
  address-pool=dhcp-pool \
  lease-time=10m

# Create DHCP network
/ip dhcp-server network add \
  address=192.168.88.0/24 \
  gateway=192.168.88.1 \
  dns=8.8.8.8,8.8.4.4
```

---

### Step 6: Test API Connection

Test if the backend can connect to your router:

**From your backend server:**
```bash
cd backend
node -e "
const RouterOSAPI = require('routeros-api');
const conn = RouterOSAPI.connect({
  host: '192.168.88.1',
  user: 'admin',
  password: 'your_password',
  port: 8728
});
conn.on('connected', () => {
  console.log('✅ Connected to MikroTik!');
  conn.close();
});
conn.on('error', (err) => {
  console.error('❌ Connection failed:', err.message);
});
"
```

Or test from backend:
```bash
npm run dev
# Then test: curl http://localhost:3000/health
```

---

### Step 7: Manual Test - Add IP to Whitelist

Test manually adding an IP to verify everything works:

**Using Terminal/SSH:**
```bash
# Add test IP for 60 minutes
/ip firewall address-list add \
  list=allowed-users \
  address=192.168.88.100 \
  timeout=60m \
  comment="Test connection"

# Verify it was added
/ip firewall address-list print where list=allowed-users
```

**Remove test entry:**
```bash
/ip firewall address-list remove [find address=192.168.88.100]
```

---

## 🔧 Advanced Configuration

### Bandwidth Limiting by Package

You can limit bandwidth based on package type:

**Create Queue Tree:**
```bash
# Create parent queue
/queue tree add \
  name=parent-queue \
  parent=global \
  max-limit=100M

# Create queue for 30 Minutes package (5Mbps)
/queue tree add \
  name=package-30min \
  parent=parent-queue \
  packet-mark=30min-package \
  max-limit=5M/5M

# Create queue for 1 Day package (10Mbps)
/queue tree add \
  name=package-1day \
  parent=parent-queue \
  packet-mark=1day-package \
  max-limit=10M/10M
```

**Mark packets in firewall:**
```bash
/ip firewall mangle add \
  chain=forward \
  src-address-list=allowed-users \
  action=mark-packet \
  new-packet-mark=package-1day \
  passthrough=yes
```

### Hotspot Integration (Alternative Method)

If you're using MikroTik Hotspot:

```bash
# Create hotspot user profile
/ip hotspot user profile add \
  name=paid-user \
  rate-limit=10M/10M \
  shared-users=1

# The backend can create users like this:
/ip hotspot user add \
  name=user123 \
  password=temp123 \
  profile=paid-user \
  limit-uptime=1h
```

---

## 📊 Monitoring & Management

### View Active Connections

```bash
# List all whitelisted users
/ip firewall address-list print where list=allowed-users

# Show with details
/ip firewall address-list print detail where list=allowed-users
```

### Check Firewall Statistics

```bash
# View firewall rule statistics
/ip firewall filter print stats where action=drop

# Reset counters
/ip firewall filter reset-counters-all
```

### View DHCP Leases

```bash
# See all DHCP leases
/ip dhcp-server lease print

# Find specific IP
/ip dhcp-server lease print where address=192.168.88.100
```

---

## 🛡️ Security Best Practices

### 1. Change Default Password
```bash
/user set admin password=your_strong_password
```

### 2. Create API-Only User (Recommended)
```bash
# Create user for API access only
/user add \
  name=api-user \
  password=api_strong_password \
  group=read

# Or create with write access
/user add \
  name=api-user \
  password=api_strong_password \
  group=full
```

Then use this user in your `.env` file instead of admin.

### 3. Restrict API Access by IP
```bash
# Only allow API from your backend server IP
/ip service set api \
  address=192.168.1.100/32 \
  disabled=no
```

### 4. Enable Firewall for Router Itself
```bash
# Block access to router from WAN
/ip firewall filter add \
  chain=input \
  in-interface=ether1 \
  src-address=!192.168.88.0/24 \
  action=drop \
  comment="Block WAN access to router"
```

---

## 🐛 Troubleshooting

### Issue: Cannot Connect to Router API

**Check:**
1. API service is enabled:
   ```bash
   /ip service print where name=api
   ```

2. Port is correct (default 8728):
   ```bash
   /ip service print where name=api
   ```

3. Firewall allows API port:
   ```bash
   /ip firewall filter print where dst-port=8728
   ```

4. Test connection:
   ```bash
   telnet router-ip 8728
   ```

### Issue: IP Not Being Whitelisted

**Check:**
1. Address list exists:
   ```bash
   /ip firewall address-list print where list=allowed-users
   ```

2. IP format is correct (should be like 192.168.88.100)

3. Check router logs:
   ```bash
   /log print
   ```

4. Verify API user has permissions:
   ```bash
   /user print
   ```

### Issue: Users Still Blocked After Whitelisting

**Check:**
1. Firewall rules order - drop rules must come AFTER allow rules
   ```bash
   /ip firewall filter print
   ```

2. Move drop rules to bottom:
   ```bash
   /ip firewall filter move [find comment~"Block non-whitelisted"] 0
   ```

3. Check if IP is actually in list:
   ```bash
   /ip firewall address-list print where address=192.168.88.100
   ```

### Issue: Connection Expires Too Early

**Check:**
1. Router time is correct:
   ```bash
   /system clock print
   ```

2. Timeout format is correct (should be like `1440m` for 24 hours)

3. Check address list timeout:
   ```bash
   /ip firewall address-list print detail where list=allowed-users
   ```

---

## 📝 Complete Configuration Script

Here's a complete script you can run on your MikroTik router:

```bash
# Enable API
/ip service enable api
/ip service set api port=8728

# Create address list
/ip firewall address-list add list=allowed-users comment="GORNHOM WiFi Users"

# Create firewall rules (block non-whitelisted)
/ip firewall filter add \
  chain=forward \
  src-address-list=!allowed-users \
  action=drop \
  comment="Block non-whitelisted - Outgoing"

/ip firewall filter add \
  chain=forward \
  dst-address-list=!allowed-users \
  action=drop \
  comment="Block non-whitelisted - Incoming"

# Optional: Create API-only user
/user add name=api-user password=your_api_password group=full

# Optional: Restrict API to backend server IP
/ip service set api address=192.168.1.100/32

# Verify configuration
/ip service print where name=api
/ip firewall address-list print where list=allowed-users
/ip firewall filter print where comment~"Block non-whitelisted"
```

---

## ✅ Verification Checklist

After configuration, verify:

- [ ] API service is enabled and running
- [ ] Address list `allowed-users` exists
- [ ] Firewall rules are configured correctly
- [ ] Backend can connect to router
- [ ] Test IP can be added manually
- [ ] Test IP gets internet access
- [ ] Test IP is blocked when removed from list

---

## 🎯 Next Steps

1. ✅ Configure your router using this guide
2. ✅ Update backend `.env` with router credentials
3. ✅ Start backend server: `npm run dev`
4. ✅ Update frontend `API_BASE_URL` in `packages.html`
5. ✅ Test with a real payment

---

## 📞 Need More Help?

- **MikroTik Documentation**: https://help.mikrotik.com/
- **RouterOS API**: https://wiki.mikrotik.com/wiki/Manual:API
- **MikroTik Forum**: https://forum.mikrotik.com/

For backend-specific issues, see `backend/SETUP_INSTRUCTIONS.md`
