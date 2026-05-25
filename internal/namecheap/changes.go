package namecheap

import (
	"context"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	log "github.com/sirupsen/logrus"
)

type changesRunner interface {
	AddChangeCreate(domain string, ep *namecheapChangeCreate)
	AddChangeUpdate(domain string, ep *namecheapChangeUpdate)
	AddChangeDelete(domain string, ep *namecheapChangeDelete)
	ApplyChanges(ctx context.Context) error
}

type namecheapChanges struct {
	dnsClient  apiClient
	dryRun     bool
	defaultTTL int

	creates map[string][]*namecheapChangeCreate
	updates map[string][]*namecheapChangeUpdate
	deletes map[string][]*namecheapChangeDelete
}

func NewNamecheapChanges(dnsClient apiClient, dryRun bool, defaultTTL int) *namecheapChanges {
	return &namecheapChanges{
		dnsClient:  dnsClient,
		dryRun:     dryRun,
		defaultTTL: defaultTTL,
		creates:    make(map[string][]*namecheapChangeCreate),
		updates:    make(map[string][]*namecheapChangeUpdate),
		deletes:    make(map[string][]*namecheapChangeDelete),
	}
}

func (c *namecheapChanges) AddChangeCreate(domain string, change *namecheapChangeCreate) {
	c.creates[domain] = append(c.creates[domain], change)
}

func (c *namecheapChanges) AddChangeUpdate(domain string, change *namecheapChangeUpdate) {
	c.updates[domain] = append(c.updates[domain], change)
}

func (c *namecheapChanges) AddChangeDelete(domain string, change *namecheapChangeDelete) {
	c.deletes[domain] = append(c.deletes[domain], change)
}

func (c namecheapChanges) ApplyChanges(ctx context.Context) error {
	allDomains := make(map[string]bool)
	for domain := range c.creates {
		allDomains[domain] = true
	}
	for domain := range c.updates {
		allDomains[domain] = true
	}
	for domain := range c.deletes {
		allDomains[domain] = true
	}

	if len(allDomains) == 0 {
		log.Debug("No changes to be applied found.")
		return nil
	}

	for domain := range allDomains {
		if err := c.applyDomainChanges(ctx, domain); err != nil {
			return err
		}
	}

	return nil
}

func (c namecheapChanges) applyDomainChanges(ctx context.Context, domain string) error {
	deletes := c.deletes[domain]
	creates := c.creates[domain]
	updates := c.updates[domain]

	if c.dryRun {
		log.Infof("DryRun mode enabled - skipping actual changes to domain %s", domain)
		return nil
	}

	if len(deletes) > 0 && len(creates) == 0 && len(updates) == 0 {
		for _, d := range deletes {
			log.Infof("Deleting record [%s] of type [%s] from domain [%s]", d.hostName, d.recordType, domain)
			if err := c.dnsClient.DeleteRecord(ctx, domain, d.recordType, d.hostName); err != nil {
				return err
			}
		}
		return nil
	}

	currentRecords, emailType, err := c.dnsClient.GetRecords(domain)
	if err != nil {
		return err
	}

	hostRecords := []namecheap.DomainsDNSHostRecord{}
	deleteMap := make(map[string]bool)
	for _, d := range deletes {
		key := d.recordType + "/" + d.hostName
		deleteMap[key] = true
	}

	for _, record := range currentRecords {
		if record.Type == nil || record.Name == nil {
			continue
		}
		key := *record.Type + "/" + *record.Name
		if deleteMap[key] {
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
		hostRecords = append(hostRecords, hostRecord)
	}

	for _, cr := range creates {
		for _, record := range cr.records {
			hostRecords = append(hostRecords, record)
		}
	}

	for _, ur := range updates {
		for _, record := range ur.records {
			hostRecords = append(hostRecords, record)
		}
	}

	log.Infof("Setting %d records for domain %s", len(hostRecords), domain)
	return c.dnsClient.SetRecords(domain, hostRecords, emailType)
}