const routerConfig = require('../config/router');
const fs = require('fs').promises;
const path = require('path');

class RouterService {
  constructor() {
    this.routerType = process.env.ROUTER_TYPE || 'mikrotik';
    this.config = routerConfig[this.routerType];
    this.activeConnections = new Map(); // In-memory store (use database in production)
  }

  async activateConnection(data) {
    console.log(`🔧 Using router type: ${this.routerType}`);
    
    switch (this.routerType) {
      case 'mikrotik':
        return await this.activateMikroTik(data);
      case 'openwrt':
        return await this.activateOpenWrt(data);
      case 'pfsense':
        return await this.activatePfSense(data);
      case 'generic':
        return await this.activateGeneric(data);
      default:
        return { 
          success: false, 
          message: `Unsupported router type: ${this.routerType}` 
        };
    }
  }

  // MikroTik RouterOS Integration (RECOMMENDED)
  async activateMikroTik(data) {
    try {
      // Use RouterOS API via direct connection
      // For now, we'll use a simplified approach that logs the action
      // In production, you can use routeros-api-connector or implement direct API calls
      
      console.log('📡 Connecting to MikroTik router...');
      console.log(`   Host: ${this.config.host}`);
      console.log(`   User: ${this.config.username}`);
      
      // For testing without router, we'll simulate the connection
      // In production, implement actual RouterOS API connection here
      // You can use: https://www.npmjs.com/package/routeros-api-connector
      // Or implement direct API protocol
      
      // Simulate API call (replace with actual RouterOS API implementation)
      const simulateConnection = process.env.NODE_ENV === 'development' && !this.config.password;
      
      if (simulateConnection) {
        console.log('⚠️  Simulating MikroTik connection (no router configured)');
        console.log(`   Would add IP ${data.userIP} to allowed-users list`);
        console.log(`   Timeout: ${data.durationMinutes} minutes`);
        
        this.storeConnection(data, `mikrotik_${data.transactionId}`);
        return {
          success: true,
          token: `mikrotik_${data.transactionId}`,
          sessionId: `session_${Date.now()}`,
          message: 'Connection simulated (configure router for real activation)'
        };
      }
      
      // Try to use routeros-api-connector if available
      let RouterOSAPI;
      try {
        RouterOSAPI = require('routeros-api-connector');
      } catch (e) {
        // Fallback: Use generic method or direct API
        console.log('⚠️  routeros-api-connector not installed. Using generic method.');
        console.log(`📝 To enable real MikroTik integration, install: npm install routeros-api-connector`);
        console.log(`   Or configure router manually with IP: ${data.userIP}`);
        
        this.storeConnection(data, `mikrotik_${data.transactionId}`);
        return {
          success: true,
          token: `mikrotik_${data.transactionId}`,
          sessionId: `session_${Date.now()}`,
          message: 'Connection logged. Install routeros-api-connector for automatic activation.'
        };
      }
      
      // If package is available, use it
      const connection = RouterOSAPI.connect({
        host: this.config.host,
        user: this.config.username,
        password: this.config.password,
        port: this.config.port || 8728,
        timeout: 5000
      });

      return new Promise((resolve, reject) => {
        connection.on('error', (err) => {
          console.error('❌ MikroTik connection error:', err.message);
          connection.close();
          reject({ 
            success: false, 
            message: `Router connection failed: ${err.message}`,
            error: err.message
          });
        });

        connection.on('connected', () => {
          console.log('✅ Connected to MikroTik router');
          
          const timeout = data.durationMinutes || 60;
          const businessPhone = data.businessPhone || data.recipientPhone || "+254116465399";
          const comment = `GORNHOM: ${data.packageName} | Customer: ${data.phoneNumber} | Business: ${businessPhone} | TXN: ${data.transactionId}`;
          
          // Add user to firewall address list
          connection.write('/ip/firewall/address-list/add', {
            list: 'allowed-users',
            address: data.userIP,
            timeout: `${timeout}m`,
            comment: comment
          }, (err, result) => {
            connection.close();
            
            if (err) {
              console.error('❌ MikroTik API error:', err.message);
              
              // Check if IP already exists
              if (err.message && err.message.includes('already have')) {
                // Update existing entry
                connection.write('/ip/firewall/address-list/print', {
                  '?list': 'allowed-users',
                  '?address': data.userIP
                }, (updateErr, entries) => {
                  if (!updateErr && entries && entries.length > 0) {
                    const entryId = entries[0]['.id'];
                    connection.write('/ip/firewall/address-list/set', {
                      '.id': entryId,
                      timeout: `${timeout}m`,
                      comment: comment
                    }, (setErr) => {
                      connection.close();
                      if (setErr) {
                        resolve({ 
                          success: false, 
                          message: `Failed to update existing entry: ${setErr.message}` 
                        });
                      } else {
                        console.log('✅ MikroTik: Updated existing whitelist entry');
                        this.storeConnection(data, `mikrotik_${data.transactionId}`);
                        resolve({
                          success: true,
                          token: `mikrotik_${data.transactionId}`,
                          sessionId: entryId
                        });
                      }
                    });
                  } else {
                    resolve({ 
                      success: false, 
                      message: `Failed to add IP: ${err.message}` 
                    });
                  }
                });
              } else {
                resolve({ 
                  success: false, 
                  message: `Router API error: ${err.message}`,
                  error: err.message
                });
              }
            } else {
              console.log('✅ MikroTik: User added to whitelist');
              const sessionId = result[0]?.['.id'] || `session_${Date.now()}`;
              this.storeConnection(data, `mikrotik_${data.transactionId}`);
              
              resolve({
                success: true,
                token: `mikrotik_${data.transactionId}`,
                sessionId: sessionId
              });
            }
          });
        });

        // Connection timeout
        setTimeout(() => {
          connection.close();
          reject({ 
            success: false, 
            message: 'Connection timeout - router not responding' 
          });
        }, 10000);
      });
    } catch (error) {
      console.error('❌ MikroTik activation error:', error);
      return { 
        success: false, 
        message: error.message || 'Failed to connect to router',
        error: error.message
      };
    }
  }

