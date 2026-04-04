const express = require('express');
const router = express.Router();
const crypto = require('crypto');
const axios = require('axios');
const routerService = require('../services/routerService');

const PACKAGE_DURATIONS = {
  '30 Minutes': 30,
  '1 Hour': 60,
  '1 Day': 1440,
  '1 Week': 10080,
  '1 Month': 43200,
  'Enterprise Plan': 43200
};

// Verify payment with Paystack before activating
async function verifyPaystackPayment(reference) {
  const response = await axios.get(
    `https://api.paystack.co/transaction/verify/${reference}`,
    {
      headers: { Authorization: `Bearer ${process.env.PAYSTACK_SECRET_KEY}` },
      timeout: 10000
    }
  );
  return response.data;
}

// Paystack webhook handler
router.post('/paystack', express.raw({ type: 'application/json' }), async (req, res) => {
  try {
    // Verify webhook signature
    const hash = crypto
      .createHmac('sha512', process.env.PAYSTACK_SECRET_KEY || '')
      .update(req.body)
      .digest('hex');

    if (hash !== req.headers['x-paystack-signature']) {
      console.error('❌ Invalid Paystack webhook signature');
      return res.status(400).send('Invalid signature');
    }

    const event = JSON.parse(req.body);

    console.log(`\n📨 Paystack Webhook | Event: ${event.event} | Ref: ${event.data?.reference}`);

    if (event.event === 'charge.success') {
      const { reference, amount, customer, metadata } = event.data;

      // Verify with Paystack API (don't trust webhook alone)
      let verified;
      try {
        verified = await verifyPaystackPayment(reference);
      } catch (err) {
        console.error('❌ Paystack verification failed:', err.message);
        return res.status(200).json({ received: true }); // Still 200 so Paystack doesn't retry
      }

      if (verified.data?.status !== 'success') {
        console.error(`❌ Payment ${reference} not confirmed by Paystack`);
        return res.status(200).json({ received: true });
      }

      console.log(`✅ Payment verified | Ref: ${reference} | Amount: KES ${amount / 100}`);

      // Extract package info from metadata
      const fields = metadata?.custom_fields || [];
      const get = (name) => fields.find(f => f.variable_name === name)?.value;

      const packageName = get('package');
      const customerPhone = get('customer_phone');
      const durationMinutes = PACKAGE_DURATIONS[packageName];

      if (packageName && durationMinutes && customerPhone) {
        // Activate connection via router
        const result = await routerService.activateConnection({
          transactionId: reference,
          phoneNumber: customerPhone,
          packageName,
          packagePrice: amount / 100,
          packageCurrency: 'KES',
          durationMinutes,
          paymentMethod: 'paystack'
        });

        if (result.success) {
          console.log(`🌐 Connection activated via webhook | Package: ${packageName} | Phone: ${customerPhone}`);
        } else {
          console.error('❌ Webhook connection activation failed:', result.message);
        }
      }
    }

    res.status(200).json({ received: true });
  } catch (error) {
    console.error('Webhook error:', error.message);
    res.status(200).json({ received: true }); // Always 200 to prevent Paystack retries
  }
});

module.exports = router;
