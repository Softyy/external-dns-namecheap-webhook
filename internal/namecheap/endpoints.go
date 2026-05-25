package namecheap

import (
	"fmt"
	"strings"

	namecheap "github.com/namecheap/go-namecheap-sdk/v2/namecheap"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

type namecheapRecord struct {
	HostRecord *namecheap.DomainsDNSHostRecordDetailed
	Domain     string
	EmailType  string
}

func createEndpointFromRecord(record namecheapRecord) *endpoint.Endpoint {
	recordType := *record.HostRecord.Type
	name := buildFQDN(*record.HostRecord.Name, record.Domain)
	ttl := endpoint.TTL(*record.HostRecord.TTL)
	target := *record.HostRecord.Address

	if recordType == "MX" && record.HostRecord.MXPref != nil {
		target = fmt.Sprintf("%d %s", *record.HostRecord.MXPref, *record.HostRecord.Address)
	}

	return endpoint.NewEndpointWithTTL(name, recordType, ttl, target)
}

func buildFQDN(host, domain string) string {
	if host == "@" {
		return domain
	}
	return fmt.Sprintf("%s.%s", host, domain)
}

func extractHostFromFQDN(fqdn, domain string) string {
	if fqdn == domain {
		return "@"
	}
	return strings.TrimSuffix(fqdn, "."+domain)
}

func mergeEndpointsByNameType(endpoints []*endpoint.Endpoint) []*endpoint.Endpoint {
	merged := make(map[string]*endpoint.Endpoint)

	for _, ep := range endpoints {
		key := fmt.Sprintf("%s/%s", ep.DNSName, ep.RecordType)
		if existing, found := merged[key]; found {
			existing.Targets = append(existing.Targets, ep.Targets...)
		} else {
			merged[key] = ep
		}
	}

	result := make([]*endpoint.Endpoint, 0, len(merged))
	for _, ep := range merged {
		result = append(result, ep)
	}

	return result
}

func getEndpointLogFields(ep *endpoint.Endpoint) log.Fields {
	return log.Fields{
		"dnsName":    ep.DNSName,
		"recordTTL":  ep.RecordTTL,
		"recordType": ep.RecordType,
		"targets":    strings.Join(ep.Targets, ","),
	}
}

func endpointsByDomain(endpoints []*endpoint.Endpoint, domainMap map[string]bool) map[string][]*endpoint.Endpoint {
	endpointsByDomain := make(map[string][]*endpoint.Endpoint)

	for _, ep := range endpoints {
		domain := findDomainForEndpoint(ep.DNSName, domainMap)
		if domain != "" {
			endpointsByDomain[domain] = append(endpointsByDomain[domain], ep)
		}
	}

	return endpointsByDomain
}

func findDomainForEndpoint(dnsName string, domainMap map[string]bool) string {
	parts := strings.Split(dnsName, ".")
	if len(parts) < 2 {
		return ""
	}

	for i := 2; i <= len(parts); i++ {
		domain := strings.Join(parts[len(parts)-i:], ".")
		if _, exists := domainMap[domain]; exists {
			return domain
		}
	}

	return strings.Join(parts[len(parts)-2:], ".")
}

func getEndpointTTL(ep *endpoint.Endpoint, defaultTTL int) int {
	if ep.RecordTTL.IsConfigured() {
		return int(ep.RecordTTL)
	}
	return defaultTTL
}

func adjustMXTarget(domain string, target string) string {
	target = strings.TrimSuffix(target, ".")
	if target == domain {
		return "@"
	}
	if strings.HasSuffix(target, "."+domain) {
		return strings.TrimSuffix(target, "."+domain)
	}
	return target
}

func adjustCNAMETarget(domain string, target string) string {
	target = strings.TrimSuffix(target, ".")
	if strings.HasSuffix(target, "."+domain) {
		return strings.TrimSuffix(target, "."+domain)
	}
	if target == domain {
		return "@"
	}
	return target
}