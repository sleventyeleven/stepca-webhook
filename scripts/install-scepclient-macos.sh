#!/bin/bash

# SCEP Client Installation Script for macOS
# This script installs micromdm/scepclient and configures it for Step CA

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== SCEP Client Installation for macOS ===${NC}\n"

# Check if Homebrew is installed
if ! command -v brew &> /dev/null; then
    echo -e "${RED}Error: Homebrew is not installed${NC}"
    echo "Please install Homebrew from https://brew.sh/"
    exit 1
fi

echo -e "${GREEN}✓ Homebrew is installed${NC}\n"

# Install SCEP client
echo "Installing micromdm/scepclient..."
brew install micromdm/scepclient/scepclient

# Set up Step CA root certificate
echo -e "\n${YELLOW}Setting up Step CA root certificate...${NC}"
STEP_CA_ROOT_PATH="${HOME}/Library/Application Support/step-ca"
STEP_CA_ROOT_CERT="${STEP_CA_ROOT_PATH}/root_ca.crt"

# Create Step CA directory if it doesn't exist
mkdir -p "${STEP_CA_ROOT_PATH}"

# Check if root CA certificate already exists
if [ -f "${STEP_CA_ROOT_CERT}" ]; then
    echo -e "${GREEN}✓ Root CA certificate already exists${NC}"
else
    echo -e "${YELLOW}Please download your Step CA root certificate to:${NC}"
    echo -e "${YELLOW}${STEP_CA_ROOT_CERT}${NC}"
    echo -e "${YELLOW}You can generate it from your Step CA server:${NC}"
    echo "step ca root ${STEP_CA_ROOT_CERT}"
fi

# Create SCEP profile configuration
echo -e "\n${YELLOW}Creating SCEP profile configuration...${NC}"

# Get device hostname
DEVICE_HOSTNAME=$(hostname)

# Generate a random serial number (16 characters)
DEVICE_SERIAL=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)

# SCEP server URL (adjust based on your setup)
SCEP_SERVER_URL="https://your-webhook-server:4443/scep/challenge/${DEVICE_SERIAL}"

# Step CA URL (adjust based on your setup)
STEP_CA_URL="https://your-step-ca-server:4443"

# Create SCEP profile
cat > ~/Library/Application Support/scepclient/scep-client.yaml <<EOF
# SCEP Client Configuration
server: ${SCEP_SERVER_URL}

# CA URL (for certificate issuance)
ca_url: ${STEP_CA_URL}

# Device information
device:
  serial: ${DEVICE_SERIAL}
  hostname: ${DEVICE_HOSTNAME}

# Certificate profile
profile:
  name: "step-ca"
  challenge:
    # Challenge will be automatically handled
    enabled: true

# Certificate request settings
request:
  key_type: RSA2048
  key_size: 2048
  validity: 365

# Logging
logging:
  level: info
  file: /var/log/scepclient.log
EOF

echo -e "${GREEN}✓ SCEP client configuration created${NC}"
echo -e "${GREEN}✓ Device serial: ${DEVICE_SERIAL}${NC}\n"

# Test connection
echo -e "${YELLOW}Testing SCEP connection...${NC}"
if scepclient ping; then
    echo -e "${GREEN}✓ SCEP client is working correctly${NC}\n"
else
    echo -e "${RED}✗ SCEP client ping failed${NC}"
    echo "Please check your configuration and try again"
    exit 1
fi

# Instructions
echo -e "${GREEN}=== Installation Complete ===${NC}\n"
echo "SCEP client has been installed successfully."
echo ""
echo "Configuration file location: ~/Library/Application Support/scepclient/scep-client.yaml"
echo ""
echo "To request a certificate, run:"
echo "  scepclient request --profile step-ca"
echo ""
echo "To update the device serial in the dashboard:"
echo "  1. Navigate to the dashboard at http://localhost:8080"
echo "  2. Register the device with serial: ${DEVICE_SERIAL}"
echo "  3. Provide the passphrase used in the configuration"
echo ""
echo "To view logs:"
echo "  tail -f /var/log/scepclient.log"