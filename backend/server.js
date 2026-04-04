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
const locationRoutes = require('./routes/locations');
const packageRoutes = require('./routes/packages');

app.use('/api/connection', connectionRoutes);
app.use('/api/webhooks', webhookRoutes);
app.use('/api/mpesa', mpesaRoutes);
app.use('/api/locations', locationRoutes);
app.use('/api/packages', packageRoutes);

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

// Serve frontend static files
const FRONTEND_PATH = path.join(__dirname, '../frontend');
app.use('/static', express.static(path.join(FRONTEND_PATH, 'assets')));

// Captive portal — MikroTik redirects all HTTP traffic here
// Handles Apple/Android/Windows captive portal detection probes too
const PORTAL_REDIRECT = `http://${process.env.SERVER_IP || 'localhost'}:${process.env.PORT || 3000}/portal`;

const captiveProbes = [
  '/hotspot-detect.html',           // Apple iOS/macOS
  '/library/test/success.html',     // Apple
  '/connecttest.txt',               // Windows
  '/redirect',                      // Windows
  '/ncsi.txt',                      // Windows NCSI
  '/generate_204',                  // Android
  '/gen_204',                       // Android
  '/mobile/status.php',             // Android
];

captiveProbes.forEach(probe => {
  app.get(probe, (req, res) => {
    // Return unexpected response so OS detects captive portal
    // and shows "Sign in to network" notification
    res.redirect(302, PORTAL_REDIRECT);
  });
});

// Apple-specific: must return non-success body
app.get('/hotspot-detect.html', (req, res) => {
  res.status(200).send('<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>'.replace('Success', 'GORNHOM WiFi — Sign in required'));
});

// Android-specific: must NOT return 204
app.get('/generate_204', (req, res) => {
  res.redirect(302, PORTAL_REDIRECT);
});
app.get('/gen_204', (req, res) => {
  res.redirect(302, PORTAL_REDIRECT);
});

// Main portal page — serves the login/payment page
app.get('/portal', (req, res) => {
  res.sendFile(path.join(FRONTEND_PATH, 'public', 'index.html'));
});
app.get('/portal/packages', (req, res) => {
  res.sendFile(path.join(FRONTEND_PATH, 'public', 'packages.html'));
});
app.get('/portal/session', (req, res) => {
  res.sendFile(path.join(FRONTEND_PATH, 'public', 'session.html'));
});
// Serve frontend assets (logo etc.) for portal pages
app.use('/assets', express.static(path.join(FRONTEND_PATH, 'assets')));
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

// Admin stats endpoint
app.get('/api/admin/stats', (req, res) => {
  const routerSvc = require('./services/routerService');
  const connections = Array.from(routerSvc.activeConnections.values());
  const now = Date.now();

  const active = connections.filter(c => new Date(c.expiryTime).getTime() > now);
  const expired = connections.filter(c => new Date(c.expiryTime).getTime() <= now);
  const totalRevenue = connections.reduce((sum, c) => sum + (parseFloat(c.packagePrice) || 0), 0);

  // Package breakdown
  const packageCounts = {};
  connections.forEach(c => {
    packageCounts[c.packageName] = (packageCounts[c.packageName] || 0) + 1;
  });
  const topPackages = Object.entries(packageCounts)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 5)
    .map(([name, count]) => ({ name, count }));

  res.json({
    activeConnections: active.length,
    totalConnections: connections.length,
    expiredConnections: expired.length,
    totalRevenue: totalRevenue.toFixed(2),
    recentRevenue: totalRevenue.toFixed(2),
    topPackages,
    serverUptime: Math.floor(process.uptime()),
    connections: connections.map(c => ({
      phone: c.phoneNumber,
      package: c.packageName,
      packagePrice: c.packagePrice || 0,
      startTime: c.startTime,
      expiresAt: c.expiryTime,
      transactionId: c.transactionId,
      active: new Date(c.expiryTime).getTime() > now
    })).sort((a, b) => new Date(b.startTime) - new Date(a.startTime))
  });
});
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
