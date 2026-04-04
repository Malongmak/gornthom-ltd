# M-Pesa STK Push API Reference

This document describes the backend API endpoints required for M-Pesa STK Push integration.

## Base URL Configuration

Update the `API_BASE_URL` constant in `packages.html` with your actual backend API URL:
```javascript
const API_BASE_URL = "https://your-api-domain.com/api";
```

## Required Endpoints

### 1. Initiate STK Push

**Endpoint:** `POST /api/mpesa/stk-push`

**Request Body:**
```json
{
  "phoneNumber": "254712345678",
  "amount": 60.00,
  "accountReference": "GORNHOM_WIFI",
  "transactionDesc": "Payment for 1 Day package"
}
```

**Response (Success):**
```json
{
  "success": true,
  "checkoutRequestId": "ws_CO_191220231020440123456789",
  "message": "STK Push sent successfully"
}
```

**Response (Error):**
```json
{
  "success": false,
  "message": "Invalid phone number"
}
```

### 2. Check Payment Status

**Endpoint:** `POST /api/mpesa/payment-status`

**Request Body:**
```json
{
  "checkoutRequestId": "ws_CO_191220231020440123456789"
}
```

**Response (Pending):**
```json
{
  "status": "pending",
  "resultCode": null
}
}
```

**Response (Success):**
```json
{
  "status": "success",
  "resultCode": "0",
  "transactionId": "QGH4X5Z6Y7",
  "mpesaReceiptNumber": "QGH4X5Z6Y7",
  "message": "Payment successful"
}
```

**Response (Failed):**
```json
{
  "status": "failed",
  "resultCode": "1032",
  "resultDesc": "Request cancelled by user",
  "message": "Payment cancelled"
}
```

## M-Pesa API Integration

Your backend should integrate with Safaricom's M-Pesa Daraja API:

1. **Generate Access Token** - Authenticate with M-Pesa API
2. **Initiate STK Push** - Use the STK Push API endpoint
3. **Handle Callback** - Process M-Pesa callback to update payment status
4. **Query Status** - Optionally query payment status from M-Pesa

### M-Pesa Daraja API Documentation
- [STK Push API](https://developer.safaricom.co.ke/APIs/MpesaExpressSimulate)
- [Callback URL Setup](https://developer.safaricom.co.ke/APIs/MpesaExpressSimulate)

## Phone Number Format

Phone numbers should be in the format: `254XXXXXXXXX` (12 digits)
- Convert `0712345678` → `254712345678`
- Convert `+254712345678` → `254712345678`

## Error Handling

The frontend will:
- Poll for payment status every 3 seconds
- Timeout after 60 attempts (3 minutes)
- Display appropriate error messages
- Allow users to retry failed payments

## Security Considerations

1. **API Authentication** - Implement proper authentication (API keys, JWT tokens)
2. **HTTPS Only** - Always use HTTPS for API calls
3. **Input Validation** - Validate phone numbers and amounts server-side
4. **Rate Limiting** - Implement rate limiting to prevent abuse
5. **Callback Verification** - Verify M-Pesa callbacks are authentic

## Testing

For testing, you can use:
- M-Pesa Sandbox environment
- Test phone numbers provided by Safaricom
- Test amounts (typically 1 KES for testing)