  // OpenWrt Integration (via SSH)
  async activateOpenWrt(data) {
    try {
      const { exec } = require('child_process');
      const { promisify } = require('util');
      const execAsync = promisify(exec);

      const sshCommand = `ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 ${this.config.username}@${this.config.host} \
        "iptables -I FORWARD -s ${data.userIP} -j ACCEPT && \
         iptables -I FORWARD -d ${data.userIP} -j ACCEPT && \
         echo 'Connection activated for ${data.userIP}'"`;

      const { stdout, stderr } = await execAsync(sshCommand);

      if (stderr && !stderr.includes('Warning')) {
        throw new Error(stderr);
      }

      console.log('✅ OpenWrt: Firewall rules added');
      this.storeConnection(data, `openwrt_${data.transactionId}`);
      
      return {
        success: true,
        token: `openwrt_${data.transactionId}`,
        sessionId: `session_${Date.now()}`
      };
    } catch (error) {
      console.error('❌ OpenWrt activation error:', error.message);
      return { 
        success: false, 
        message: `SSH connection failed: ${error.message}`,
        error: error.message
      };
    }
  }

  // pfSense Integration
  async activatePfSense(data) {
    try {
      const axios = require('axios');
      
      const response = await axios.post(
        `https://${this.config.host}/api/v1/firewall/rule`,
        {
          interface: 'lan',
          type: 'pass',
          source: data.userIP,
          destination: 'any',
          description: `GORNHOM: ${data.packageName} - Customer: ${data.phoneNumber} - Business: ${data.businessPhone || data.recipientPhone || "+254116465399"}`
        },
        {
          auth: {
            username: this.config.username,
            password: this.config.password
          },
          headers: {
            'Content-Type': 'application/json'
          },
          timeout: 5000
        }
      );

      if (response.data.success) {
        console.log('✅ pfSense: Firewall rule added');
        this.storeConnection(data, `pfsense_${data.transactionId}`);
        return {
          success: true,
          token: `pfsense_${data.transactionId}`,
          sessionId: response.data.id || `session_${Date.now()}`
        };
      } else {
        throw new Error(response.data.message || 'Failed to add rule');
      }
    } catch (error) {
      console.error('❌ pfSense activation error:', error.message);
      return { 
        success: false, 
        message: `pfSense API error: ${error.message}`,
        error: error.message
      };
    }
  }

