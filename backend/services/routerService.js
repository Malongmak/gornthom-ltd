const routerConfig = require('../config/router');
const fs = require('fs').promises;
const path = require('path');

class RouterService {
  constructor() {
    this.routerType = process.env.ROUTER_TYPE || 'mikrotik';
    this.config = routerConfig[this.routerType];
    this.activeConnections = new Map(); // keyed by token
    this.phoneIndex = new Map();        // phone → token (for reconnect lookup)
  }

  async activateConnection(data) {
    switch (this.routerType) {
      case 'mikrotik': return await this.activateMikroTik(data);
      case 'openwrt':  return await this.activateOpenWrt(data);
      case 'pfsense':  return await this.activatePfSense(data);
      default:         return await this.activateGeneric(data);
    }
  }

  // ─── MikroTik ────────────────────────────────────────────────────────────────
  async activateMikroTik(data) {
    try {
      console.log(`📡 MikroTik: connecting to ${this.config.host}`);

      const simulateConnection = process.env.NODE_ENV === 'development' && !this.config.password;
      if (simulateConnection) {
        console.log(`⚠️  Simulating — would whitelist MAC:${data.macAddress || 'unknown'} IP:${data.userIP} for ${data.durationMinutes}m`);
        this.storeConnection(data, `mikrotik_${data.transactionId}`);
        return { success: true, token: `mikrotik_${data.transactionId}`, sessionId: `session_${Date.now()}`, message: 'Simulated' };
      }

      let RouterOSAPI;
      try { RouterOSAPI = require('routeros-api'); }
      catch (e) {
        this.storeConnection(data, `mikrotik_${data.transactionId}`);
        return { success: true, token: `mikrotik_${data.transactionId}`, sessionId: `session_${Date.now()}`, message: 'routeros-api not installed' };
      }

      const { RouterOSClient } = RouterOSAPI;
      const api = new RouterOSClient({
        host: this.config.host, user: this.config.username,
        password: this.config.password, port: this.config.port || 8728, timeout: 10
      });

      try {
        const client = await api.connect();
        const timeout = data.durationMinutes || 60;
        const comment = `GORNHOM|${data.phoneNumber}|${data.packageName}|${data.transactionId}`;

        // Prefer MAC address — survives IP changes on reconnect
        const identifier = data.macAddress || data.userIP;

        const addOrUpdate = async (list, address) => {
          try {
            await client.menu('/ip firewall address-list').add({ list, address, timeout: `${timeout}m`, comment });
          } catch (e) {
            if (e.message && e.message.includes('already have')) {
              const entries = await client.menu('/ip firewall address-list')
                .where('list', list).where('address', address).get();
              if (entries.length > 0) {
                await client.menu('/ip firewall address-list').where('id', entries[0].id).update({ timeout: `${timeout}m`, comment });
              }
            } else throw e;
          }
        };

        // Whitelist both MAC (for reconnect) and IP (for immediate access)
        if (data.macAddress) await addOrUpdate('allowed-macs', data.macAddress);
        await addOrUpdate('allowed-users', data.userIP);

        api.close();
        this.storeConnection(data, `mikrotik_${data.transactionId}`);
        return { success: true, token: `mikrotik_${data.transactionId}`, sessionId: `session_${Date.now()}` };
      } catch (e) { api.close(); throw e; }
    } catch (error) {
      console.error('❌ MikroTik error:', error.message);
      return { success: false, message: error.message };
    }
  }

  // ─── OpenWrt ─────────────────────────────────────────────────────────────────
  async activateOpenWrt(data) {
    try {
      const { exec } = require('child_process');
      const { promisify } = require('util');
      const execAsync = promisify(exec);

      // Use MAC-based rule if available, fall back to IP
      let rule;
      if (data.macAddress) {
        rule = `iptables -I FORWARD -m mac --mac-source ${data.macAddress} -j ACCEPT`;
      } else {
        rule = `iptables -I FORWARD -s ${data.userIP} -j ACCEPT && iptables -I FORWARD -d ${data.userIP} -j ACCEPT`;
      }

      await execAsync(`ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 ${this.config.username}@${this.config.host} "${rule}"`);
      this.storeConnection(data, `openwrt_${data.transactionId}`);
      return { success: true, token: `openwrt_${data.transactionId}`, sessionId: `session_${Date.now()}` };
    } catch (error) {
      return { success: false, message: `SSH failed: ${error.message}` };
    }
  }

  // ─── pfSense ─────────────────────────────────────────────────────────────────
  async activatePfSense(data) {
    try {
      const axios = require('axios');
      const response = await axios.post(
        `https://${this.config.host}/api/v1/firewall/rule`,
        { interface: 'lan', type: 'pass', source: data.userIP, destination: 'any',
          description: `GORNHOM|${data.phoneNumber}|${data.packageName}` },
        { auth: { username: this.config.username, password: this.config.password }, timeout: 5000 }
      );
      if (!response.data.success) throw new Error(response.data.message);
      this.storeConnection(data, `pfsense_${data.transactionId}`);
      return { success: true, token: `pfsense_${data.transactionId}`, sessionId: `session_${Date.now()}` };
    } catch (error) {
      return { success: false, message: error.message };
    }
  }

