/**
 * GORNHOM WiFi Billing - API Tests
 * Run with: npm test
 */

const request = require('supertest');
const crypto = require('crypto');

// Set test env before requiring app
process.env.NODE_ENV = 'development';
process.env.ROUTER_TYPE = 'generic'; // Use generic so no real router needed
process.env.PAYSTACK_SECRET_KEY = 'sk_test_dummy_key_for_testing';
process.env.PORT = '3001';

const app = require('../server');

// ─── Health Check ────────────────────────────────────────────────────────────

describe('Health Check', () => {
  test('GET /health returns 200 and status ok', async () => {
    const res = await request(app).get('/health');
    expect(res.status).toBe(200);
    expect(res.body.status).toBe('ok');
    expect(res.body.routerType).toBe('generic');
  });

  test('GET / returns API info with all endpoints', async () => {
    const res = await request(app).get('/');
    expect(res.status).toBe(200);
    expect(res.body.name).toBe('GORNHOM WiFi Backend API');
    expect(res.body.endpoints).toHaveProperty('mpesaStkPush');
    expect(res.body.endpoints).toHaveProperty('paystackWebhook');
  });
});

// ─── Connection Activation ────────────────────────────────────────────────────

describe('POST /api/connection/activate', () => {
  const validPayload = {
    transactionId: 'TXN_TEST_001',
    phoneNumber: '254712345678',
    packageName: '1 Hour',
    durationMinutes: 60,
    packagePrice: 10,
    packageCurrency: 'KES',
    userIP: '192.168.1.100'
  };

  test('activates connection with valid payload', async () => {
    const res = await request(app)
      .post('/api/connection/activate')
      .send(validPayload);

    expect(res.status).toBe(200);
    expect(res.body.success).toBe(true);
    expect(res.body.connectionToken).toBeDefined();
    expect(res.body.sessionId).toBeDefined();
    expect(res.body.expiresAt).toBeDefined();
  });

  test('returns 400 when transactionId is missing', async () => {
    const { transactionId, ...payload } = validPayload;
    const res = await request(app)
      .post('/api/connection/activate')
      .send(payload);

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });

  test('returns 400 when phoneNumber is missing', async () => {
    const { phoneNumber, ...payload } = validPayload;
    const res = await request(app)
      .post('/api/connection/activate')
      .send(payload);

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });

  test('returns 400 when durationMinutes is zero or negative', async () => {
    const res = await request(app)
      .post('/api/connection/activate')
      .send({ ...validPayload, durationMinutes: 0 });

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });

  test('returns 400 when userIP is invalid', async () => {
    const res = await request(app)
      .post('/api/connection/activate')
      .send({ ...validPayload, userIP: 'not-an-ip' });

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });
});

// ─── Connection Status ────────────────────────────────────────────────────────

describe('GET /api/connection/status', () => {
  test('returns 401 when no token provided', async () => {
    const res = await request(app).get('/api/connection/status');
    expect(res.status).toBe(401);
    expect(res.body.active).toBe(false);
  });

  test('returns inactive for unknown token', async () => {
    const res = await request(app)
      .get('/api/connection/status?token=unknown_token_xyz');
    expect(res.status).toBe(200);
    expect(res.body.active).toBe(false);
  });

  test('returns active status for a valid activated connection', async () => {
    // First activate a connection
    const activateRes = await request(app)
      .post('/api/connection/activate')
      .send({
        transactionId: 'TXN_STATUS_TEST',
        phoneNumber: '254712345678',
        packageName: '1 Day',
        durationMinutes: 1440,
        userIP: '192.168.1.101'
      });

    expect(activateRes.body.success).toBe(true);
    const token = activateRes.body.connectionToken;

    // Then check its status
    const statusRes = await request(app)
      .get(`/api/connection/status?token=${token}`);

    expect(statusRes.status).toBe(200);
    expect(statusRes.body.active).toBe(true);
    expect(statusRes.body.remainingMinutes).toBeGreaterThan(0);
    expect(statusRes.body.packageName).toBe('1 Day');
  });
});

// ─── Connection Revoke ────────────────────────────────────────────────────────

describe('POST /api/connection/revoke', () => {
  test('returns 400 when neither IP nor token provided', async () => {
    const res = await request(app)
      .post('/api/connection/revoke')
      .send({});

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });

  test('revokes an active connection by token', async () => {
    // Activate first
    const activateRes = await request(app)
      .post('/api/connection/activate')
      .send({
        transactionId: 'TXN_REVOKE_TEST',
        phoneNumber: '254712345678',
        packageName: '30 Minutes',
        durationMinutes: 30,
        userIP: '192.168.1.102'
      });

    const token = activateRes.body.connectionToken;

    const revokeRes = await request(app)
      .post('/api/connection/revoke')
      .send({ token });

    expect(revokeRes.status).toBe(200);
    expect(revokeRes.body.success).toBe(true);
  });
});

// ─── Paystack Webhook ─────────────────────────────────────────────────────────

