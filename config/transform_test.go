package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

// Mock YAML inputs for various test cases
var validYAML = []byte(`
options:
  confidence: 50 # default confidence level for all transformations unless otherwise specified

transformations:
  FQDN->IPAddress:
    priority: 1
    confidence: 80
  FQDN->DomainRecord:
    priority: 2
  FQDN->ALL: 
    exclude: [RIRORG,FQDN]
  IPAddress->IPAddress:
    priority: 1
    confidence: 80
  IPAddress->Netblock:
    priority: 2
  IPAddress->RIRORG:
    # leaving both priority and confidence out

`)

var conflictingNoneYAML = []byte(`
options:
  confidence: 50

transformations:
  FQDN->IPAddress:
    priority: 1
    confidence: 80
  FQDN->none:
    priority: 2
  FQDN->ALL: 
    exclude: [TLS,FQDN]
  IPAddress->IPAddress:
    priority: 1
    confidence: 80
  IPAddress->Netblock:
    priority: 2
  IPAddress->TLS:
    # leaving both priority and confidence out
`)

var conflictingNoneYAML2 = []byte(`
options:
  confidence: 50

transformations:
  FQDN->none:
    priority: 2
  FQDN->IPAddress:
    priority: 1
    confidence: 80
  FQDN->ALL: 
    exclude: [TLS,FQDN]
  IPAddress->IPAddress:
    priority: 1
    confidence: 80
  IPAddress->Netblock:
    priority: 2
  IPAddress->TLS:
    # leaving both priority and confidence out
`)

var invalidKeyYAML = []byte(`
options:
  confidence: 50

transformations:
  FQDN-IPAddress:
    priority: 1
`)

var nonOAMtoYAML = []byte(`
options:
  confidence: 50 # default confidence level for all transformations unless otherwise specified

transformations:
  FQDN->IPAddress:
    priority: 1
    confidence: 80
  FQDN->Amass:
    priority: 2
  FQDN->ALL: 
    exclude: [RIRORG,FQDN]
`)

var nonOAMfromYAML = []byte(`
options:
  confidence: 50 # default confidence level for all transformations unless otherwise specified

transformations:
  FQDN->IPAddress:
    priority: 1
    confidence: 80
  Amass->DomainRecord:
    priority: 2
  FQDN->ALL: 
    exclude: [RIRORG,FQDN]
`)

// Utility function to unmarshal YAML and load transformation settings
func prepareConfig(yamlInput []byte) (*Config, error) {
	conf := NewConfig()
	err := yaml.Unmarshal(yamlInput, conf)
	if err != nil {
		return nil, err
	}
	err = conf.loadTransformSettings(conf)
	return conf, err
}

func TestLoadTransformSettings(t *testing.T) {
	// Test with valid YAML input
	t.Run("valid YAML and settings", func(t *testing.T) {
		conf, err := prepareConfig(validYAML)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if conf.Transformations["FQDN->DomainRecord"].Confidence != 50 {
			t.Errorf("Expected confidence to be set to global value")
		}
	})

	// Test with conflicting 'none' transformation
	t.Run("conflicting transformations - none after", func(t *testing.T) {
		_, err := prepareConfig(conflictingNoneYAML)
		if err == nil {
			t.Fatalf("Expected error due to conflicting 'none' transformation, got nil")
		}
	})

	// Test with conflicting 'none' transformation
	t.Run("conflicting transformations - none before", func(t *testing.T) {
		_, err := prepareConfig(conflictingNoneYAML2)
		if err == nil {
			t.Fatalf("Expected error due to conflicting 'none' transformation, got nil")
		}
	})

	// Test with invalid key format in YAML
	t.Run("invalid key format", func(t *testing.T) {
		_, err := prepareConfig(invalidKeyYAML)
		if err == nil {
			t.Fatalf("Expected error due to invalid key format, got nil")
		}
	})

	// Test with non-OAM compliant 'to' transformation
	t.Run("non-OAM compliant 'to' transformation", func(t *testing.T) {
		_, err := prepareConfig(nonOAMtoYAML)
		if err == nil {
			t.Fatalf("Expected error due to non-OAM compliant 'to' transformation, got nil")
		}
	})

	// Test with non-OAM compliant 'from' transformation
	t.Run("non-OAM compliant 'from' transformation", func(t *testing.T) {
		_, err := prepareConfig(nonOAMfromYAML)
		if err == nil {
			t.Fatalf("Expected error due to non-OAM compliant 'from' transformation, got nil")
		}
	})
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		expected  *Transformation
		expectErr bool
	}{
		{
			name:      "Valid key1",
			key:       "FQDN->IPAddress",
			expected:  &Transformation{From: "fqdn", To: "ipaddress"},
			expectErr: false,
		},
		{
			name:      "Valid key2",
			key:       "FQDN->IPAddress",
			expected:  &Transformation{From: "fqdn", To: "ipaddress"},
			expectErr: false,
		},
		{
			name:      "Invalid key delimiter",
			key:       "FQDN-IPAddress",
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "Empty key",
			key:       "",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := &Transformation{}
			if tt.name == "Valid key1" {
				tf.From = "FQDN"
				tf.To = "IPAddress"
			}
			err := tf.Split(tt.key)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tf.From != tt.expected.From || tf.To != tt.expected.To {
					t.Errorf("Expected From: %s, To: %s, got From: %s, To: %s", tt.expected.From, tt.expected.To, tf.From, tf.To)
				}
			}
		})
	}
}

