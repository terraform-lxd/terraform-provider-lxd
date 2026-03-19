package provider_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccProvider_unixSocket(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure provider connects via unix socket using default address.
				Config: testAccProvider_unixSocket(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "auth_user_method", "unix"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
		},
	})
}

func TestAccProvider_bearerToken(t *testing.T) {
	token, cleanup := acctest.ConfigureBearerToken(t)
	defer cleanup()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckLocalServerHTTPS(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure authentication succeeds with a valid bearer token.
				Config: testAccProvider_bearerToken(token),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "auth_user_method", "bearer"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
		},
	})
}

func TestAccProvider_bearerTokenFile(t *testing.T) {
	token, cleanup := acctest.ConfigureBearerToken(t)
	defer cleanup()

	tokenFile := filepath.Join(t.TempDir(), "bearer_token")
	err := os.WriteFile(tokenFile, []byte(token), 0600)
	if err != nil {
		t.Fatalf("Failed to write bearer token to file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckLocalServerHTTPS(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure authentication succeeds when bearer token is read from a file.
				Config: testAccProvider_bearerTokenFile(tokenFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "auth_user_method", "bearer"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
		},
	})
}

func TestAccProvider_mtls(t *testing.T) {
	clientCert, clientKey, cleanup := acctest.ConfigureMutualTLS(t)
	defer cleanup()

	serverCertFingerprint := acctest.GetServerCertificateFingerprint(t)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckLocalServerHTTPS(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure provider connects using inline mTLS client certificate and key.
				Config: testAccProvider_mtls(clientCert, clientKey, serverCertFingerprint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "auth_user_method", "tls"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
		},
	})
}

func TestAccProvider_mtlsFromFile(t *testing.T) {
	clientCert, clientKey, cleanup := acctest.ConfigureMutualTLS(t)
	defer cleanup()

	serverCertFingerprint := acctest.GetServerCertificateFingerprint(t)

	// Write credentials to temporary files.
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "client.crt")
	keyFile := filepath.Join(tmpDir, "client.key")

	err := os.WriteFile(certFile, []byte(clientCert), 0600)
	if err != nil {
		t.Fatalf("Failed to write client certificate file: %v", err)
	}

	err = os.WriteFile(keyFile, []byte(clientKey), 0600)
	if err != nil {
		t.Fatalf("Failed to write client key file: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckLocalServerHTTPS(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure provider connects using mTLS certificate and key from files.
				Config: testAccProvider_mtlsFromFile(certFile, keyFile, serverCertFingerprint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "auth_user_method", "tls"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
		},
	})
}

func TestAccProvider_mtlsTrustToken(t *testing.T) {
	// Generate client certificate and key without adding to the server's trust store.
	clientCert, clientKey, cleanup := acctest.GenerateClientCertificate(t)
	defer cleanup()

	serverCertFingerprint := acctest.GetServerCertificateFingerprint(t)
	invalidToken := `eyJjbGllbnRfbmFtZSI6InRtcF90b2tlbiIsImZpbmdlcnByaW50IjoiWW91X2hhdmVfZGVjb2RlZF9hX3RlbXBvcmFyeV90b2tlbi5Db25ncmF0dWxhdGlvbnMhIiwiYWRkcmVzc2VzIjpbIjEyNy4wLjAuMTo4NDQzIl0sInNlY3JldCI6IlRoaXNfaXNfYV90b3Bfc2VjcmV0LkRvX25vdF90ZWxsX2l0X3RvX2FueW9uZSEiLCJleHBpcmVzX2F0IjoiMDAwMS0wMS0wMVQwMDowMDowMFoifQo=`

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckLocalServerHTTPS(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure authentication fails with untrusted client certificate (no trust token).
				Config:      testAccProvider_trustToken(clientCert, clientKey, "", serverCertFingerprint),
				ExpectError: regexp.MustCompile(`Unable to authenticate with\s+remote server`),
			},
			{
				// Ensure authentication fails with an invalid trust token.
				Config:      testAccProvider_trustToken(clientCert, clientKey, invalidToken, serverCertFingerprint),
				ExpectError: regexp.MustCompile(`(?s)(fingerprint.*does not match|Unable to authenticate)`),
			},
			{
				// Ensure authentication succeeds with a valid trust token.
				Config: testAccProvider_trustToken(clientCert, clientKey, acctest.ConfigureTrustToken(t), ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "remote", "tf-remote"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
			{
				// Ensure authentication succeeds without a trust token once trusted.
				Config: testAccProvider_trustToken(clientCert, clientKey, "", serverCertFingerprint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "remote", "tf-remote"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttrSet("lxd_noop.noop", "server_version"),
				),
			},
		},
	})
}

