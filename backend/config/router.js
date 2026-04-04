module.exports = {
  mikrotik: {
    host: process.env.MIKROTIK_HOST || '192.168.88.1',
    username: process.env.MIKROTIK_USER || 'admin',
    password: process.env.MIKROTIK_PASSWORD || '',
    port: parseInt(process.env.MIKROTIK_PORT || '8728')
  },
  openwrt: {
    host: process.env.OPENWRT_HOST || '192.168.1.1',
    username: process.env.OPENWRT_USER || 'root',
    password: process.env.OPENWRT_PASSWORD || '',
    sshPort: parseInt(process.env.OPENWRT_SSH_PORT || '22')
  },
  pfsense: {
    host: process.env.PFSENSE_HOST || '192.168.1.1',
    username: process.env.PFSENSE_USER || 'admin',
    password: process.env.PFSENSE_PASSWORD || '',
    apiKey: process.env.PFSENSE_API_KEY || ''
  },
  generic: {
    logPath: process.env.LOG_PATH || './logs/connections.log'
  }
};
