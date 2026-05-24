# Testing ExternalDNS Namecheap Webhook Locally

This document provides a step-by-step guide to test the ExternalDNS Namecheap webhook locally.

## Prerequisites

- Go development environment
- Namecheap account with API access
- Domain registered on Namecheap that you can use for testing
- Your IP address whitelisted in Namecheap API settings

## Setup

1. Clone the repository
   ```bash
   git clone https://github.com/yourusername/external-dns-namecheap-webhook.git
   cd external-dns-namecheap-webhook
   ```

2. Build the webhook
   ```bash
   go build -o namecheap-webhook ./cmd/webhook
   ```

3. Create a `.env` file with your Namecheap API credentials
   ```bash
   cat > .env << EOF
   NAMECHEAP_API_KEY=your_api_key
   NAMECHEAP_USERNAME=your_username
   NAMECHEAP_API_USER=your_api_user
   NAMECHEAP_CLIENT_IP=your_whitelisted_ip
   NAMECHEAP_SANDBOX=true
   DEBUG=true
   EOF
   ```

4. Source the environment variables (**must use `set -a` to export them**)
   ```bash
   set -a; source .env; set +a
   ```

## Running the Webhook

```bash
./namecheap-webhook
```

The webhook will start and listen on the following endpoints:
- Webhook API: http://localhost:8888
- Health/Metrics API: http://localhost:8080

## Manual Testing

You can manually test the webhook using the following curl commands:

### 1. Check Health Status

```bash
# Check if webhook is healthy
curl http://localhost:8080/healthz

# Check if webhook is ready to serve requests
curl http://localhost:8080/readyz
```

### 2. List All Endpoints (DNS Records)

```bash
curl -s http://localhost:8888/endpoints | jq
```

### 3. Create a New DNS Record

```bash
curl -X POST http://localhost:8888/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "dnsName": "test.example.com",
        "recordType": "A",
        "targets": ["192.168.1.100"],
        "recordTTL": 300
      }
    ]
  }'
```

### 4. Update an Existing DNS Record

```bash
curl -X POST http://localhost:8888/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "dnsName": "test.example.com",
        "recordType": "A",
        "targets": ["192.168.1.200"],
        "recordTTL": 600
      }
    ]
  }'
```

### 5. Delete a DNS Record

```bash
curl -X DELETE http://localhost:8888/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "endpoints": [
      {
        "dnsName": "test.example.com",
        "recordType": "A"
      }
    ]
  }'
```

## Automated Testing

We provide an automated test script that performs all the above operations in sequence:

```bash
DOMAIN=yourdomain.com ./test/test_webhook.sh
```

The script will:
1. Check webhook health
2. List current records
3. Create a test record
4. Verify the record was created
5. Update the record
6. Delete the record
7. Verify the record was deleted

## Troubleshooting

For basic troubleshooting:

- **API Authentication Issues**: Make sure your API key and credentials are correct, and your IP is whitelisted
- **DNS Propagation Delays**: DNS changes may not be immediately visible due to DNS caching
- **Sandbox Mode**: When using `NAMECHEAP_SANDBOX=true`, make sure you're testing with domains supported in the sandbox environment
- **Rate Limiting**: Namecheap API has rate limits that might affect heavy testing

For more detailed troubleshooting guidance, refer to our comprehensive [Troubleshooting Guide](troubleshooting.md).

## Monitoring Logs

When running with `DEBUG=true`, the webhook will output detailed logs about API requests and responses. Monitor these logs to understand what's happening:

```bash
./namecheap-webhook 2>&1 | grep -E "error|Deleting|Creating|Updating"
```

## Integration with ExternalDNS

After confirming that the webhook works correctly for your domain, you can deploy it alongside ExternalDNS in a Kubernetes cluster for automatic DNS management.
