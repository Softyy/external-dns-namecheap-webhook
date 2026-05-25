package namecheap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		expected    *Configuration
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"NAMECHEAP_USERNAME":  "testuser",
				"NAMECHEAP_API_USER":  "testapiuser",
				"NAMECHEAP_API_KEY":   "testapikey",
				"NAMECHEAP_CLIENT_IP": "1.2.3.4",
			},
			expectError: false,
			expected: &Configuration{
				UserName:             "testuser",
				ApiUser:              "testapiuser",
				APIKey:               "testapikey",
				ClientIp:             "1.2.3.4",
				UseSandbox:           false,
				DryRun:               false,
				Debug:                false,
				BatchSize:            100,
				DefaultTTL:           7200,
				DomainFilter:         []string{""},
				ExcludeDomains:       []string{""},
				RegexDomainFilter:    "",
				RegexDomainExclusion: "",
			},
		},
		{
			name:        "missing required fields",
			envVars:     map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := NewConfiguration()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.UserName, cfg.UserName)
				assert.Equal(t, tt.expected.ApiUser, cfg.ApiUser)
				assert.Equal(t, tt.expected.APIKey, cfg.APIKey)
				assert.Equal(t, tt.expected.ClientIp, cfg.ClientIp)
				assert.Equal(t, tt.expected.UseSandbox, cfg.UseSandbox)
				assert.Equal(t, tt.expected.DefaultTTL, cfg.DefaultTTL)
				assert.Equal(t, tt.expected.BatchSize, cfg.BatchSize)
			}
		})
	}
}

func TestGetDomainFilter_ListFilter(t *testing.T) {
	config := Configuration{
		DomainFilter:   []string{"example.com", "test.org"},
		ExcludeDomains: []string{"bad.com"},
	}

	filter := GetDomainFilter(config)
	assert.NotNil(t, filter)
	assert.True(t, filter.Match("example.com"))
	assert.True(t, filter.Match("test.org"))
}

func TestGetDomainFilter_RegexFilter(t *testing.T) {
	config := Configuration{
		RegexDomainFilter:    ".*\\.example\\.com",
		RegexDomainExclusion: "bad\\.example\\.com",
	}

	filter := GetDomainFilter(config)
	assert.NotNil(t, filter)
	assert.True(t, filter.Match("sub.example.com"))
}

func TestIsSupportedRecordType(t *testing.T) {
	supported := []string{"A", "AAAA", "ALIAS", "CAA", "CNAME", "MX", "NS", "TXT"}
	for _, rt := range supported {
		assert.True(t, IsSupportedRecordType(rt), "Expected %s to be supported", rt)
	}

	unsupported := []string{"SRV", "PTR", "SOA", "URL", "URL301", "FRAME", "MXE"}
	for _, rt := range unsupported {
		assert.False(t, IsSupportedRecordType(rt), "Expected %s to NOT be supported", rt)
	}
}

