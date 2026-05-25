package namecheap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"external-dns/webhooks/namecheap/internal/metrics"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

const (
	actGetZones   = "get_zones"
	actGetRecords = "get_records"
	actSetRecords = "set_records"
	actDelRecords = "delete_records"
)

type namecheapDNS struct {
	client *namecheap.Client
}

func NewNamecheapDNS(config *Configuration) (*namecheapDNS, error) {
	if config.APIKey == "" {
		return nil, errors.New("empty API key provided")
	}
	return &namecheapDNS{
		client: namecheap.NewClient(&namecheap.ClientOptions{
			UserName:   config.UserName,
			ApiUser:    config.ApiUser,
			ApiKey:     config.APIKey,
			ClientIp:   config.ClientIp,
			UseSandbox: config.UseSandbox,
		}),
	}, nil
}

func (n *namecheapDNS) GetZones(ctx context.Context) ([]namecheap.DomainsGetInfoResult, error) {
	m := metrics.GetOpenMetricsInstance()

	args := &namecheap.DomainsGetListArgs{
		PageSize: namecheap.Int(100),
		Page:     namecheap.Int(1),
	}

	var domains []namecheap.Domain
	var results []namecheap.DomainsGetInfoResult

	for {
		start := time.Now()
		response, err := n.client.Domains.GetList(args)
		if err != nil {
			m.IncFailedApiCallsTotal(actGetZones)
			return nil, fmt.Errorf("failed to get domain list: %v", err)
		}
		delay := time.Since(start)
		m.IncSuccessfulApiCallsTotal(actGetZones)
		m.AddApiDelayHist(actGetZones, delay.Milliseconds())

		if response.Domains != nil {
			domains = append(domains, *response.Domains...)
		}

		if response.Paging == nil ||
			response.Paging.TotalItems == nil ||
			response.Paging.CurrentPage == nil ||
			response.Paging.PageSize == nil ||
			*response.Paging.CurrentPage >= *response.Paging.TotalItems / *response.Paging.PageSize + 1 {
			break
		}
		*args.Page++
	}

	for _, domain := range domains {
		if domain.Name == nil {
			continue
		}
		domainName := *domain.Name
		start := time.Now()
		infoResponse, err := n.client.Domains.GetInfo(domainName)
		if err != nil {
			slog.Warn("Failed to get info for domain", "domain", domainName, "error", err)
			m.IncFailedApiCallsTotal(actGetZones)
			continue
		}
		delay := time.Since(start)
		m.IncSuccessfulApiCallsTotal(actGetZones)
		m.AddApiDelayHist(actGetZones, delay.Milliseconds())

		if infoResponse.DomainDNSGetListResult != nil {
			results = append(results, *infoResponse.DomainDNSGetListResult)
		}
	}

	return results, nil
}

func (n *namecheapDNS) GetRecords(domain string) ([]namecheap.DomainsDNSHostRecordDetailed, string, error) {
	m := metrics.GetOpenMetricsInstance()
	start := time.Now()

	response, err := n.client.DomainsDNS.GetHosts(domain)
	if err != nil {
		m.IncFailedApiCallsTotal(actGetRecords)
		return nil, "", fmt.Errorf("failed to get DNS records for domain %s: %v", domain, err)
	}
	delay := time.Since(start)
	m.IncSuccessfulApiCallsTotal(actGetRecords)
	m.AddApiDelayHist(actGetRecords, delay.Milliseconds())

	var emailType string
	if response.DomainDNSGetHostsResult != nil &&
		response.DomainDNSGetHostsResult.IsUsingOurDNS != nil &&
		!*response.DomainDNSGetHostsResult.IsUsingOurDNS {
		slog.Warn("Domain is not using Namecheap DNS, records cannot be modified", "domain", domain)
	}

	if response.DomainDNSGetHostsResult != nil && response.DomainDNSGetHostsResult.EmailType != nil {
		emailType = *response.DomainDNSGetHostsResult.EmailType
	}

	if response.DomainDNSGetHostsResult == nil || response.DomainDNSGetHostsResult.Hosts == nil {
		return []namecheap.DomainsDNSHostRecordDetailed{}, emailType, nil
	}

	return *response.DomainDNSGetHostsResult.Hosts, emailType, nil
}

func (n *namecheapDNS) SetRecords(domain string, records []namecheap.DomainsDNSHostRecord, emailType string) error {
	m := metrics.GetOpenMetricsInstance()
	start := time.Now()

	args := &namecheap.DomainsDNSSetHostsArgs{
		Domain:  namecheap.String(domain),
		Records: &records,
	}

	if emailType != "" {
		args.EmailType = namecheap.String(emailType)
	}

	response, err := n.client.DomainsDNS.SetHosts(args)
	if err != nil {
		m.IncFailedApiCallsTotal(actSetRecords)
		return fmt.Errorf("failed to set DNS records for domain %s: %v", domain, err)
	}

	delay := time.Since(start)
	m.IncSuccessfulApiCallsTotal(actSetRecords)
	m.AddApiDelayHist(actSetRecords, delay.Milliseconds())

	if response.DomainDNSSetHostsResult == nil || !*response.DomainDNSSetHostsResult.IsSuccess {
		return fmt.Errorf("failed to set DNS records for domain %s", domain)
	}

	return nil
}

func (n *namecheapDNS) DeleteRecord(ctx context.Context, domain string, recordType string, name string) error {
	m := metrics.GetOpenMetricsInstance()
	start := time.Now()

	currentRecords, emailType, err := n.GetRecords(domain)
	if err != nil {
		return fmt.Errorf("failed to get current records for domain %s when trying to delete: %v", domain, err)
	}

	updatedRecords := []namecheap.DomainsDNSHostRecord{}
	recordFound := false

	for _, record := range currentRecords {
		if record.Type != nil && record.Name != nil && *record.Type == recordType && *record.Name == name {
			recordFound = true
			slog.Info("Deleting record", "type", recordType, "name", name, "domain", domain)
			continue
		}

		hostRecord := namecheap.DomainsDNSHostRecord{
			HostName:   record.Name,
			RecordType: record.Type,
			Address:    record.Address,
			TTL:        record.TTL,
		}

		if record.Type != nil && *record.Type == "MX" && record.MXPref != nil {
			hostRecord.MXPref = namecheap.UInt8(uint8(*record.MXPref))
		}

		updatedRecords = append(updatedRecords, hostRecord)
	}

	if !recordFound {
		slog.Warn("Record not found, nothing to delete", "type", recordType, "name", name, "domain", domain)
		return nil
	}

	err = n.SetRecords(domain, updatedRecords, emailType)
	if err != nil {
		m.IncFailedApiCallsTotal(actDelRecords)
		return fmt.Errorf("failed to update records after deletion for domain %s: %v", domain, err)
	}

	delay := time.Since(start)
	m.IncSuccessfulApiCallsTotal(actDelRecords)
	m.AddApiDelayHist(actDelRecords, delay.Milliseconds())

	return nil
}