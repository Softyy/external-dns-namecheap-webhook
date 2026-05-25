package namecheap

import (
	"strconv"
	"strings"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

type namecheapChangeCreate struct {
	hostName string
	records  []namecheap.DomainsDNSHostRecord
}

type namecheapChangeUpdate struct {
	hostName string
	records  []namecheap.DomainsDNSHostRecord
}

type namecheapChangeDelete struct {
	hostName   string
	recordType string
}

func newChangeCreate(domain string, ep *endpoint.Endpoint, defaultTTL int) *namecheapChangeCreate {
	hostName := extractHostFromFQDN(ep.DNSName, domain)
	ttl := getEndpointTTL(ep, defaultTTL)
	records := []namecheap.DomainsDNSHostRecord{}

	for _, target := range ep.Targets {
		record := namecheap.DomainsDNSHostRecord{
			HostName:   namecheap.String(hostName),
			RecordType: namecheap.String(ep.RecordType),
			Address:    namecheap.String(target),
			TTL:        namecheap.Int(ttl),
		}
		if ep.RecordType == "MX" {
			priority := 10
			parts := strings.SplitN(target, " ", 2)
			if len(parts) == 2 {
				if p, err := strconv.Atoi(parts[0]); err == nil {
					priority = p
					record.Address = namecheap.String(adjustMXTarget(domain, parts[1]))
				}
			}
			record.MXPref = namecheap.UInt8(uint8(priority))
		} else if ep.RecordType == "CNAME" {
			record.Address = namecheap.String(adjustCNAMETarget(domain, target))
		}
		records = append(records, record)
	}

	log.Debugf("Change create: [%s] type [%s] in domain [%s] with %d records", hostName, ep.RecordType, domain, len(records))

	return &namecheapChangeCreate{
		hostName: hostName,
		records:  records,
	}
}

func newChangeUpdate(domain string, ep *endpoint.Endpoint, defaultTTL int) *namecheapChangeUpdate {
	hostName := extractHostFromFQDN(ep.DNSName, domain)
	ttl := getEndpointTTL(ep, defaultTTL)
	records := []namecheap.DomainsDNSHostRecord{}

	for _, target := range ep.Targets {
		record := namecheap.DomainsDNSHostRecord{
			HostName:   namecheap.String(hostName),
			RecordType: namecheap.String(ep.RecordType),
			Address:    namecheap.String(target),
			TTL:        namecheap.Int(ttl),
		}
		if ep.RecordType == "MX" {
			priority := 10
			parts := strings.SplitN(target, " ", 2)
			if len(parts) == 2 {
				if p, err := strconv.Atoi(parts[0]); err == nil {
					priority = p
					record.Address = namecheap.String(adjustMXTarget(domain, parts[1]))
				}
			}
			record.MXPref = namecheap.UInt8(uint8(priority))
		} else if ep.RecordType == "CNAME" {
			record.Address = namecheap.String(adjustCNAMETarget(domain, target))
		}
		records = append(records, record)
	}

	log.Debugf("Change update: [%s] type [%s] in domain [%s] with %d records", hostName, ep.RecordType, domain, len(records))

	return &namecheapChangeUpdate{
		hostName: hostName,
		records:  records,
	}
}

func newChangeDelete(domain string, ep *endpoint.Endpoint) *namecheapChangeDelete {
	hostName := extractHostFromFQDN(ep.DNSName, domain)
	log.Debugf("Change delete: [%s] type [%s] from domain [%s]", hostName, ep.RecordType, domain)
	return &namecheapChangeDelete{
		hostName:   hostName,
		recordType: ep.RecordType,
	}
}