describe('POST /api/webhooks/paystack', () => {
  function makeWebhookPayload(overrides = {}) {
    return {
      event: 'charge.success',
      data: {
        reference: 'GORNHOM_TEST_REF_001',
        amount: 6000, // KES 60 in kobo
        customer: { email: 'test@example.com' },
        metadata: {
          custom_fields: [
            { variable_name: 'package', value: '1 Day' },
            { variable_name: 'customer_phone', value: '254712345678' },
            { variable_name: 'business_phone', value: '+254116465399' }
          ]
        },
        ...overrides
      }
    };
  }

  test('returns 400 for invalid webhook signature', async () => {
    const payload = makeWebhookPayload();
    const body = Buffer.from(JSON.stringify(payload));
    const res = await request(app)
      .post('/api/webhooks/paystack')
      .set('Content-Type', 'application/json')
      .set('x-paystack-signature', 'invalid_signature')
      .send(body);

    expect(res.status).toBe(400);
  });

  test('returns 200 for valid webhook signature', async () => {
    const payload = makeWebhookPayload();
    const bodyStr = JSON.stringify(payload);
    const hash = crypto
      .createHmac('sha512', process.env.PAYSTACK_SECRET_KEY)
      .update(bodyStr)
      .digest('hex');

    // Paystack verification will fail (dummy key) but webhook still returns 200
    const res = await request(app)
      .post('/api/webhooks/paystack')
      .set('Content-Type', 'application/json')
      .set('x-paystack-signature', hash)
      .send(bodyStr);

    expect(res.status).toBe(200);
    expect(res.body.received).toBe(true);
  });
});

// ─── M-Pesa Routes ────────────────────────────────────────────────────────────

describe('POST /api/mpesa/stk-push', () => {
  test('returns 500 when M-Pesa is not configured', async () => {
    const res = await request(app)
      .post('/api/mpesa/stk-push')
      .send({ phoneNumber: '254712345678', amount: 10 });

    // Should fail because MPESA_SHORTCODE etc. are not set in test env
    expect(res.status).toBeGreaterThanOrEqual(400);
    expect(res.body.success).toBe(false);
  });

  test('returns 400 when phoneNumber is missing', async () => {
    const res = await request(app)
      .post('/api/mpesa/stk-push')
      .send({ amount: 10 });

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });

  test('returns 400 when amount is missing', async () => {
    const res = await request(app)
      .post('/api/mpesa/stk-push')
      .send({ phoneNumber: '254712345678' });

    expect(res.status).toBe(400);
    expect(res.body.success).toBe(false);
  });
});

describe('POST /api/mpesa/payment-status', () => {
  test('returns 400 when checkoutRequestId is missing', async () => {
    const res = await request(app)
      .post('/api/mpesa/payment-status')
      .send({});

    expect(res.status).toBe(400);
  });

  test('returns pending for unknown checkoutRequestId', async () => {
    const res = await request(app)
      .post('/api/mpesa/payment-status')
      .send({ checkoutRequestId: 'ws_CO_unknown_999' });

    expect(res.status).toBe(200);
    expect(res.body.status).toBe('pending');
  });
});

describe('POST /api/mpesa/callback', () => {
  test('handles successful M-Pesa callback', async () => {
    const { pendingPayments } = require('../routes/mpesa');

    // Pre-seed a pending payment
    pendingPayments.set('ws_CO_TEST_001', {
      status: 'pending',
      phoneNumber: '254712345678',
      amount: 60,
      packageInfo: { name: '1 Day', durationMinutes: 1440 },
      createdAt: Date.now()
    });

    const callbackPayload = {
      Body: {
        stkCallback: {
          CheckoutRequestID: 'ws_CO_TEST_001',
          ResultCode: 0,
          ResultDesc: 'The service request is processed successfully.',
          CallbackMetadata: {
            Item: [
              { Name: 'Amount', Value: 60 },
              { Name: 'MpesaReceiptNumber', Value: 'RGH12345XYZ' },
              { Name: 'PhoneNumber', Value: 254712345678 }
            ]
          }
        }
      }
    };

    const res = await request(app)
      .post('/api/mpesa/callback')
      .send(callbackPayload);

    expect(res.status).toBe(200);
    expect(res.body.ResultCode).toBe(0);

    // Verify payment status was updated
    const payment = pendingPayments.get('ws_CO_TEST_001');
    expect(payment.status).toBe('success');
    expect(payment.transactionId).toBe('RGH12345XYZ');
  });

  test('handles failed M-Pesa callback', async () => {
    const { pendingPayments } = require('../routes/mpesa');

    pendingPayments.set('ws_CO_TEST_002', {
      status: 'pending',
      phoneNumber: '254712345678',
      amount: 60,
      packageInfo: {},
      createdAt: Date.now()
    });

    const callbackPayload = {
      Body: {
        stkCallback: {
          CheckoutRequestID: 'ws_CO_TEST_002',
          ResultCode: 1032,
          ResultDesc: 'Request cancelled by user.'
        }
      }
    };

    const res = await request(app)
      .post('/api/mpesa/callback')
      .send(callbackPayload);

    expect(res.status).toBe(200);

    const payment = pendingPayments.get('ws_CO_TEST_002');
    expect(payment.status).toBe('failed');
    expect(payment.resultCode).toBe('1032');
  });
});

// ─── 404 Handler ─────────────────────────────────────────────────────────────

describe('404 Handler', () => {
  test('returns 404 for unknown routes', async () => {
    const res = await request(app).get('/api/nonexistent');
    expect(res.status).toBe(404);
    expect(res.body.success).toBe(false);
  });
});
