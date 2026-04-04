const express = require('express');
const router = express.Router();
const axios = require('axios');
const routerService = require('../services/routerService');

const MPESA_ENV = process.env.MPESA_ENV || 'sandbox';
const BASE_URL = MPESA_ENV === 'production'
  ? 'https://api.safaricom.co.ke'
  : 'https://sandbox.safaricom.co.ke';

// Get M-Pesa OAuth token
async function getMpesaToken() {
  const { MPESA_CONSUMER_KEY, MPESA_CONSUMER_SECRET } = process.env;
  const credentials = Buffer.from(`${MPESA_CONSUMER_KEY}:${MPESA_CONSUMER_SECRET}`).toString('base64');

  const response = await axios.get(`${BASE_URL}/oauth/v1/generate?grant_type=client_credentials`, {
    headers: { Authorization: `Basic ${credentials}` },
    timeout: 10000
  });
  return response.data.access_token;
}

// In-memory store for pending payments (use a DB in production)
const pendingPayments = new Map();

// POST /api/mpesa/stk-push
router.post('/stk-push', async (req, res) => {
  try {
    const { phoneNumber, amount, accountReference, transactionDesc, packageInfo } = req.body;

    if (!phoneNumber || !amount) {
      return res.status(400).json({ success: false, message: 'phoneNumber and amount are required' });
    }

    const { MPESA_SHORTCODE, MPESA_PASSKEY, MPESA_CALLBACK_URL } = process.env;

    if (!MPESA_SHORTCODE || !MPESA_PASSKEY || !MPESA_CALLBACK_URL) {
      return res.status(500).json({ success: false, message: 'M-Pesa not configured. Check .env file.' });
    }

    const token = await getMpesaToken();

    const timestamp = new Date().toISOString().replace(/[^0-9]/g, '').slice(0, 14);
    const password = Buffer.from(`${MPESA_SHORTCODE}${MPESA_PASSKEY}${timestamp}`).toString('base64');

    const stkPayload = {
      BusinessShortCode: MPESA_SHORTCODE,
      Password: password,
      Timestamp: timestamp,
      TransactionType: 'CustomerPayBillOnline',
      Amount: Math.ceil(parseFloat(amount)),
      PartyA: phoneNumber,
      PartyB: MPESA_SHORTCODE,
      PhoneNumber: phoneNumber,
      CallBackURL: MPESA_CALLBACK_URL,
      AccountReference: accountReference || 'GORNHOM',
      TransactionDesc: transactionDesc || 'WiFi Package'
    };

    const response = await axios.post(
      `${BASE_URL}/mpesa/stkpush/v1/processrequest`,
      stkPayload,
      { headers: { Authorization: `Bearer ${token}` }, timeout: 15000 }
    );

    const { CheckoutRequestID, ResponseCode, ResponseDescription } = response.data;

    if (ResponseCode !== '0') {
      return res.status(400).json({ success: false, message: ResponseDescription });
    }

    // Store pending payment with package info for later activation
    pendingPayments.set(CheckoutRequestID, {
      status: 'pending',
      phoneNumber,
      amount,
      packageInfo: packageInfo || {},
      createdAt: Date.now()
    });

    console.log(`📱 STK Push sent to ${phoneNumber} | CheckoutRequestID: ${CheckoutRequestID}`);

    res.json({ success: true, checkoutRequestId: CheckoutRequestID, message: 'STK Push sent' });
  } catch (error) {
    console.error('STK Push error:', error.response?.data || error.message);
    res.status(500).json({ success: false, message: error.response?.data?.errorMessage || error.message });
  }
});

// POST /api/mpesa/payment-status
router.post('/payment-status', async (req, res) => {
  try {
    const { checkoutRequestId } = req.body;

    if (!checkoutRequestId) {
      return res.status(400).json({ success: false, message: 'checkoutRequestId is required' });
    }

    const payment = pendingPayments.get(checkoutRequestId);

    if (!payment) {
      return res.json({ status: 'pending', message: 'Payment not yet confirmed' });
    }

    res.json({
      status: payment.status,
      resultCode: payment.resultCode,
      transactionId: payment.transactionId,
      message: payment.message
    });
  } catch (error) {
    console.error('Payment status error:', error.message);
    res.status(500).json({ success: false, message: error.message });
  }
});

// POST /api/mpesa/callback  (Safaricom calls this after payment)
router.post('/callback', async (req, res) => {
  try {
    const { Body } = req.body;
    const { stkCallback } = Body;
    const { CheckoutRequestID, ResultCode, ResultDesc, CallbackMetadata } = stkCallback;

    console.log(`\n📲 M-Pesa Callback | CheckoutRequestID: ${CheckoutRequestID} | ResultCode: ${ResultCode}`);

    const payment = pendingPayments.get(CheckoutRequestID);

    if (ResultCode === 0) {
      // Payment successful - extract transaction details
      const items = CallbackMetadata?.Item || [];
      const get = (name) => items.find(i => i.Name === name)?.Value;

      const transactionId = get('MpesaReceiptNumber');
      const amount = get('Amount');
      const phoneNumber = get('PhoneNumber')?.toString();

      console.log(`✅ M-Pesa payment confirmed | TXN: ${transactionId} | Amount: ${amount} | Phone: ${phoneNumber}`);

      if (payment) {
        payment.status = 'success';
        payment.resultCode = '0';
        payment.transactionId = transactionId;
        payment.message = 'Payment successful';

        // Auto-activate connection if package info is available
        if (payment.packageInfo?.durationMinutes) {
          const userIP = payment.packageInfo.userIP;
          if (userIP) {
            await routerService.activateConnection({
              userIP,
              phoneNumber: payment.phoneNumber,
              packageName: payment.packageInfo.name,
              packagePrice: amount,
              packageCurrency: 'KES',
              durationMinutes: payment.packageInfo.durationMinutes,
              transactionId,
              paymentMethod: 'mpesa'
            });
            console.log(`🌐 Auto-activated connection for IP: ${userIP}`);
          }
        }
      }
    } else {
      console.log(`❌ M-Pesa payment failed | ResultCode: ${ResultCode} | ${ResultDesc}`);
      if (payment) {
        payment.status = 'failed';
        payment.resultCode = String(ResultCode);
        payment.message = ResultDesc;
      }
    }

    // Always respond 200 to Safaricom
    res.status(200).json({ ResultCode: 0, ResultDesc: 'Accepted' });
  } catch (error) {
    console.error('M-Pesa callback error:', error.message);
    res.status(200).json({ ResultCode: 0, ResultDesc: 'Accepted' });
  }
});

module.exports = router;
module.exports.pendingPayments = pendingPayments;
