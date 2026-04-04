const express = require('express');
const router = express.Router();

// Package definitions — source of truth for both frontend and backend
let packages = [
  { id: '30min',      name: '30 Minutes',    duration: '30 min',   durationMinutes: 30,    price: 5,    currency: 'KES', speed: '5Mbps',       tier: 'Lite',     maxDevices: 1, active: true },
  { id: '1hour',      name: '1 Hour',        duration: '1 hour',   durationMinutes: 60,    price: 10,   currency: 'KES', speed: '5Mbps',       tier: 'Basic',    maxDevices: 1, active: true },
  { id: '1day',       name: '1 Day',         duration: '24 hours', durationMinutes: 1440,  price: 60,   currency: 'KES', speed: '10Mbps',      tier: 'Standard', maxDevices: 2, active: true, popular: true },
  { id: '1week',      name: '1 Week',        duration: '7 days',   durationMinutes: 10080, price: 260,  currency: 'KES', speed: '20Mbps',      tier: 'Premium',  maxDevices: 3, active: true },
  { id: '1month',     name: '1 Month',       duration: '30 days',  durationMinutes: 43200, price: 500,  currency: 'KES', speed: 'Unlimited',   tier: 'Ultimate', maxDevices: 5, active: true },
  { id: 'enterprise', name: 'Enterprise Plan', duration: '30 days', durationMinutes: 43200, price: 2500, currency: 'KES', speed: '100Mbps',   tier: 'Business', maxDevices: 25, active: true, enterprise: true },
];

// GET /api/packages — public, used by packages.html
router.get('/', (req, res) => {
  res.json(packages.filter(p => p.active));
});

// GET /api/packages/all — admin, includes inactive
router.get('/all', (req, res) => {
  res.json(packages);
});

// PATCH /api/packages/:id — admin updates a package
router.patch('/:id', (req, res) => {
  const pkg = packages.find(p => p.id === req.params.id);
  if (!pkg) return res.status(404).json({ success: false, message: 'Package not found' });

  const allowed = ['maxDevices', 'price', 'speed', 'active', 'name'];
  allowed.forEach(field => {
    if (req.body[field] !== undefined) pkg[field] = req.body[field];
  });

  console.log(`📦 Package updated: ${pkg.name} — maxDevices: ${pkg.maxDevices}`);
  res.json({ success: true, package: pkg });
});

module.exports = router;
module.exports.getPackages = () => packages;
