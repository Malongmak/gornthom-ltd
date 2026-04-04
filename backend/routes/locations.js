const express = require('express');
const router = express.Router();

// In-memory store — replace with a DB in production
let locations = [
  { id: 'GH-NRB-001', name: 'Nairobi Central Hub',   status: 'online',  region: 'Nairobi' },
  { id: 'GH-MBA-042', name: 'Mombasa Coastal Link',   status: 'online',  region: 'Mombasa' },
  { id: 'GH-KIS-109', name: 'Kisumu West Station',    status: 'offline', region: 'Kisumu'  },
  { id: 'GH-ELD-215', name: 'Eldoret Tech Park',      status: 'online',  region: 'Eldoret' },
];

// GET /api/locations  — list all, enriched with live session data
router.get('/', (req, res) => {
  const routerSvc = require('../services/routerService');
  const connections = Array.from(routerSvc.activeConnections.values());
  const now = Date.now();

  const enriched = locations.map(loc => {
    // Count active sessions whose phone area code loosely maps to region
    // In production you'd store locationId on each session
    const activeUsers = connections.filter(c => new Date(c.expiryTime).getTime() > now).length;
    const revenue = connections
      .filter(c => new Date(c.expiryTime).getTime() > now)
      .reduce((s, c) => s + (parseFloat(c.packagePrice) || 0), 0);

    return {
      ...loc,
      activeUsers: loc.status === 'online' ? activeUsers : 0,
      dailyRevenue: loc.status === 'online' ? parseFloat((revenue / Math.max(locations.filter(l=>l.status==='online').length,1)).toFixed(2)) : 0,
    };
  });

  const online = enriched.filter(l => l.status === 'online');
  const totalActiveUsers = connections.filter(c => new Date(c.expiryTime).getTime() > now).length;
  const totalRevenue = connections.reduce((s, c) => s + (parseFloat(c.packagePrice) || 0), 0);

  res.json({
    locations: enriched,
    summary: {
      total: locations.length,
      online: online.length,
      offline: locations.length - online.length,
      activeUsers: totalActiveUsers,
      totalRevenue: totalRevenue.toFixed(2),
    }
  });
});

// POST /api/locations — add a new hotspot
router.post('/', (req, res) => {
  const { name, region } = req.body;
  if (!name || !region) {
    return res.status(400).json({ success: false, message: 'name and region are required' });
  }
  const id = 'GH-' + region.slice(0,3).toUpperCase() + '-' + String(Date.now()).slice(-3);
  const loc = { id, name, region, status: 'online' };
  locations.push(loc);
  console.log(`📍 New hotspot added: ${name} (${id})`);
  res.json({ success: true, location: loc });
});

// PATCH /api/locations/:id — update status
router.patch('/:id', (req, res) => {
  const loc = locations.find(l => l.id === req.params.id);
  if (!loc) return res.status(404).json({ success: false, message: 'Location not found' });
  if (req.body.status) loc.status = req.body.status;
  if (req.body.name) loc.name = req.body.name;
  res.json({ success: true, location: loc });
});

// DELETE /api/locations/:id
router.delete('/:id', (req, res) => {
  const idx = locations.findIndex(l => l.id === req.params.id);
  if (idx === -1) return res.status(404).json({ success: false, message: 'Location not found' });
  locations.splice(idx, 1);
  res.json({ success: true });
});

module.exports = router;
