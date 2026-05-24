# Troubleshooting the ExternalDNS Namecheap Webhook

This document provides guidance for troubleshooting common issues with the ExternalDNS Namecheap Webhook.

## Environment Variables

The webhook requires several environment variables to function correctly:

- `NAMECHEAP_API_KEY` - Your Namecheap API key
- `NAMECHEAP_USERNAME` - Your Namecheap username
- `NAMECHEAP_API_USER` - The API user (often the same as your username)
- `NAMECHEAP_CLIENT_IP` - Your whitelisted IP address for API access
- `NAMECHEAP_SANDBOX` - Set to "true" for testing with the sandbox environment

These can be set directly in your environment or provided in a `.env` file.

## Port Conflicts

The webhook uses two ports:
- Webhook API Port (default: 8888) - For the ExternalDNS to communicate with the webhook
- Metrics Port (default: 8081) - For health and readiness checks

If either port is already in use, you can set alternatives using environment variables:
```bash
export WEBHOOK_PORT=9999
export METRICS_PORT=9090
```

## Testing Issues

### Empty Records List

If the webhook returns an empty records list (`{}`), this could be due to:

1. **DryRun Mode** - Check if `DRY_RUN` environment variable is set to "true"
2. **Sandbox Mode** - In sandbox mode, only test domains may be accessible
3. **Domain Filters** - The domain filter may be excluding your test domains
4. **API Authentication** - Your API credentials may be invalid or IP not whitelisted

### "Not Found" Errors

If you receive "not found" errors from the webhook endpoints:

1. Verify the webhook is running (check with `curl http://localhost:8081/healthz`)
2. Ensure you're using the correct port
3. Check that the endpoint path is correct (`/endpoints`)
4. Verify the request format matches the expected webhook format

## Common Errors

### API Authentication Issues

```
ERROR Failed to get domains: Authentication failed
```

This indicates that your Namecheap API credentials are incorrect or your IP is not whitelisted.

### Port Already in Use

```
FATA[0000] Failed to start metrics server: listen tcp 0.0.0.0:8080: bind: address already in use
```

Use a different port by setting the `METRICS_PORT` environment variable.

### Missing Environment Variables

If environment variables are missing, the webhook may start but fail when attempting to communicate with Namecheap's API.

## API Endpoint Pattern

The webhook follows the standard ExternalDNS webhook API pattern:

- `GET /endpoints` - Lists all DNS records
- `POST /endpoints` - Creates or updates DNS records
- `DELETE /endpoints` - Deletes DNS records

All endpoints expect and return data in the format defined by the ExternalDNS webhook protocol. The data structure looks like:

```json
{
  "endpoints": [
    {
      "dnsName": "subdomain.example.com",
      "recordType": "A",
      "targets": ["192.168.1.100"],
      "recordTTL": 300
    }
  ]
}
```

If you're experiencing "not found" errors, ensure you're using the correct URL path `/endpoints` (not `/endpoint` or `/api/endpoints`).

## Debugging

To enable verbose debugging output:

```bash
export DEBUG=true
```

You can then filter the logs for relevant information:

```bash
./namecheap-webhook 2>&1 | grep -E "error|deleting|creating|updating"
```

## Testing in Isolation

You can test individual components:

1. Health check:
   ```bash
   curl http://localhost:8081/healthz
   ```

2. List records:
   ```bash
   curl http://localhost:8888/endpoints
   ```

3. Create a record:
   ```bash
   curl -X POST "http://localhost:8888/endpoints" \
     -H "Content-Type: application/json" \
     -d '{"endpoints":[{"dnsName":"test.example.com","recordType":"A","targets":["192.168.1.100"],"recordTTL":300}]}'
   ```

## Running in Container

When running in a container, ensure the container exposes both the webhook port (8888) and the metrics port (8081).
