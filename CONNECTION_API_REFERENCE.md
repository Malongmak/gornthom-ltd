# Internet Connection Activation API Reference

This document describes the backend API endpoints required for real-time internet connection activation after payment.

## Base URL Configuration

Update the `API_BASE_URL` constant in `packages.html` with your actual backend API URL:
```javascript
const API_BASE_URL = "https://your-api-domain.com/api";
```

## Required Endpoints

### 1. Activate Internet Connection

**Endpoint:** `POST /api/connection/activate`

This endpoint is called automatically after successful payment to grant internet access to the user.

**Request Body:**
```json
{
  "transactionId": "GORNHOM_1234567890",
  "phoneNumber": "254712345678",
  "packageName": "1 Day",
  "packagePrice": "60",
  "packageCurrency": "KES",
  "durationMinutes": 1440,
  "expiryTime": "2024-12-20T18:30:00.000Z",
  "deviceId": "MAC_ADDRESS_OR_IP",
  "userEmail": "kerubinotheng1977@gmail.com",
  "paymentMethod": "paystack"
}
```

**Response (Success):**
```json
{
  "success": true,
  "connectionToken": "conn_token_abc123",
  "sessionId": "session_xyz789",
  "message": "Internet connection activated successfully",
  "expiresAt": "2024-12-20T18:30:00.000Z"
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Failed to activate connection"
}
```

### 2. Check Connection Status

**Endpoint:** `GET /api/connection/status`

Check if a user's connection is still active.

**Request Headers:**
```
Authorization: Bearer {connectionToken}
```

**Response (Active):**
```json
{
  "active": true,
  "remainingMinutes": 120,
  "expiresAt": "2024-12-20T18:30:00.000Z",
  "packageName": "1 Day"
}
```

**Response (Expired):**
```json
{
  "active": false,
  "message": "Connection expired"
}
```

## Backend Implementation Requirements

Your backend should:

1. **Router Integration**
   - Whitelist user's MAC address/IP in router firewall
   - Configure bandwidth limits based on package
   - Set up session timeout based on package duration

2. **Session Management**
   - Store active sessions in database
   - Track connection start time and expiry
   - Automatically revoke access when session expires

3. **Network Configuration**
   - Integrate with your router's API (e.g., OpenWrt, MikroTik, pfSense)
   - Configure firewall rules to allow internet access
   - Set up bandwidth shaping/limiting

4. **Payment Verification**
   - Verify payment with Paystack webhook
   - Only activate connection for verified payments
   - Handle payment failures gracefully

## Router Integration Examples

### MikroTik RouterOS
```javascript
// Example: Add user to firewall whitelist
const mikrotikAPI = require('routeros-api');
const connection = mikrotikAPI.connect({
  host: 'router-ip',
  user: 'admin',
  password: 'password'
});

// Add firewall rule
connection.write('/ip/firewall/address-list/add', {
  list: 'allowed-users',
  address: userIP,
  timeout: durationMinutes + 'm'
});
```

### OpenWrt/LuCI
```javascript
// Example: Configure firewall via SSH
const { exec } = require('child_process');
exec(`ssh root@router "iptables -I FORWARD -s ${userIP} -j ACCEPT"`);
```

### pfSense
```javascript
// Example: Add firewall rule via API
const response = await fetch('https://pfsense-api/rule/add', {
  method: 'POST',
  body: JSON.stringify({
    interface: 'lan',
    source: userIP,
    action: 'pass'
  })
});
```

## Webhook Configuration

### Paystack Webhook

Configure Paystack to send webhooks to:
```
POST https://your-api-domain.com/api/webhooks/paystack
```

**Webhook Payload:**
```json
{
  "event": "charge.success",
  "data": {
    "reference": "GORNHOM_1234567890",
    "amount": 6000,
    "customer": {
      "email": "kerubinotheng1977@gmail.com"
    },
    "metadata": {
      "package": "1 Day",
      "phone": "254712345678"
    }
  }
}
```

## Security Considerations

1. **Authentication**: Use API keys or JWT tokens for API authentication
2. **HTTPS Only**: Always use HTTPS for API calls
3. **Payment Verification**: Verify all payments via webhook before activating connections
4. **Rate Limiting**: Implement rate limiting to prevent abuse
5. **Input Validation**: Validate all inputs server-side
6. **MAC Address Verification**: Verify user's MAC address matches payment device

## Testing

For testing without a real router:
1. Use a mock backend that logs connection requests
2. Test with localhost API
3. Verify session storage is working correctly
4. Test connection expiry timers

## Production Deployment

1. Set up your backend server with router API access
2. Configure Paystack webhooks
3. Update `API_BASE_URL` in `packages.html`
4. Test with real payments and connections
5. Monitor connection activations and expirations
