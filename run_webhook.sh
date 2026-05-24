#!/bin/bash
# Helper script to load environment variables from .env file and run the webhook

# Set colored output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Display banner
echo -e "${GREEN}=====================================================${NC}"
echo -e "${GREEN}    ExternalDNS Namecheap Webhook Runner Script      ${NC}"
echo -e "${GREEN}=====================================================${NC}"

# Cleanup any existing webhook processes
echo -e "${YELLOW}Cleaning up any existing webhook processes...${NC}"
pkill -f namecheap-webhook 2>/dev/null || true
sleep 1

# Check if webhook binary exists
if [ ! -f "./namecheap-webhook" ]; then
  echo -e "${YELLOW}Building webhook...${NC}"
  go build -o namecheap-webhook ./cmd/webhook
fi

# Find available ports
WEBHOOK_PORT=${WEBHOOK_PORT:-8888}
METRICS_PORT=${METRICS_PORT:-8081}

# Check if ports are in use
while netstat -tuln | grep -q ":$WEBHOOK_PORT "; do
  echo -e "${YELLOW}Port $WEBHOOK_PORT already in use, trying port $((WEBHOOK_PORT+1))...${NC}"
  WEBHOOK_PORT=$((WEBHOOK_PORT+1))
done

while netstat -tuln | grep -q ":$METRICS_PORT "; do
  echo -e "${YELLOW}Port $METRICS_PORT already in use, trying port $((METRICS_PORT+1))...${NC}"
  METRICS_PORT=$((METRICS_PORT+1))
done

echo -e "${GREEN}Using webhook port: $WEBHOOK_PORT${NC}"
echo -e "${GREEN}Using metrics port: $METRICS_PORT${NC}"

# Load environment variables from .env file
echo -e "${YELLOW}Loading environment variables from .env file...${NC}"
if [ -f .env ]; then
  # Load and export all variables from .env (set -a ensures they're exported)
  set -a
  source <(grep -v '^#' .env | grep -v '^//' | grep -v '^$')
  set +a
  
  # Verify environment variables were loaded
  echo -e "${GREEN}Loaded environment variables:${NC}"
  echo -e "NAMECHEAP_USERNAME: ${GREEN}${NAMECHEAP_USERNAME:-${YELLOW}not set}${NC}"
  echo -e "NAMECHEAP_API_USER: ${GREEN}${NAMECHEAP_API_USER:-${YELLOW}not set}${NC}"
  echo -e "NAMECHEAP_API_KEY: ${GREEN}${NAMECHEAP_API_KEY:+set [length: ${#NAMECHEAP_API_KEY}]}${YELLOW}${NAMECHEAP_API_KEY:-not set}${NC}"
  echo -e "NAMECHEAP_CLIENT_IP: ${GREEN}${NAMECHEAP_CLIENT_IP:-${YELLOW}not set}${NC}"
  echo -e "NAMECHEAP_SANDBOX: ${GREEN}${NAMECHEAP_SANDBOX:-${YELLOW}not set}${NC}"
  echo -e "DEBUG: ${GREEN}${DEBUG:-${YELLOW}not set}${NC}"

  # Check for required variables
  missing=0
  for var in "NAMECHEAP_USERNAME" "NAMECHEAP_API_USER" "NAMECHEAP_API_KEY" "NAMECHEAP_CLIENT_IP"; do
    if [ -z "${!var}" ]; then
      echo -e "${YELLOW}Warning: $var is not set. Webhook may not function correctly.${NC}"
      missing=1
    fi
  done
  
  if [ "$missing" = "1" ]; then
    echo -e "${YELLOW}Some required environment variables are missing.${NC}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      echo -e "${YELLOW}Exiting.${NC}"
      exit 1
    fi
  fi
  
  # Run the webhook
  echo ""
  echo -e "${GREEN}Starting webhook...${NC}"
  # Export the ports we found available
  export WEBHOOK_PORT=$WEBHOOK_PORT
  export METRICS_PORT=$METRICS_PORT
  
  # Show API URLs
  echo -e "${GREEN}Webhook API will be available at:${NC} http://localhost:$WEBHOOK_PORT/endpoints"
  echo -e "${GREEN}Health checks will be available at:${NC} http://localhost:$METRICS_PORT/healthz"
  echo -e "${GREEN}Readiness probe will be available at:${NC} http://localhost:$METRICS_PORT/readyz"
  echo -e "${GREEN}=====================================================${NC}"
  
  ./namecheap-webhook
else
  echo -e "${YELLOW}Error: .env file not found! Creating a sample .env file...${NC}"
  cat > .env.sample << EOF
# Namecheap API credentials
NAMECHEAP_API_KEY=your_api_key_here
NAMECHEAP_USERNAME=your_username_here
NAMECHEAP_API_USER=your_api_user_here
NAMECHEAP_CLIENT_IP=your_whitelisted_ip_here
NAMECHEAP_SANDBOX=true

# Optional configuration
DEBUG=true
DRY_RUN=false
DEFAULT_TTL=600

# Optional domain filtering
# DOMAIN_FILTER=example.com,example.org
# EXCLUDE_DOMAIN_FILTER=test.com
# REGEXP_DOMAIN_FILTER=.*\.example\.com
EOF
  echo -e "${YELLOW}Created .env.sample file. Please rename it to .env and fill in your credentials.${NC}"
  exit 1
fi
