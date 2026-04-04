#!/bin/bash

# GORNHOM Backend Setup Script
# This script helps you quickly set up the backend server

echo "🚀 GORNHOM Backend Setup"
echo "========================"
echo ""

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js first."
    echo "   Visit: https://nodejs.org/"
    exit 1
fi

echo "✅ Node.js version: $(node --version)"
echo ""

# Create backend directory
echo "📁 Creating backend directory..."
mkdir -p gornhom-backend
cd gornhom-backend

# Initialize npm project
echo "📦 Initializing npm project..."
npm init -y

# Install dependencies
echo "📥 Installing dependencies..."
npm install express cors dotenv axios
npm install routeros-api --save  # For MikroTik routers

# Install dev dependencies
npm install --save-dev nodemon

# Create directory structure
echo "📂 Creating directory structure..."
mkdir -p routes services config logs

# Create .env file
echo "⚙️  Creating .env file..."
cat > .env << EOF
# Server Configuration
PORT=3000
NODE_ENV=development

# Router Configuration
ROUTER_TYPE=mikrotik

# MikroTik Settings
MIKROTIK_HOST=192.168.88.1
MIKROTIK_USER=admin
MIKROTIK_PASSWORD=your_password_here
MIKROTIK_PORT=8728

# Paystack Webhook Secret
PAYSTACK_SECRET_KEY=sk_live_your_secret_key_here
EOF

echo ""
echo "✅ Backend setup complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Edit .env file with your router credentials"
echo "   2. Copy server files from ROUTER_SETUP_GUIDE.md"
echo "   3. Run: npm run dev"
echo ""
echo "📖 See ROUTER_SETUP_GUIDE.md for detailed instructions"
