const express = require('express');
const router = express.Router();
const routerService = require('../services/routerService');
const { body, validationResult } = require('express-validator');

// Validation middleware
const validateActivation = [
  body('transactionId').notEmpty().withMessage('Transaction ID is required'),
  body('phoneNumber').notEmpty().withMessage('Phone number is required'),
  body('packageName').notEmpty().withMessage('Package name is required'),
  body('durationMinutes').isInt({ min: 1 }).withMessage('Duration must be a positive number'),
  body('userIP').optional().isIP().withMessage('Invalid IP address')
];

// Activate internet connection
router.post('/activate', validateActivation, async (req, res) => {
  try {
    // Check validation errors
    const errors = validationResult(req);
    if (!errors.isEmpty()) {
      return res.status(400).json({
        success: false,
        message: 'Validation failed',
        errors: errors.array()
      });
    }

    const {
      transactionId,
      phoneNumber,
      businessPhone,
      recipientPhone,
      packageName,
      packagePrice,
      packageCurrency,
      durationMinutes,
      expiryTime,
      deviceId,
      userEmail,
      paymentMethod
    } = req.body;
    
    // Business phone number for receiving all subscription payments
    const BUSINESS_PHONE = "+254116465399";

    // Get user's IP address
    const userIP = req.body.userIP || 
                   req.headers['x-forwarded-for']?.split(',')[0] || 
                   req.ip || 
                   req.connection.remoteAddress;

    // Try to resolve MAC address from ARP table (works when backend is on same network as router)
    let macAddress = req.body.macAddress || null;
    if (!macAddress && userIP) {
      try {
        const { exec } = require('child_process');
        const { promisify } = require('util');
        const execAsync = promisify(exec);
        const { stdout } = await execAsync(`arp -n ${userIP} 2>/dev/null || true`);
        const match = stdout.match(/([0-9a-f]{2}[:-]){5}[0-9a-f]{2}/i);
        if (match) macAddress = match[0];
      } catch (e) { /* ARP lookup failed, continue without MAC */ }
    }

    console.log('\n🌐 ===== CONNECTION ACTIVATION REQUEST =====');
    console.log(`📞 Customer Phone: ${phoneNumber}`);
    console.log(`📞 Business Phone: ${BUSINESS_PHONE} (Payment Recipient)`);
    console.log(`📦 Package: ${packageName}`);
    console.log(`💰 Price: ${packageCurrency || 'KES'} ${packagePrice}`);
    console.log(`⏱️  Duration: ${durationMinutes} minutes`);
    console.log(`🖥️  User IP: ${userIP}`);
    console.log(`🆔 Transaction: ${transactionId}`);
    console.log('==========================================\n');

    // Activate connection via router
    const result = await routerService.activateConnection({
      userIP: userIP,
      macAddress: macAddress,
      phoneNumber: phoneNumber,
      businessPhone: businessPhone || BUSINESS_PHONE, // Payment recipient
      recipientPhone: recipientPhone || BUSINESS_PHONE, // All payments go to this number
      packageName: packageName,
      packagePrice: packagePrice,
      packageCurrency: packageCurrency,
      durationMinutes: parseInt(durationMinutes),
      expiryTime: expiryTime,
      transactionId: transactionId,
      deviceId: deviceId,
      userEmail: userEmail,
      paymentMethod: paymentMethod || 'paystack'
    });

    if (result.success) {
      console.log('✅ Connection activated successfully');
      console.log(`   Token: ${result.token}`);
      console.log(`   Session: ${result.sessionId}\n`);

      res.json({
        success: true,
        connectionToken: result.token || `conn_${transactionId}`,
        sessionId: result.sessionId || `session_${Date.now()}`,
        message: 'Internet connection activated successfully',
        expiresAt: expiryTime || new Date(Date.now() + durationMinutes * 60 * 1000).toISOString(),
        userIP: userIP
      });
    } else {
      console.error('❌ Connection activation failed:', result.message);
      res.status(500).json({
        success: false,
        message: result.message || 'Failed to activate connection',
        error: result.error
      });
    }
  } catch (error) {
    console.error('❌ Connection activation error:', error);
    res.status(500).json({
      success: false,
      message: 'Internal server error',
      error: process.env.NODE_ENV === 'development' ? error.message : undefined
    });
  }
});

// Check connection status
router.get('/status', async (req, res) => {
  try {
    const token = req.headers.authorization?.replace('Bearer ', '') || 
                  req.query.token;
    
    if (!token) {
      return res.status(401).json({ 
        active: false, 
        message: 'No token provided' 
      });
    }

    const status = await routerService.checkConnectionStatus(token);
    res.json(status);
  } catch (error) {
    console.error('Status check error:', error);
    res.status(500).json({ 
      active: false, 
      message: 'Error checking status' 
    });
  }
});

// Revoke connection (for testing/admin)
router.post('/revoke', async (req, res) => {
  try {
    const { userIP, token } = req.body;
    
    if (!userIP && !token) {
      return res.status(400).json({
        success: false,
        message: 'IP address or token required'
      });
    }

    const result = await routerService.revokeConnection(userIP || token);
    res.json(result);
  } catch (error) {
    console.error('Revoke connection error:', error);
    res.status(500).json({
      success: false,
      message: 'Error revoking connection'
    });
  }
});

// Reconnect — user returns after disconnect, reclaim remaining session time
router.post('/reconnect', async (req, res) => {
  try {
    const { phoneNumber, userIP, macAddress } = req.body;
    if (!phoneNumber) return res.status(400).json({ success: false, message: 'phoneNumber is required' });

    const ip = userIP || req.headers['x-forwarded-for']?.split(',')[0] || req.ip;
    const result = await routerService.reconnectByPhone(phoneNumber, ip, macAddress);
    res.status(result.success ? 200 : 404).json(result);
  } catch (error) {
    res.status(500).json({ success: false, message: error.message });
  }
});

// Device check — can this device join an existing session?
router.post('/device-check', async (req, res) => {
  try {
    const { token, macAddress, userIP } = req.body;
    if (!token) return res.status(400).json({ allowed: false, reason: 'token is required' });

    const deviceId = macAddress || userIP || req.ip;
    const result = routerService.canAddDevice(token, deviceId);
    res.json(result);
  } catch (error) {
    res.status(500).json({ allowed: false, reason: error.message });
  }
});

module.exports = router;