func TestAccProvider_serverCertificateFingerprint(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
			acctest.PreCheckLocalServerHTTPS(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure connection fails when an incorrect server certificate fingerprint is provided.
				Config:      testAccProvider_serverCertFingerprint("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
				ExpectError: regexp.MustCompile(`fingerprint mismatch`),
			},
		},
	})
}

func TestAccProvider_conflictBearerTokenAndClientCert(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure an error is returned when both bearer_token and client_certificate are set.
				Config:      testAccProvider_conflictBearerTokenAndClientCert(),
				ExpectError: regexp.MustCompile(`cannot be specified when`),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccProvider_conflictBearerTokenAndBearerTokenFile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure an error is returned when both bearer_token and bearer_token_file are set.
				Config:      testAccProvider_conflictBearerTokenAndBearerTokenFile(),
				ExpectError: regexp.MustCompile(`cannot be specified when`),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccProvider_incompleteMtls(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure an error is returned when client_certificate is set without client_key.
				Config:      testAccProvider_incompleteMtlsCertOnly(),
				ExpectError: regexp.MustCompile(`(?i)both.*client.certificate.*and.*client.key.*must be provided`),
			},
			{
				// Ensure an error is returned when client_key is set without client_certificate.
				Config:      testAccProvider_incompleteMtlsKeyOnly(),
				ExpectError: regexp.MustCompile(`(?i)both.*client.certificate.*and.*client.key.*must be provided`),
			},
		},
	})
}

