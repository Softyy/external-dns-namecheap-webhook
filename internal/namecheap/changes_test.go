package namecheap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	"sigs.k8s.io/external-dns/endpoint"
)

type mockAPIClient struct {
	zones   []namecheap.DomainsGetInfoResult
	records map[string][]namecheap.DomainsDNSHostRecordDetailed
	setErr  error
}

func (m *mockAPIClient) GetZones(ctx context.Context) ([]namecheap.DomainsGetInfoResult, error) {
	return m.zones, nil
}

func (m *mockAPIClient) GetRecords(domain string) ([]namecheap.DomainsDNSHostRecordDetailed, string, error) {
	if m.records == nil {
		return []namecheap.DomainsDNSHostRecordDetailed{}, "", nil
	}
	return m.records[domain], "", nil
}

func (m *mockAPIClient) SetRecords(domain string, records []namecheap.DomainsDNSHostRecord, emailType string) error {
	return m.setErr
}

func (m *mockAPIClient) DeleteRecord(ctx context.Context, domain string, recordType string, name string) error {
	return m.setErr
}

func TestProcessCreateActions(t *testing.T) {
	domainMap := map[string]bool{"example.com": true}
	creates := []*endpoint.Endpoint{
		endpoint.NewEndpointWithTTL("www.example.com", endpoint.RecordTypeA, endpoint.TTL(300), "1.2.3.4"),
	}

	client := &mockAPIClient{}
	runner := NewNamecheapChanges(client, false, 7200)

	processCreateActions(domainMap, creates, runner, 7200)

	assert.Len(t, runner.creates, 1)
	assert.Contains(t, runner.creates, "example.com")
}

func TestProcessDeleteActions(t *testing.T) {
	domainMap := map[string]bool{"example.com": true}
	deletes := []*endpoint.Endpoint{
		endpoint.NewEndpoint("old.example.com", endpoint.RecordTypeA, "1.2.3.4"),
	}

	client := &mockAPIClient{}
	runner := NewNamecheapChanges(client, false, 7200)

	processDeleteActions(domainMap, deletes, runner)

	assert.Len(t, runner.deletes, 1)
	assert.Contains(t, runner.deletes, "example.com")
	assert.Equal(t, "old", runner.deletes["example.com"][0].hostName)
}

func TestProcessUpdateActions(t *testing.T) {
	domainMap := map[string]bool{"example.com": true}
	updates := []*endpoint.Endpoint{
		endpoint.NewEndpointWithTTL("www.example.com", endpoint.RecordTypeA, endpoint.TTL(600), "5.6.7.8"),
	}

	client := &mockAPIClient{}
	runner := NewNamecheapChanges(client, false, 7200)

	processUpdateActions(domainMap, updates, runner, 7200)

	assert.Len(t, runner.updates, 1)
	assert.Contains(t, runner.updates, "example.com")
}

func TestApplyChanges_DeleteOnly(t *testing.T) {
	name := "old"
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

	runner := NewNamecheapChanges(client, false, 7200)
	runner.AddChangeDelete("example.com", &namecheapChangeDelete{hostName: "old", recordType: "A"})

	err := runner.ApplyChanges(context.Background())
	assert.NoError(t, err)
}

func TestApplyChanges_DryRun(t *testing.T) {
	client := &mockAPIClient{}
	runner := NewNamecheapChanges(client, true, 7200)

	domainMap := map[string]bool{"example.com": true}
	deletes := []*endpoint.Endpoint{
		endpoint.NewEndpoint("old.example.com", endpoint.RecordTypeA, "1.2.3.4"),
	}

	processDeleteActions(domainMap, deletes, runner)

	err := runner.ApplyChanges(context.Background())
	assert.NoError(t, err)
}

func TestApplyChanges_Empty(t *testing.T) {
	client := &mockAPIClient{}
	runner := NewNamecheapChanges(client, false, 7200)

	err := runner.ApplyChanges(context.Background())
	assert.NoError(t, err)
}

func TestNewChangeCreate(t *testing.T) {
	ep := endpoint.NewEndpointWithTTL("www.example.com", endpoint.RecordTypeA, endpoint.TTL(300), "1.2.3.4")
	change := newChangeCreate("example.com", ep, 7200)

	assert.Equal(t, "www", change.hostName)
	assert.Len(t, change.records, 1)
}

func TestNewChangeCreate_MX(t *testing.T) {
	ep := endpoint.NewEndpointWithTTL("mail.example.com", endpoint.RecordTypeMX, endpoint.TTL(300), "10 mail.example.com")
	change := newChangeCreate("example.com", ep, 7200)

	assert.Equal(t, "mail", change.hostName)
	assert.Len(t, change.records, 1)
}

func TestNewChangeDelete(t *testing.T) {
	ep := endpoint.NewEndpoint("old.example.com", endpoint.RecordTypeA, "1.2.3.4")
	change := newChangeDelete("example.com", ep)

	assert.Equal(t, "old", change.hostName)
	assert.Equal(t, "A", change.recordType)
}