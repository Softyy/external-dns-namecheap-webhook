package namecheap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

func TestFetchRecords(t *testing.T) {
	name := "www"
	recordType := "A"
	address := "1.2.3.4"
	ttl := 300

	client := &mockAPIClient{
		records: map[string][]namecheap.DomainsDNSHostRecordDetailed{
			"example.com": {
				{Name: &name, Type: &recordType, Address: &address, TTL: &ttl},
			},
		},
	}

	records, err := fetchRecords(context.Background(), "example.com", client)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "www", *records[0].HostRecord.Name)
	assert.Equal(t, "example.com", records[0].Domain)
}

func TestFetchRecords_Empty(t *testing.T) {
	client := &mockAPIClient{}

	records, err := fetchRecords(context.Background(), "example.com", client)
	assert.NoError(t, err)
	assert.Len(t, records, 0)
}

func TestFetchZones(t *testing.T) {
	domainName := "example.com"
	client := &mockAPIClient{
		zones: []namecheap.DomainsGetInfoResult{
			{DomainName: &domainName},
		},
	}

	zones, err := fetchZones(context.Background(), client)
	assert.NoError(t, err)
	assert.Len(t, zones, 1)
}

func TestFetchZones_Empty(t *testing.T) {
	client := &mockAPIClient{}

	zones, err := fetchZones(context.Background(), client)
	assert.NoError(t, err)
	assert.Len(t, zones, 0)
}