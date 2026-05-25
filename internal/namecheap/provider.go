package namecheap

import (
	"context"
	"fmt"
	"log/slog"

	"external-dns/webhooks/namecheap/internal/metrics"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

type NamecheapProvider struct {
	provider.BaseProvider
	client       apiClient
	debug        bool
	dryRun       bool
	defaultTTL   int
	batchSize    int
	domainFilter *endpoint.DomainFilter
	domainMap    map[string]bool
}

func NewNamecheapProvider(config *Configuration) (*NamecheapProvider, error) {
	client, err := NewNamecheapDNS(config)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate namecheap DNS provider: %w", err)
	}

	return &NamecheapProvider{
		client:       client,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		defaultTTL:   config.DefaultTTL,
		batchSize:    config.BatchSize,
		domainFilter: GetDomainFilter(*config),
		domainMap:    make(map[string]bool),
	}, nil
}

func (p *NamecheapProvider) Zones(ctx context.Context) ([]namecheap.DomainsGetInfoResult, error) {
	m := metrics.GetOpenMetricsInstance()
	result := []namecheap.DomainsGetInfoResult{}

	domains, err := fetchZones(ctx, p.client)
	if err != nil {
		return nil, err
	}

	filteredOutZones := 0
	for _, domain := range domains {
		if domain.DomainName == nil {
			continue
		}

		domainName := *domain.DomainName
		if p.domainFilter.Match(domainName) {
			result = append(result, domain)
			p.domainMap[domainName] = true
		} else {
			filteredOutZones++
		}
	}
	m.SetFilteredOutZones(filteredOutZones)

	slog.Debug("Got zones", "total", len(result), "filtered", filteredOutZones)

	return result, nil
}

func (p *NamecheapProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	return endpoints, nil
}

func (p *NamecheapProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		if zone.DomainName == nil {
			continue
		}
		domainName := *zone.DomainName

		records, err := fetchRecords(ctx, domainName, p.client)
		if err != nil {
			return nil, err
		}

		skippedRecords := 0
		for _, r := range records {
			recordType := *r.HostRecord.Type
			if IsSupportedRecordType(recordType) {
				ep := createEndpointFromRecord(r)
				endpoints = append(endpoints, ep)
			} else {
				skippedRecords++
			}
		}
		m := metrics.GetOpenMetricsInstance()
		m.SetSkippedRecords(domainName, skippedRecords)
	}

	endpoints = mergeEndpointsByNameType(endpoints)

	if p.debug {
		slog.Debug("Returning endpoints", "count", len(endpoints))
		for _, ep := range endpoints {
			slog.Debug("Endpoint", "dnsName", ep.DNSName, "recordType", ep.RecordType, "targets", ep.Targets.String(), "ttl", ep.RecordTTL)
		}
	}

	return endpoints, nil
}

func (p *NamecheapProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if !changes.HasChanges() {
		slog.Debug("No changes to be applied found.")
		return nil
	}

	changesRunner := NewNamecheapChanges(p.client, p.dryRun, p.defaultTTL)

	slog.Debug("Preparing creates")
	processCreateActions(p.domainMap, changes.Create, changesRunner, p.defaultTTL)
	slog.Debug("Preparing updates")
	processUpdateActions(p.domainMap, changes.UpdateNew, changesRunner, p.defaultTTL)
	slog.Debug("Preparing deletes")
	processDeleteActions(p.domainMap, changes.Delete, changesRunner)

	return changesRunner.ApplyChanges(ctx)
}

func (p NamecheapProvider) GetDomainFilter() endpoint.DomainFilterInterface {
	return p.domainFilter
}

func processCreateActions(domainMap map[string]bool, creates []*endpoint.Endpoint, runner changesRunner, defaultTTL int) {
	byDomain := endpointsByDomain(creates, domainMap)
	for domain, eps := range byDomain {
		for _, ep := range eps {
			change := newChangeCreate(domain, ep, defaultTTL)
			runner.AddChangeCreate(domain, change)
		}
	}
}

func processUpdateActions(domainMap map[string]bool, updates []*endpoint.Endpoint, runner changesRunner, defaultTTL int) {
	byDomain := endpointsByDomain(updates, domainMap)
	for domain, eps := range byDomain {
		for _, ep := range eps {
			change := newChangeUpdate(domain, ep, defaultTTL)
			runner.AddChangeUpdate(domain, change)
		}
	}
}

func processDeleteActions(domainMap map[string]bool, deletes []*endpoint.Endpoint, runner changesRunner) {
	byDomain := endpointsByDomain(deletes, domainMap)
	for domain, eps := range byDomain {
		for _, ep := range eps {
			change := newChangeDelete(domain, ep)
			runner.AddChangeDelete(domain, change)
		}
	}
}