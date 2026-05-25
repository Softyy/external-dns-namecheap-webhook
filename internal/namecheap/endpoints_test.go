package namecheap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestBuildFQDN(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		domain   string
		expected string
	}{
		{"apex record", "@", "example.com", "example.com"},
		{"subdomain", "www", "example.com", "www.example.com"},
		{"deep subdomain", "a.b", "example.com", "a.b.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFQDN(tt.host, tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractHostFromFQDN(t *testing.T) {
	tests := []struct {
		name     string
		fqdn     string
		domain   string
		expected string
	}{
		{"apex record", "example.com", "example.com", "@"},
		{"subdomain", "www.example.com", "example.com", "www"},
		{"deep subdomain", "a.b.example.com", "example.com", "a.b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHostFromFQDN(tt.fqdn, tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeEndpointsByNameType(t *testing.T) {
	ep1 := endpoint.NewEndpoint("www.example.com", "A", "1.2.3.4")
	ep2 := endpoint.NewEndpoint("www.example.com", "A", "5.6.7.8")
	ep3 := endpoint.NewEndpoint("mail.example.com", "MX", "10 mail.example.com")

	result := mergeEndpointsByNameType([]*endpoint.Endpoint{ep1, ep2, ep3})

	assert.Len(t, result, 2)

	var wwwEp, mxEp *endpoint.Endpoint
	for _, ep := range result {
		if ep.DNSName == "www.example.com" {
			wwwEp = ep
		}
		if ep.DNSName == "mail.example.com" {
			mxEp = ep
		}
	}

	assert.NotNil(t, wwwEp)
	assert.NotNil(t, mxEp)
	assert.Len(t, wwwEp.Targets, 2)
	assert.Len(t, mxEp.Targets, 1)
}

func TestGetEndpointTTL(t *testing.T) {
	epWithTTL := endpoint.NewEndpointWithTTL("www.example.com", "A", endpoint.TTL(300), "1.2.3.4")
	epWithoutTTL := endpoint.NewEndpoint("www.example.com", "A", "1.2.3.4")

	assert.Equal(t, 300, getEndpointTTL(epWithTTL, 7200))
	assert.Equal(t, 7200, getEndpointTTL(epWithoutTTL, 7200))
}

func TestFindDomainForEndpoint(t *testing.T) {
	domainMap := map[string]bool{
		"example.com":     true,
		"sub.example.org": true,
	}

	tests := []struct {
		name     string
		dnsName  string
		expected string
	}{
		{"exact match", "example.com", "example.com"},
		{"subdomain match", "www.example.com", "example.com"},
		{"deep subdomain match", "a.b.sub.example.org", "sub.example.org"},
		{"no match fallback", "unknown.other.com", "other.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findDomainForEndpoint(tt.dnsName, domainMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndpointsByDomain(t *testing.T) {
	domainMap := map[string]bool{
		"example.com": true,
		"test.org":    true,
	}

	ep1 := endpoint.NewEndpoint("www.example.com", "A", "1.2.3.4")
	ep2 := endpoint.NewEndpoint("mail.test.org", "MX", "10 mail.test.org")

	result := endpointsByDomain([]*endpoint.Endpoint{ep1, ep2}, domainMap)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "example.com")
	assert.Contains(t, result, "test.org")
}

func TestCreateEndpointFromRecord(t *testing.T) {
	name := "www"
	recordType := "A"
	address := "1.2.3.4"
	ttl := 300

	record := namecheapRecord{
		HostRecord: &namecheap.DomainsDNSHostRecordDetailed{
			Name:    &name,
			Type:    &recordType,
			Address: &address,
			TTL:     &ttl,
		},
		Domain: "example.com",
	}

	ep := createEndpointFromRecord(record)

	assert.Equal(t, "www.example.com", ep.DNSName)
	assert.Equal(t, "A", ep.RecordType)
	assert.Equal(t, endpoint.TTL(300), ep.RecordTTL)
	assert.Contains(t, ep.Targets, "1.2.3.4")
}