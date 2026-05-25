package namecheap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

func TestNamecheapProvider_Zones(t *testing.T) {
	domainName := "example.com"
	client := &mockAPIClient{
		zones: []namecheap.DomainsGetInfoResult{
			{DomainName: &domainName},
		},
	}

	provider := &NamecheapProvider{
		client:       client,
		domainFilter:  endpoint.NewDomainFilter([]string{}),
		domainMap:     make(map[string]bool),
		defaultTTL:    7200,
	}

	zones, err := provider.Zones(context.Background())
	assert.NoError(t, err)
	assert.Len(t, zones, 1)
	assert.Equal(t, "example.com", *zones[0].DomainName)
}

func TestNamecheapProvider_Records(t *testing.T) {
	domainName := "example.com"
	name := "www"
	recordType := "A"
	address := "1.2.3.4"
	ttl := 300

	client := &mockAPIClient{
		zones: []namecheap.DomainsGetInfoResult{
			{DomainName: &domainName},
		},
		records: map[string][]namecheap.DomainsDNSHostRecordDetailed{
			"example.com": {
				{Name: &name, Type: &recordType, Address: &address, TTL: &ttl},
			},
		},
	}

	provider := &NamecheapProvider{
		client:       client,
		domainFilter:  endpoint.NewDomainFilter([]string{}),
		domainMap:     make(map[string]bool),
		defaultTTL:    7200,
	}

	endpoints, err := provider.Records(context.Background())
	assert.NoError(t, err)
	assert.Len(t, endpoints, 1)
	assert.Equal(t, "www.example.com", endpoints[0].DNSName)
	assert.Equal(t, "A", endpoints[0].RecordType)
}

func TestNamecheapProvider_AdjustEndpoints(t *testing.T) {
	provider := &NamecheapProvider{}
	ep := []*endpoint.Endpoint{
		endpoint.NewEndpoint("www.example.com", "A", "1.2.3.4"),
	}

	result, err := provider.AdjustEndpoints(ep)
	assert.NoError(t, err)
	assert.Equal(t, ep, result)
}

func TestNamecheapProvider_GetDomainFilter(t *testing.T) {
	filter := endpoint.NewDomainFilter([]string{"example.com"})
	provider := &NamecheapProvider{
		domainFilter: filter,
	}

	result := provider.GetDomainFilter()
	assert.NotNil(t, result)
}

func TestNamecheapProvider_ApplyChanges_NoChanges(t *testing.T) {
	client := &mockAPIClient{}
	provider := &NamecheapProvider{
		client:       client,
		domainMap:     make(map[string]bool),
		defaultTTL:    7200,
	}

	changes := &plan.Changes{}
	err := provider.ApplyChanges(context.Background(), changes)
	assert.NoError(t, err)
}