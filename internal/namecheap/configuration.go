package namecheap

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/codingconcepts/env"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

type Configuration struct {
	UserName             string   `env:"NAMECHEAP_USERNAME" required:"true"`
	ApiUser              string   `env:"NAMECHEAP_API_USER" required:"true"`
	APIKey               string   `env:"NAMECHEAP_API_KEY" required:"true"`
	ClientIp             string   `env:"NAMECHEAP_CLIENT_IP" required:"true"`
	UseSandbox           bool     `env:"NAMECHEAP_SANDBOX" default:"false"`
	DryRun               bool     `env:"DRY_RUN" default:"false"`
	Debug                bool     `env:"DEBUG" default:"false"`
	BatchSize            int      `env:"BATCH_SIZE" default:"100"`
	DefaultTTL           int      `env:"DEFAULT_TTL" default:"7200"`
	DomainFilter         []string `env:"DOMAIN_FILTER" default:""`
	ExcludeDomains       []string `env:"EXCLUDE_DOMAIN_FILTER" default:""`
	RegexDomainFilter    string   `env:"REGEXP_DOMAIN_FILTER" default:""`
	RegexDomainExclusion string   `env:"REGEXP_DOMAIN_FILTER_EXCLUSION" default:""`
}

func NewConfiguration() (*Configuration, error) {
	cfg := &Configuration{}
	if err := env.Set(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func GetDomainFilter(config Configuration) *endpoint.DomainFilter {
	var domainFilter *endpoint.DomainFilter
	createMsg := "Creating Namecheap provider with "

	if config.RegexDomainFilter != "" {
		createMsg += fmt.Sprintf("regexp domain filter: '%s', ", config.RegexDomainFilter)
		if config.RegexDomainExclusion != "" {
			createMsg += fmt.Sprintf("with exclusion: '%s', ", config.RegexDomainExclusion)
		}
		domainFilter = endpoint.NewRegexDomainFilter(
			regexp.MustCompile(config.RegexDomainFilter),
			regexp.MustCompile(config.RegexDomainExclusion),
		)
	} else {
		if len(config.DomainFilter) > 0 {
			createMsg += fmt.Sprintf("zoneNode filter: '%s', ", strings.Join(config.DomainFilter, ","))
		}
		if len(config.ExcludeDomains) > 0 {
			createMsg += fmt.Sprintf("Exclude domain filter: '%s', ", strings.Join(config.ExcludeDomains, ","))
		}
		domainFilter = endpoint.NewDomainFilterWithExclusions(config.DomainFilter, config.ExcludeDomains)
	}

	createMsg = strings.TrimSuffix(createMsg, ", ")
	if strings.HasSuffix(createMsg, "with ") {
		createMsg += "no kind of domain filters"
	}
	log.Info(createMsg)
	return domainFilter
}

func IsSupportedRecordType(recordType string) bool {
	switch recordType {
	case "A", "AAAA", "ALIAS", "CAA", "CNAME", "MX", "NS", "TXT":
		return true
	default:
		return false
	}
}