func TestAccProvider_invalidProtocol(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure an error is returned when an unsupported protocol is configured.
				Config:      testAccProvider_invalidProtocol(),
				ExpectError: regexp.MustCompile(`Attribute remote\[0\].protocol value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccProvider_multipleRemotes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure multiple remotes work correctly when a default_remote is specified.
				Config: testAccProvider_multipleRemotes(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_noop.noop", "project", "default"),
					resource.TestCheckResourceAttr("lxd_noop.noop", "auth_user_method", "unix"),
				),
			},
		},
	})
}

func TestAccProvider_requireDefaultRemote(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Ensure an error is returned when a default remote is not specified.
				Config:      testAccProvider_requireDefaultRemote(),
				ExpectError: regexp.MustCompile(`When multiple remotes are defined, a default remote must be specified`),
				PlanOnly:    true,
			},
		},
	})
}

// testAccProvider_unixSocket returns a provider config that uses the default unix socket.
func testAccProvider_unixSocket() string {
	return `
provider "lxd" {
  remote {
    name    = "local"
    address = "unix://"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_bearerToken returns a provider config that uses bearer token authentication.
func testAccProvider_bearerToken(token string) string {
	return fmt.Sprintf(`
provider "lxd" {
  remote {
    name         = "https-remote"
    address      = "https://127.0.0.1:8443"
    bearer_token = %q
  }
}

resource "lxd_noop" "noop" {}
`, token)
}

// testAccProvider_bearerTokenFile returns a provider config that reads the bearer token from a file.
func testAccProvider_bearerTokenFile(tokenFile string) string {
	return fmt.Sprintf(`
provider "lxd" {
  remote {
    name              = "https-remote"
    address           = "https://127.0.0.1:8443"
    bearer_token_file = %q
  }
}

resource "lxd_noop" "noop" {}
`, tokenFile)
}

// testAccProvider_mtls returns a provider config that uses inline mTLS credentials.
func testAccProvider_mtls(clientCert, clientKey, serverFingerprint string) string {
	return fmt.Sprintf(`
provider "lxd" {
  remote {
    name 			   = "https-remote"
    address                        = "https://127.0.0.1:8443"
    client_certificate             = %q
    client_key                     = %q
    server_certificate_fingerprint = %q
  }
}

resource "lxd_noop" "noop" {}
`, clientCert, clientKey, serverFingerprint)
}

// testAccProvider_mtlsFromFile returns a provider config that reads mTLS credentials from files.
func testAccProvider_mtlsFromFile(certFile, keyFile, serverFingerprint string) string {
	return fmt.Sprintf(`
provider "lxd" {
  remote {
    name   		           = "https-remote"
    address                        = "https://127.0.0.1:8443"
    client_certificate_file        = %q
    client_key_file                = %q
    server_certificate_fingerprint = %q
  }
}

resource "lxd_noop" "noop" {}
`, certFile, keyFile, serverFingerprint)
}

func testAccProvider_trustToken(clientCert string, clientKey string, trustToken string, serverFingerprint string) string {
	return fmt.Sprintf(`
provider "lxd" {
  remote {
    name                           = "tf-remote"
    protocol                       = "lxd"
    address                        = "https://127.0.0.1:8443"
    trust_token                    = %q
    client_certificate             = %q
    client_key                     = %q
    server_certificate_fingerprint = %q
  }
}

resource "lxd_noop" "noop" {
  remote = "tf-remote"
}
`, trustToken, clientCert, clientKey, serverFingerprint)
}

// testAccProvider_conflictBearerTokenAndClientCert returns a provider config with both bearer token and client cert set.
func testAccProvider_conflictBearerTokenAndClientCert() string {
	return `
provider "lxd" {
  remote {
    name               = "https-remote"
    address            = "https://127.0.0.1:8443"
    bearer_token       = "some-token"
    client_certificate = "some-cert"
    client_key         = "some-key"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_conflictBearerTokenAndBearerTokenFile returns a provider config with both bearer token and bearer token file set.
func testAccProvider_conflictBearerTokenAndBearerTokenFile() string {
	return `
provider "lxd" {
  remote {
    name              = "https-remote"
    address           = "https://127.0.0.1:8443"
    bearer_token      = "some-token"
    bearer_token_file = "/tmp/token"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_incompleteMtlsCertOnly returns a provider config with client_certificate but no client_key.
func testAccProvider_incompleteMtlsCertOnly() string {
	return `
provider "lxd" {
  remote {
    name               = "https-remote"
    address            = "https://127.0.0.1:8443"
    client_certificate = "some-cert"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_incompleteMtlsKeyOnly returns a provider config with client_key but no client_certificate.
func testAccProvider_incompleteMtlsKeyOnly() string {
	return `
provider "lxd" {
  remote {
    name       = "https-remote"
    address    = "https://127.0.0.1:8443"
    client_key = "some-key"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_invalidProtocol returns a provider config with an unsupported protocol.
func testAccProvider_invalidProtocol() string {
	return `
provider "lxd" {
  remote {
    name     = "https-remote"
    address  = "https://127.0.0.1:8443"
    protocol = "unsupported"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_serverCertFingerprint returns a provider config with a server certificate fingerprint.
func testAccProvider_serverCertFingerprint(fingerprint string) string {
	return fmt.Sprintf(`
provider "lxd" {
  remote {
    name                           = "https-remote"
    address                        = "https://127.0.0.1:8443"
    server_certificate_fingerprint = %q
  }
}

resource "lxd_noop" "noop" {}
`, fingerprint)
}

// testAccProvider_multipleRemotes returns a provider config with multiple remotes
// that require a default remote to be specified.
func testAccProvider_multipleRemotes() string {
	return `
provider "lxd" {
  default_remote = "local1"

  remote {
    name     = "local1"
    address  = "unix://"
  }

  remote {
    name     = "local2"
    address  = "unix://"
  }
}

resource "lxd_noop" "noop" {}
`
}

// testAccProvider_requireDefaultRemote returns a provider config with multiple remotes
// that require a default remote to be specified.
func testAccProvider_requireDefaultRemote() string {
	return `
provider "lxd" {
  remote {
    name     = "local1"
    address  = "unix://"
  }

  remote {
    name     = "local2"
    address  = "unix://"
  }
}

resource "lxd_noop" "noop" {}
`
}