func TestIsMatch(t *testing.T) {
	m := &Matches{
		to: map[string]struct{}{
			"ipaddress":    {},
			"domainrecord": {},
			"rirorg":       {},
		},
	}
	m2 := &Matches{}

	tests := []struct {
		name     string
		to       string
		expected bool
	}{
		{
			name:     "Existing match",
			to:       "ipaddress",
			expected: true,
		},
		{
			name:     "Non-existing match",
			to:       "tls",
			expected: false,
		},
		{
			name:     "Empty match",
			to:       "",
			expected: false,
		},
		{
			name:     "Nil match",
			to:       "",
			expected: false,
		},
	}

	var result bool
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Nil match" {
				result = m2.IsMatch(tt.to)
			} else {
				result = m.IsMatch(tt.to)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, but got %v", tt.expected, result)
			}
		})
	}
}
func TestCheckTransformations(t *testing.T) {
	conf := NewConfig()
	conf.Transformations = map[string]*Transformation{
		"FQDN->IPAddress": {
			From:       "fqdn",
			To:         "ipaddress",
			Priority:   1,
			Confidence: 80,
		},
		"FQDN->DomainRecord": {
			From:     "fqdn",
			To:       "domainrecord",
			Priority: 2,
		},
		"FQDN->ALL": {
			From:    "fqdn",
			To:      "all",
			Exclude: []string{"tls", "fqdn", "rirorg"},
		},
		"DomainRecord->ALL": {
			From:    "domainrecord",
			To:      "all",
			Exclude: []string{"fqdn"},
		},
	}
	conf2 := NewConfig()
	conf2.Transformations = map[string]*Transformation{
		"FQDN->IPAddress": {
			From: "fqdn",
			To:   "ipaddress",
		},
	}

	tests := []struct {
		name       string
		from       string
		tos        []string
		expectErr  bool
		errMessage string
		expected   *Matches
	}{
		{
			name:      "Valid transformation",
			from:      "fqdn",
			tos:       []string{"ipaddress"},
			expectErr: false,
			expected: &Matches{
				to: map[string]struct{}{
					"ipaddress": {},
				},
			},
		},
		{
			name:       "No match",
			from:       "fqdn",
			tos:        []string{"rirorg"},
			expectErr:  true,
			errMessage: "zero transformation matches in the session config",
			expected:   &Matches{to: make(map[string]struct{})}},
		{
			name:      "Transformation to 'all'",
			from:      "fqdn",
			tos:       []string{"registrant", "rirorg"},
			expectErr: false,
			expected: &Matches{
				to: map[string]struct{}{
					"registrant": {},
				},
			},
		},
		{
			name:       "Transformation with excluded targets",
			from:       "fqdn",
			tos:        []string{"fqdn", "tls"},
			expectErr:  true,
			errMessage: "zero transformation matches in the session config",
			expected:   &Matches{to: make(map[string]struct{})}},
		{
			name:       "No \"from\" matches with config",
			from:       "ip",
			tos:        []string{"tls", "rirorg"},
			expectErr:  true,
			errMessage: "zero transformation matches in the session config",
			expected:   &Matches{to: make(map[string]struct{})}},
		{
			name:       "No \"to\" matches with config",
			from:       "domainrecord",
			tos:        []string{"fqdn"},
			expectErr:  true,
			errMessage: "zero transformation matches in the session config",
			expected:   &Matches{to: make(map[string]struct{})}},
		{
			name:       "Nil \"to\" matches with config",
			from:       "fqdn",
			tos:        []string{"rirorg"},
			expectErr:  true,
			errMessage: "zero transformation matches in the session config",
			expected:   &Matches{to: make(map[string]struct{})}},
	}

	var err error
	var matches *Matches
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "Nil \"to\" matches with config" {
				matches, err = conf.CheckTransformations(tt.from, tt.tos...)
			} else {
				matches, err = conf2.CheckTransformations(tt.from, tt.tos...)
			}
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if matches != nil && tt.expected != nil && !reflect.DeepEqual(*matches, *tt.expected) {
				t.Errorf("Expected matches: %v, but got: %v", tt.expected, matches)
			}
		})
	}
}
