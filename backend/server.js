const express = require('express');
const cors = require('cors');
const path = require('path');
require('dotenv').config();

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Request logging
app.use((req, res, next) => {
  console.log(`${new Date().toISOString()} - ${req.method} ${req.path}`);
  next();
});

// Routes
const connectionRoutes = require('./routes/connection');
const webhookRoutes = require('./routes/webhooks');
const mpesaRoutes = require('./routes/mpesa');

app.use('/api/connection', connectionRoutes);
app.use('/api/webhooks', webhookRoutes);
app.use('/api/mpesa', mpesaRoutes);

// Paystack payment verification endpoint (called by frontend after payment)
app.get('/api/paystack/verify/:reference', async (req, res) => {
  try {
    const axios = require('axios');
    const response = await axios.get(
      `https://api.paystack.co/transaction/verify/${req.params.reference}`,
      { headers: { Authorization: `Bearer ${process.env.PAYSTACK_SECRET_KEY}` }, timeout: 10000 }
    );
    const data = response.data;
    res.json({ success: data.data?.status === 'success', status: data.data?.status, data: data.data });
  } catch (error) {
    res.status(500).json({ success: false, message: error.message });
  }
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ 
    status: 'ok', 
    message: 'GORNHOM Backend API is running',
    routerType: process.env.ROUTER_TYPE || 'not configured',
    timestamp: new Date().toISOString()
  });
});

// Root endpoint
app.get('/', (req, res) => {
  res.json({
    name: 'GORNHOM WiFi Backend API',
    version: '1.0.0',
    endpoints: {
      health: '/health',
      activate: 'POST /api/connection/activate',
      status: 'GET /api/connection/status',
      paystackWebhook: 'POST /api/webhooks/paystack',
      paystackVerify: 'GET /api/paystack/verify/:reference',
      mpesaStkPush: 'POST /api/mpesa/stk-push',
      mpesaStatus: 'POST /api/mpesa/payment-status',
      mpesaCallback: 'POST /api/mpesa/callback'
    }
  });
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('Error:', err);
  res.status(err.status || 500).json({
    success: false,
    message: err.message || 'Internal server error'
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    success: false,
    message: 'Endpoint not found'
  });
});

// Start server only when run directly (not during tests)
if (require.main === module) {
  app.listen(PORT, () => {
    console.log('🚀 GORNHOM Backend Server Started');
    console.log(`📡 Server running on port ${PORT}`);
    console.log(`🌐 Router Type: ${process.env.ROUTER_TYPE || 'Not configured'}`);
    console.log(`📋 Environment: ${process.env.NODE_ENV || 'development'}`);
    console.log(`\n✅ API Endpoints:`);
    console.log(`   Health: http://localhost:${PORT}/health`);
    console.log(`   Activate: http://localhost:${PORT}/api/connection/activate`);
    console.log(`   Status: http://localhost:${PORT}/api/connection/status`);
    console.log(`\n💡 Update API_BASE_URL in packages.html to: http://your-server-ip:${PORT}/api\n`);
  });
}

module.exports = app;
