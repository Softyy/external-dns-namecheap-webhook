package namecheap

import (
	"context"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

type apiClient interface {
	GetZones(ctx context.Context) ([]namecheap.DomainsGetInfoResult, error)
	GetRecords(domain string) ([]namecheap.DomainsDNSHostRecordDetailed, string, error)
	SetRecords(domain string, records []namecheap.DomainsDNSHostRecord, emailType string) error
	DeleteRecord(ctx context.Context, domain string, recordType string, name string) error
}

func fetchRecords(ctx context.Context, domain string, client apiClient) ([]namecheapRecord, error) {
	detailedRecords, emailType, err := client.GetRecords(domain)
	if err != nil {
		return nil, err
	}

	records := []namecheapRecord{}
	for i := range detailedRecords {
		records = append(records, namecheapRecord{
			HostRecord: &detailedRecords[i],
			Domain:     domain,
			EmailType:  emailType,
		})
	}

	return records, nil
}

func fetchZones(ctx context.Context, client apiClient) ([]namecheap.DomainsGetInfoResult, error) {
	return client.GetZones(ctx)
}

