#!/bin/bash
set -e

# Use environment variables if set, otherwise sensible defaults
WEBHOOK_HOST=${WEBHOOK_HOST:-"localhost"}
WEBHOOK_PORT=${WEBHOOK_PORT:-"8888"}
METRICS_PORT=${METRICS_PORT:-"8080"}
DOMAIN=${DOMAIN:-"testing.app"}
TEST_RECORD="test-$(date +%s)"

echo "=== ExternalDNS Namecheap Webhook Test ==="
echo "Webhook:  http://${WEBHOOK_HOST}:${WEBHOOK_PORT}"
echo "Metrics:  http://${WEBHOOK_HOST}:${METRICS_PORT}"
echo "Domain:   ${DOMAIN}"
echo ""

# 1. Health check
echo "1. Checking webhook health..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://${WEBHOOK_HOST}:${METRICS_PORT}/healthz" 2>&1) || true
if [ "$HTTP_CODE" == "200" ]; then
  echo "   Healthy"
else
  echo "   FAIL (HTTP $HTTP_CODE)"
  echo "   Is the webhook running on metrics port ${METRICS_PORT}?"
  exit 1
fi

# 2. Readiness check
echo "2. Checking webhook readiness..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://${WEBHOOK_HOST}:${METRICS_PORT}/readyz" 2>&1) || true
if [ "$HTTP_CODE" == "200" ]; then
  echo "   Ready"
else
  echo "   NOT READY (HTTP $HTTP_CODE)"
  echo "   The webhook may still be starting up."
  exit 1
fi

# 3. Domain filter negotiation
echo "3. Checking domain filter..."
FILTER=$(curl -s "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/")
echo "   $FILTER"
echo ""

# 4. List existing records
echo "4. Listing current records..."
RECORDS=$(curl -s "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/records")
echo "$RECORDS" | python3 -m json.tool 2>/dev/null || echo "$RECORDS"
echo ""

# 5. Create a DNS record
echo "5. Creating test A record: ${TEST_RECORD}.${DOMAIN} -> 192.168.1.100"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/records" \
  -H "Content-Type: application/json" \
  -d '{
    "create": [
      {
        "dnsName": "'"${TEST_RECORD}.${DOMAIN}"'",
        "recordType": "A",
        "targets": ["192.168.1.100"],
        "recordTTL": 300
      }
    ]
  }')
echo "   Response code: $HTTP_CODE"

# 6. Verify the record appears
echo "6. Verifying record appears in records..."
sleep 3
RECORDS=$(curl -s "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/records")
if echo "$RECORDS" | grep -q "${TEST_RECORD}"; then
  echo "   Found ${TEST_RECORD}.${DOMAIN}"
  echo "$RECORDS" | python3 -m json.tool 2>/dev/null | grep -A3 "${TEST_RECORD}" || echo "$RECORDS" | grep "${TEST_RECORD}"
else
  echo "   (Record may not appear immediately due to DNS propagation)"
fi

# 7. Update the record
echo "7. Updating record to 192.168.1.200..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/records" \
  -H "Content-Type: application/json" \
  -d '{
    "updateNew": [
      {
        "dnsName": "'"${TEST_RECORD}.${DOMAIN}"'",
        "recordType": "A",
        "targets": ["192.168.1.200"],
        "recordTTL": 600
      }
    ]
  }')
echo "   Response code: $HTTP_CODE"

# 8. Delete the record
echo "8. Deleting test record..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "http://${WEBHOOK_HOST}:${WEBHOOK_PORT}/records" \
  -H "Content-Type: application/json" \
  -d '{
    "delete": [
      {
        "dnsName": "'"${TEST_RECORD}.${DOMAIN}"'",
        "recordType": "A",
        "targets": ["192.168.1.100"]
      }
    ]
  }')
echo "   Response code: $HTTP_CODE"

# 9. Check metrics endpoint
echo "9. Checking metrics..."
METRICS=$(curl -s "http://${WEBHOOK_HOST}:${METRICS_PORT}/metrics" 2>&1) || true
if echo "$METRICS" | grep -q "namecheap_"; then
  echo "$METRICS" | grep "namecheap_successful_api_calls_total\|namecheap_failed_api_calls_total"
else
  echo "   (No namecheap metrics found yet)"
fi

echo ""
echo "=== Tests complete ==="