  // Generic Router (Logs for manual processing)
  async activateGeneric(data) {
    try {
      const logDir = path.join(__dirname, '../logs');
      await fs.mkdir(logDir, { recursive: true });
      
      const logFile = path.join(logDir, 'connections.log');
      const logEntry = {
        timestamp: new Date().toISOString(),
        action: 'ACTIVATE',
        ...data
      };
      
      await fs.appendFile(logFile, JSON.stringify(logEntry) + '\n');
      
      console.log(`📝 Generic Router: Connection logged to ${logFile}`);
      console.log(`   IP: ${data.userIP} | Package: ${data.packageName} | Duration: ${data.durationMinutes}min`);
      
      this.storeConnection(data, `generic_${data.transactionId}`);
      
      return {
        success: true,
        token: `generic_${data.transactionId}`,
        sessionId: `session_${Date.now()}`,
        message: 'Connection request logged. Please whitelist IP manually or configure automated system.'
      };
    } catch (error) {
      console.error('❌ Generic router logging error:', error);
      return { 
        success: false, 
        message: `Failed to log connection: ${error.message}` 
      };
    }
  }

  // Store connection in memory (use database in production)
  storeConnection(data, token) {
    const connection = {
      token: token,
      userIP: data.userIP,
      phoneNumber: data.phoneNumber,
      businessPhone: data.businessPhone || "+254116465399", // Payment recipient
      packageName: data.packageName,
      startTime: new Date().toISOString(),
      expiryTime: data.expiryTime || new Date(Date.now() + data.durationMinutes * 60 * 1000).toISOString(),
      durationMinutes: data.durationMinutes,
      transactionId: data.transactionId,
      recipientPhone: data.recipientPhone || "+254116465399" // All payments sent to this number
    };
    
    this.activeConnections.set(token, connection);
    
    // Auto-remove after expiry
    const expiryTime = new Date(connection.expiryTime).getTime() - Date.now();
    if (expiryTime > 0) {
      setTimeout(() => {
        this.activeConnections.delete(token);
        console.log(`⏰ Connection expired: ${token}`);
      }, expiryTime);
    }
  }

  async checkConnectionStatus(token) {
    const connection = this.activeConnections.get(token);
    
    if (!connection) {
      return { active: false, message: 'Connection not found' };
    }
    
    const now = new Date();
    const expiry = new Date(connection.expiryTime);
    const remainingMs = expiry.getTime() - now.getTime();
    const remainingMinutes = Math.max(0, Math.floor(remainingMs / 60000));
    
    return {
      active: remainingMinutes > 0,
      remainingMinutes: remainingMinutes,
      expiresAt: connection.expiryTime,
      packageName: connection.packageName,
      userIP: connection.userIP
    };
  }

  async revokeConnection(identifier) {
    // Find connection by IP or token
    let connection = null;
    for (const [token, conn] of this.activeConnections.entries()) {
      if (conn.userIP === identifier || token === identifier) {
        connection = conn;
        this.activeConnections.delete(token);
        break;
      }
    }
    
    if (!connection) {
      return { success: false, message: 'Connection not found' };
    }
    
    // Revoke on router (implement based on router type)
    console.log(`🔒 Revoking connection for IP: ${connection.userIP}`);
    
    return {
      success: true,
      message: 'Connection revoked',
      userIP: connection.userIP
    };
  }
}

module.exports = new RouterService();
