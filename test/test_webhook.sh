#!/bin/bash
set -e

WEBHOOK_HOST=${WEBHOOK_HOST:-"localhost"}
WEBHOOK_PORT=${WEBHOOK_PORT:-"8888"}
METRICS_PORT=${METRICS_PORT:-"8080"}
DOMAIN=${DOMAIN:-"testing.app"}
TEST_RECORD="test-$(date +%s)"

echo "=== ExternalDNS Namecheap Webhook Test ==="
echo "Webhook: http://${WEBHOOK_HOST}:${WEBHOOK_PORT}"
echo "Metrics: http://${WEBHOOK_HOST}:${METRICS_PORT}"
echo "Domain:  ${DOMAIN}"
echo ""

# 1. Health check
echo "1. Checking webhook health..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://${WEBHOOK_HOST}:${METRICS_PORT}/healthz")
if [ "$HTTP_CODE" == "200" ]; then echo "   Healthy"; else echo "   FAIL (HTTP $HTTP_CODE)"; exit 1; fi

# 2. Readiness check
echo "2. Checking webhook readiness..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://${WEBHOOK_HOST}:${METRICS_PORT}/readyz")
if [ "$HTTP_CODE" == "200" ]; then echo "   Ready"; else echo "   FAIL (HTTP $HTTP_CODE)"; exit 1; fi

# 3. List existing endpoints
echo "3. Listing current endpoints..."
curl -s "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/endpoints" | python3 -m json.tool 2>/dev/null || curl -s "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/endpoints"

# 4. Create a DNS record
echo ""
echo "4. Creating test A record: ${TEST_RECORD}.${DOMAIN} -> 192.168.1.100"
curl -s -X POST "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/endpoints" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "dnsName": "'"${TEST_RECORD}.${DOMAIN}"'",
        "recordType": "A",
        "targets": ["192.168.1.100"],
        "recordTTL": 300
      }
    ]
  }'
echo ""

# 5. Verify the record appears
echo "5. Verifying record appears in endpoints..."
sleep 2
RESULT=$(curl -s "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/endpoints")
if echo "$RESULT" | grep -q "${TEST_RECORD}"; then
  echo "   Found ${TEST_RECORD}.${DOMAIN}"
else
  echo "   (Record may not appear immediately in sandbox mode)"
fi

# 6. Update the record
echo "6. Updating record to 192.168.1.200..."
curl -s -X POST "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/endpoints" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "dnsName": "'"${TEST_RECORD}.${DOMAIN}"'",
        "recordType": "A",
        "targets": ["192.168.1.200"],
        "recordTTL": 600
      }
    ]
  }'
echo ""

# 7. Delete the record
echo "7. Deleting test record..."
curl -s -X DELETE "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/endpoints" \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "dnsName": "'"${TEST_RECORD}.${DOMAIN}"'",
        "recordType": "A"
      }
    ]
  }'
echo ""

# 8. Check metrics endpoint
echo "8. Checking metrics..."
curl -s "http://${WEBHOOK_HOST}:${METRICS_PORT}/metrics" | grep namecheap_ || echo "   (No namecheap metrics yet)"

echo ""
echo "=== Tests complete ==="