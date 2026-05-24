# ExternalDNS Namecheap Webhook

This is an extension to [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) that enables automatic DNS record management on Namecheap.com using the [Namecheap API](https://www.namecheap.com/support/api/intro/).

## How it Works

This webhook implements the [ExternalDNS Webhook Provider](https://github.com/kubernetes-sigs/external-dns/tree/master/provider/webhook) interface, allowing ExternalDNS to:

1. List existing DNS records from your Namecheap account
2. Create new DNS records based on Kubernetes Services and Ingresses
3. Update existing DNS records
4. Delete DNS records when resources are removed

## Prerequisites

- A Namecheap account
- API access enabled for your account
- API Key for your Namecheap account
- WhiteList your IP address in Namecheap API settings

## Environment Variables

The following environment variables are required:

- `NAMECHEAP_API_KEY`: Your Namecheap API key
- `NAMECHEAP_USERNAME`: Your Namecheap username
- `NAMECHEAP_API_USER`: The Namecheap API user (usually the same as your username)
- `NAMECHEAP_CLIENT_IP`: The whitelisted IP address for API access
- `NAMECHEAP_SANDBOX`: Set to "true" for testing against Namecheap's sandbox environment

Optional configuration:

- `DRY_RUN`: Set to "true" to prevent actual changes to DNS records (default: false)
- `DEBUG`: Set to "true" for verbose logging (default: false)
- `DEFAULT_TTL`: Default TTL for DNS records in seconds (default: 7200)
- `DOMAIN_FILTER`: Comma-separated list of domains to include
- `EXCLUDE_DOMAIN_FILTER`: Comma-separated list of domains to exclude
- `REGEXP_DOMAIN_FILTER`: Regular expression for domain filtering
- `REGEXP_DOMAIN_FILTER_EXCLUSION`: Regular expression for domain exclusion

Server configuration:

- `WEBHOOK_HOST`: Hostname for the webhook server (default: localhost)
- `WEBHOOK_PORT`: Port for the webhook server (default: 8888)
- `METRICS_HOST`: Hostname for the metrics/health server (default: 0.0.0.0)
- `METRICS_PORT`: Port for the metrics/health server (default: 8080)
- `READ_TIMEOUT`: Read timeout in milliseconds (default: 60000)
- `WRITE_TIMEOUT`: Write timeout in milliseconds (default: 60000)

## Quick Start

The easiest way to get started is using the included helper script:

```bash
# Clone the repository
git clone https://github.com/yourusername/external-dns-namecheap-webhook.git
cd external-dns-namecheap-webhook

# Create a .env file with your API credentials
cat > .env << EOF
NAMECHEAP_API_KEY=your_api_key
NAMECHEAP_USERNAME=your_username
NAMECHEAP_API_USER=your_api_user
NAMECHEAP_CLIENT_IP=your_whitelisted_ip
NAMECHEAP_SANDBOX=true
DEBUG=true
EOF

# Run the webhook
./run_webhook.sh
```

This script will automatically:
- Build the webhook if needed
- Find available ports if defaults are in use
- Load environment variables from your .env file
- Start the webhook server with proper configuration

## Deployment

### Using Docker

```bash
docker build -t external-dns-namecheap-webhook .

docker run -p 8888:8888 -p 8080:8080 \
  -e NAMECHEAP_API_KEY=your_api_key \
  -e NAMECHEAP_USERNAME=your_username \
  -e NAMECHEAP_API_USER=your_api_user \
  -e NAMECHEAP_CLIENT_IP=your_whitelisted_ip \
  -e NAMECHEAP_SANDBOX=false \
  external-dns-namecheap-webhook
```

### Kubernetes Deployment

1. Create a secret with your Namecheap API credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: namecheap-api-credentials
  namespace: kube-system
type: Opaque
stringData:
  NAMECHEAP_API_KEY: your_api_key
  NAMECHEAP_USERNAME: your_username
  NAMECHEAP_API_USER: your_api_user
  NAMECHEAP_CLIENT_IP: your_whitelisted_ip
  NAMECHEAP_SANDBOX: "false"
```

2. Deploy the webhook:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns-namecheap-webhook
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: external-dns-namecheap-webhook
  template:
    metadata:
      labels:
        app: external-dns-namecheap-webhook
    spec:
      containers:
      - name: external-dns-namecheap-webhook
        image: your-registry/external-dns-namecheap-webhook:latest
        ports:
        - containerPort: 8888
          name: http
        - containerPort: 8080
          name: health
        envFrom:
        - secretRef:
            name: namecheap-api-credentials
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: external-dns-namecheap-webhook
  namespace: kube-system
spec:
  selector:
    app: external-dns-namecheap-webhook
  ports:
  - port: 8888
    targetPort: 8888
    name: http
  - port: 8080
    targetPort: 8080
    name: health
```

3. Configure ExternalDNS to use the webhook:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: external-dns
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: external-dns
  template:
    metadata:
      labels:
        app: external-dns
    spec:
      containers:
      - name: external-dns
        image: k8s.gcr.io/external-dns/external-dns:v0.13.1
        args:
        - --source=service
        - --source=ingress
        - --provider=webhook
        - --webhook-provider-url=http://external-dns-namecheap-webhook:8888
        - --registry=txt
        - --txt-owner-id=k8s
```

## Example Usage

After deploying ExternalDNS with the Namecheap webhook, it will automatically create DNS records for your services and ingresses.

For example, with a Service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx
  annotations:
    external-dns.alpha.kubernetes.io/hostname: nginx.example.com
spec:
  type: LoadBalancer
  ports:
  - port: 80
    name: http
  selector:
    app: nginx
```

Or with an Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx
spec:
  rules:
  - host: nginx.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
```

## Limitations

- Namecheap API rate limits apply
- Only supports domains that use Namecheap's DNS servers
- Premium DNS services may require additional configuration

## Development

To build and run locally:

```bash
# The easiest way - uses the provided helper script
./run_webhook.sh
```

Alternatively, you can manually set up the environment:

```bash
go build -o namecheap-webhook .
export NAMECHEAP_API_KEY=your_api_key
export NAMECHEAP_USERNAME=your_username
export NAMECHEAP_API_USER=your_api_user
export NAMECHEAP_CLIENT_IP=your_whitelisted_ip
export NAMECHEAP_SANDBOX=true
./namecheap-webhook
```

### Testing

You can test the webhook functionality using the provided test script:

```bash
# Method 1: Using run_webhook.sh (recommended)
./run_webhook.sh &
DOMAIN=yourdomain.com ./test/test_webhook.sh

# Method 2: Manual setup
# Set up environment variables first
export NAMECHEAP_API_KEY=your_api_key
export NAMECHEAP_USERNAME=your_username
export NAMECHEAP_API_USER=your_api_user
export NAMECHEAP_CLIENT_IP=your_whitelisted_ip
export NAMECHEAP_SANDBOX=true

# Build and run the webhook
go build -o namecheap-webhook .
./namecheap-webhook &

# Run the test script (in a new terminal)
DOMAIN=yourdomain.com ./test/test_webhook.sh
```

The test script will:
1. Check if the webhook is healthy
2. Get current endpoints
3. Create a test record
4. Verify the record was created
5. Modify the record
6. Delete the record
7. Verify the record was deleted

### Troubleshooting

If you encounter issues with the webhook, please refer to our [Troubleshooting Guide](docs/troubleshooting.md) for common problems and solutions.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

