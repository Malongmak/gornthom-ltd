const express = require('express');
const router = express.Router();
const crypto = require('crypto');

// Paystack webhook handler
router.post('/paystack', express.raw({ type: 'application/json' }), (req, res) => {
  try {
    const hash = crypto
      .createHmac('sha512', process.env.PAYSTACK_SECRET_KEY || '')
      .update(JSON.stringify(req.body))
      .digest('hex');

    if (hash !== req.headers['x-paystack-signature']) {
      console.error('❌ Invalid Paystack webhook signature');
      return res.status(400).send('Invalid signature');
    }

    const event = req.body;
    console.log('\n📨 ===== PAYSTACK WEBHOOK RECEIVED =====');
    console.log(`Event: ${event.event}`);
    console.log(`Reference: ${event.data?.reference}`);
    console.log('==========================================\n');

    if (event.event === 'charge.success') {
      const { reference, amount, customer, metadata } = event.data;
      
      console.log('✅ Payment verified via webhook');
      console.log(`   Reference: ${reference}`);
      console.log(`   Amount: ${amount / 100} ${metadata?.currency || 'KES'}`);
      console.log(`   Customer: ${customer?.email || 'N/A'}`);
      console.log(`   Package: ${metadata?.package || 'N/A'}`);
      console.log(`   Phone: ${metadata?.phone || 'N/A'}\n`);
      
      // Here you can:
      // 1. Verify payment in database
      // 2. Activate connection if not already activated
      // 3. Send confirmation email
      // 4. Update transaction status
    }

    res.status(200).json({ received: true });
  } catch (error) {
    console.error('Webhook processing error:', error);
    res.status(500).send('Webhook processing failed');
  }
});

module.exports = router;
