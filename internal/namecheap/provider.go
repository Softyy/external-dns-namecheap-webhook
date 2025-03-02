package namecheap

import (
	"context"

	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"

	log "github.com/sirupsen/logrus"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
)

// NamecheapProvider implements ExternalDNS' provider.Provider interface for
// Namecheap.
type NamecheapProvider struct {
	provider.BaseProvider
	client           namecheap.Client
	batchSize        int
	debug            bool
	dryRun           bool
	defaultTTL       int
	zoneIDNameMapper provider.ZoneIDName
	domainFilter     endpoint.DomainFilter
}

// NewNamecheapProvider creates a new NamecheapProvider instance.
func NewNamecheapProvider(config *Configuration) (*NamecheapProvider, error) {
	var logLevel log.Level
	if config.Debug {
		logLevel = log.DebugLevel
	} else {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	return &NamecheapProvider{
		client: *namecheap.NewClient(&namecheap.ClientOptions{
			UserName:   "softyy",
			ApiUser:    "ApiUser",
			ApiKey:     "6ee70379b46846bfb61802a4f68be2d1",
			ClientIp:   "10.10.10.10",
			UseSandbox: true}),
		batchSize:    config.BatchSize,
		debug:        config.Debug,
		dryRun:       config.DryRun,
		defaultTTL:   config.DefaultTTL,
		domainFilter: GetDomainFilter(*config),
	}, nil
}

// Zones returns the list of the hosted DNS zones.
// If a domain filter is set, it only returns the zones that match it.
func (p *NamecheapProvider) Zones(ctx context.Context) ([]hdns.Zone, error) {
	metrics := metrics.GetOpenMetricsInstance()
	result := []hdns.Zone{}

	zones, err := fetchZones(ctx, p.client, p.batchSize)
	if err != nil {
		return nil, err
	}

	filteredOutZones := 0
	for _, zone := range zones {
		if p.domainFilter.Match(zone.Name) {
			result = append(result, zone)
		} else {
			filteredOutZones++
		}
	}
	metrics.SetFilteredOutZones(filteredOutZones)

	p.ensureZoneIDMappingPresent(zones)

	return result, nil
}

// AdjustEndpoints adjusts the endpoints according to the provider
// requirements.
func (p NamecheapProvider) AdjustEndpoints(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, error) {
	adjustedEndpoints := []*endpoint.Endpoint{}

	for _, ep := range endpoints {
		_, zoneName := p.zoneIDNameMapper.FindZone(ep.DNSName)
		adjustedTargets := endpoint.Targets{}
		for _, t := range ep.Targets {
			adjustedTarget := makeEndpointTarget(zoneName, t, ep.RecordType)
			adjustedTargets = append(adjustedTargets, adjustedTarget)
		}

		ep.Targets = adjustedTargets
		adjustedEndpoints = append(adjustedEndpoints, ep)
	}

	return adjustedEndpoints, nil
}

// logDebugEndpoints logs every endpoint as a a line.
func logDebugEndpoints(endpoints []*endpoint.Endpoint) {
	for idx, ep := range endpoints {
		log.WithFields(getEndpointLogFields(ep)).Debugf("Endpoint %d", idx)
	}
}

// Records returns the list of records in all zones as a slice of endpoints.
func (p *NamecheapProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	endpoints := []*endpoint.Endpoint{}
	for _, zone := range zones {
		records, err := fetchRecords(ctx, zone.ID, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}

		skippedRecords := 0
		// Add only endpoints from supported types.
		for _, r := range records {
			// Ensure the record has all the required zone information
			r.Zone = &zone
			if provider.SupportedRecordType(string(r.Type)) {
				ep := createEndpointFromRecord(r)
				endpoints = append(endpoints, ep)
			} else {
				skippedRecords++
			}
		}
		m := metrics.GetOpenMetricsInstance()
		m.SetSkippedRecords(zone.Name, skippedRecords)
	}

	// Merge endpoints with the same name and type (e.g., multiple A records for a single
	// DNS name) into one endpoint with multiple targets.
	endpoints = mergeEndpointsByNameType(endpoints)

	// Log the endpoints that were found.
	if p.debug {
		log.Debugf("Returning %d endpoints.", len(endpoints))
		logDebugEndpoints(endpoints)
	}

	return endpoints, nil
}

// ensureZoneIDMappingPresent prepares the zoneIDNameMapper, that associates
// each ZoneID woth the zone name.
func (p *NamecheapProvider) ensureZoneIDMappingPresent(zones []hdns.Zone) {
	zoneIDNameMapper := provider.ZoneIDName{}
	for _, z := range zones {
		zoneIDNameMapper.Add(z.ID, z.Name)
	}
	p.zoneIDNameMapper = zoneIDNameMapper
}

// getRecordsByZoneID returns a map that associates each ZoneID with the
// records contained in that zone.
func (p *NamecheapProvider) getRecordsByZoneID(ctx context.Context) (map[string][]hdns.Record, error) {
	recordsByZoneID := map[string][]hdns.Record{}

	zones, err := p.Zones(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch records for each zone
	for _, zone := range zones {
		records, err := fetchRecords(ctx, zone.ID, p.client, p.batchSize)
		if err != nil {
			return nil, err
		}
		// Add full zone information
		zonedRecords := []hdns.Record{}
		for _, r := range records {
			r.Zone = &zone
			zonedRecords = append(zonedRecords, r)
		}
		recordsByZoneID[zone.ID] = append(recordsByZoneID[zone.ID], zonedRecords...)
	}

	return recordsByZoneID, nil
}

// ApplyChanges applies the given set of generic changes to the provider.
func (p *NamecheapProvider) ApplyChanges(ctx context.Context, planChanges *plan.Changes) error {
	if !planChanges.HasChanges() {
		return nil
	}

	recordsByZoneID, err := p.getRecordsByZoneID(ctx)
	if err != nil {
		return err
	}

	log.Debug("Preparing creates")
	createsByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.Create)
	log.Debug("Preparing updates")
	updatesByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.UpdateNew)
	log.Debug("Preparing deletes")
	deletesByZoneID := endpointsByZoneID(p.zoneIDNameMapper, planChanges.Delete)

	changes := hetznerChanges{
		dryRun:     p.dryRun,
		defaultTTL: p.defaultTTL,
	}

	processCreateActions(p.zoneIDNameMapper, recordsByZoneID, createsByZoneID, &changes)
	processUpdateActions(p.zoneIDNameMapper, recordsByZoneID, updatesByZoneID, &changes)
	processDeleteActions(p.zoneIDNameMapper, recordsByZoneID, deletesByZoneID, &changes)

	return changes.ApplyChanges(ctx, p.client)
}