  // ─── Generic (log only) ──────────────────────────────────────────────────────
  async activateGeneric(data) {
    try {
      const logDir = path.join(__dirname, '../logs');
      await fs.mkdir(logDir, { recursive: true });
      await fs.appendFile(path.join(logDir, 'connections.log'), JSON.stringify({ timestamp: new Date().toISOString(), action: 'ACTIVATE', ...data }) + '\n');
      this.storeConnection(data, `generic_${data.transactionId}`);
      return { success: true, token: `generic_${data.transactionId}`, sessionId: `session_${Date.now()}` };
    } catch (error) {
      return { success: false, message: error.message };
    }
  }

  // ─── Reconnect: find active session by phone, re-whitelist new IP/MAC ────────
  async reconnectByPhone(phoneNumber, newIP, newMAC) {
    const token = this.phoneIndex.get(phoneNumber);
    if (!token) return { success: false, message: 'No active session for this phone number' };

    const conn = this.activeConnections.get(token);
    if (!conn) { this.phoneIndex.delete(phoneNumber); return { success: false, message: 'Session expired' }; }

    const now = Date.now();
    const remaining = new Date(conn.expiryTime).getTime() - now;
    if (remaining <= 0) {
      this.activeConnections.delete(token);
      this.phoneIndex.delete(phoneNumber);
      return { success: false, message: 'Session has expired. Please purchase a new package.' };
    }

    const remainingMinutes = Math.ceil(remaining / 60000);
    console.log(`🔄 Reconnecting ${phoneNumber} — ${remainingMinutes}m remaining`);

    // Re-activate with new IP/MAC and remaining time
    const result = await this.activateConnection({
      ...conn,
      userIP: newIP || conn.userIP,
      macAddress: newMAC || conn.macAddress,
      durationMinutes: remainingMinutes,
      transactionId: conn.transactionId // reuse same token
    });

    return {
      success: true,
      connectionToken: token,
      remainingMinutes,
      expiresAt: conn.expiryTime,
      packageName: conn.packageName,
      message: `Reconnected — ${remainingMinutes} minutes remaining`
    };
  }

  // Store session — use maxDevices from package config
  storeConnection(data, token) {
    const { getPackages } = require('./routes/packages');
    const pkg = getPackages ? getPackages().find(p => p.name === data.packageName) : null;
    const maxDevices = pkg ? pkg.maxDevices : (parseInt(process.env.MAX_DEVICES_PER_SESSION) || 1);
    const expiryTime = data.expiryTime || new Date(Date.now() + data.durationMinutes * 60 * 1000).toISOString();

    const connection = {
      token,
      userIP: data.userIP,
      macAddress: data.macAddress || null,
      phoneNumber: data.phoneNumber,
      packageName: data.packageName,
      packagePrice: data.packagePrice,
      startTime: new Date().toISOString(),
      expiryTime,
      durationMinutes: data.durationMinutes,
      transactionId: data.transactionId,
      maxDevices,
      connectedDevices: [data.macAddress || data.userIP].filter(Boolean)
    };

    this.activeConnections.set(token, connection);
    if (data.phoneNumber) this.phoneIndex.set(data.phoneNumber, token);

    // Auto-expire
    const ttl = new Date(expiryTime).getTime() - Date.now();
    if (ttl > 0) {
      setTimeout(() => {
        this.activeConnections.delete(token);
        this.phoneIndex.delete(data.phoneNumber);
        console.log(`⏰ Session expired: ${token}`);
      }, ttl);
    }
  }

  // ─── Check if a new device can join an existing session ──────────────────────
  canAddDevice(token, deviceIdentifier) {
    const conn = this.activeConnections.get(token);
    if (!conn) return { allowed: false, reason: 'Session not found' };
    if (conn.connectedDevices.includes(deviceIdentifier)) return { allowed: true, reason: 'Already connected' };
    if (conn.connectedDevices.length >= conn.maxDevices) {
      return { allowed: false, reason: `Device limit reached (max ${conn.maxDevices})` };
    }
    conn.connectedDevices.push(deviceIdentifier);
    return { allowed: true, reason: 'Device added' };
  }

  async checkConnectionStatus(token) {
    const conn = this.activeConnections.get(token);
    if (!conn) return { active: false, message: 'Session not found' };

    const remaining = new Date(conn.expiryTime).getTime() - Date.now();
    const remainingMinutes = Math.max(0, Math.floor(remaining / 60000));

    return {
      active: remainingMinutes > 0,
      remainingMinutes,
      expiresAt: conn.expiryTime,
      packageName: conn.packageName,
      userIP: conn.userIP,
      connectedDevices: conn.connectedDevices.length,
      maxDevices: conn.maxDevices
    };
  }

  async revokeConnection(identifier) {
    for (const [token, conn] of this.activeConnections.entries()) {
      if (conn.userIP === identifier || token === identifier || conn.phoneNumber === identifier) {
        this.activeConnections.delete(token);
        this.phoneIndex.delete(conn.phoneNumber);
        console.log(`🔒 Revoked session for ${conn.phoneNumber}`);
        return { success: true, message: 'Connection revoked', userIP: conn.userIP };
      }
    }
    return { success: false, message: 'Connection not found' };
  }
}

module.exports = new RouterService();
