package config

import "testing"

func TestDetermineLXDAddress(t *testing.T) {
	tests := []struct {
		Name      string
		Protocol  string
		Address   string
		Expect    string
		ExpectErr bool
	}{
		{
			Name:    "Empty address",
			Address: "",
			Expect:  "unix://",
		},
		{
			Name:    "Address starts with /",
			Address: "/path/to/socket",
			Expect:  "unix:///path/to/socket",
		},
		{
			Name:    "Address does not start with /",
			Address: "localhost:8443",
			Expect:  "https://localhost:8443",
		},
		{
			Name:    "Only hostname | Protocol lxd",
			Address: "localhost",
			Expect:  "https://localhost:8443",
		},
		{
			Name:     "Only hostname | Protocol simplestreams",
			Protocol: "simplestreams",
			Address:  "localhost",
			Expect:   "https://localhost:443",
		},
		{
			Name:    "Scheme and hostname | Protocol lxd",
			Address: "https://localhost",
			Expect:  "https://localhost:8443",
		},
		{
			Name:     "Scheme and hostname | Protocol simplestreams",
			Protocol: "simplestreams",
			Address:  "https://localhost",
			Expect:   "https://localhost:443",
		},
		{
			Name:    "Scheme, hostname, port | URL",
			Address: "https://localhost:1234",
			Expect:  "https://localhost:1234",
		},
		{
			Name:     "Scheme, hostname, port | URL path",
			Protocol: "simplestreams",
			Address:  "https://example.com/cloud-images/releases",
			Expect:   "https://example.com:443/cloud-images/releases",
		},
		{
			Name:     "Scheme, hostname, port | URL path with preconfigured port",
			Protocol: "simplestreams",
			Address:  "https://example.com:1234/cloud-images/releases",
			Expect:   "https://example.com:1234/cloud-images/releases",
		},
		// Expected errors.
		{
			Name:      "Unsupported simplestreams scheme",
			Protocol:  "simplestreams",
			Address:   "/path/to/socket",
			ExpectErr: true,
		},
		{
			Name:      "Missing hostname",
			Address:   "https://:8443",
			ExpectErr: true,
		},
		{
			Name:      "Unsupported scheme",
			Address:   "http://localhost:8443",
			ExpectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			addr, err := DetermineLXDAddress(test.Protocol, test.Address)
			if err != nil && !test.ExpectErr {
				t.Fatalf("Unexpected error: %v", err)
			}

			if err == nil && test.ExpectErr {
				t.Fatalf("Expected an error, but got none")
			}

			if addr != test.Expect {
				t.Fatalf("Expected address %q, got %q", test.Expect, addr)
			}
		})
	}
}
