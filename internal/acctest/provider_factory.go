package acctest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lxdConfig "github.com/canonical/lxd/lxc/config"
	"github.com/canonical/lxd/shared"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/provider"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// TestImage is a constant that specifies the default image used in all tests.
const TestImage = "images:alpine/edge"

// TestCachedImage is a constant that specifies the default image used in image caching tests.
// NOTE: it must be different from TestImage otherwise tests running in parallel will race to
// use and delete that image causing random failures.
const TestCachedImage = "images:alpine/edge/cloud"

var TestCachedImageSourceRemote, TestCachedImageSourceImage, _ = strings.Cut(TestCachedImage, ":")

// DisableSecureBootConfigEntry contains the instance config entry to disable secure boot.
var DisableSecureBootConfigEntry = sync.OnceValue(func() string {
	server, err := testProvider().InstanceServer("", "")
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to LXD server: %v", err))
	}

	if server.HasExtension("instance_boot_mode") {
		return `"boot.mode" = "uefi-nosecureboot"`
	}

	return `"security.secureboot" = false`
})

var testProviderRemote *provider_config.LxdRemote
var testProviderConfig *provider_config.LxdProviderConfig
var testProviderMutex sync.Mutex

// testProvider returns a LxdProviderConfig that is initialized with default
// LXD config remote.
func testProvider() *provider_config.LxdProviderConfig {
	testProviderMutex.Lock()
	defer testProviderMutex.Unlock()

	if testProviderConfig == nil {
		var err error

		if testProviderRemote == nil {
			remote, err := parseDefaultLocalConfigRemote()
			if err != nil {
				panic(fmt.Sprintf("Failed to parse default local config remote: %v", err))
			}

			testProviderRemote = remote
		}

		testProviderConfig, err = provider_config.NewLxdProviderConfig("test", *testProviderRemote)
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize provider: %v", err))
		}
	}

	return testProviderConfig
}

// ProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"lxd": providerserver.NewProtocol6WithError(provider.NewLxdProvider("test")()),
}

// Provider returns a Terraform HCL provider block configured from
// the default LXD remote. It should be prepended to test resource configs.
func Provider() string {
	if testProviderRemote == nil {
		remote, err := parseDefaultLocalConfigRemote()
		if err != nil {
			panic(fmt.Sprintf("Failed to parse default local config remote: %v", err))
		}

		testProviderRemote = remote
	}

	if strings.HasPrefix(testProviderRemote.Address, "unix://") {
		return fmt.Sprintf(`
provider "lxd" {
  address  = %q
}
`, testProviderRemote.Address)
	}

	return fmt.Sprintf(`
provider "lxd" {
  address  = %q
  client_certificate = %q
  client_key = %q
  server_certificate_fingerprint = %q
}
`,
		testProviderRemote.Address,
		testProviderRemote.ClientCertificate,
		testProviderRemote.ClientKey,
		testProviderRemote.ServerCertificateFingerprint,
	)
}

func parseDefaultLocalConfigRemote() (*provider_config.LxdRemote, error) {
	config, configDir, err := loadLocalConfig("")
	if err != nil {
		return nil, err
	}

	remoteName := config.DefaultRemote
	remote, ok := config.Remotes[remoteName]
	if !ok {
		return nil, fmt.Errorf("Default remote %q not found in config", remoteName)
	}

	if remote.Protocol == "" {
		remote.Protocol = "lxd"
	}

	if remote.Protocol != "lxd" {
		return nil, fmt.Errorf("Default remote %q is using unsupported protocol %q: Only the lxd protocol is supported", remoteName, remote.Protocol)
	}

	r := &provider_config.LxdRemote{
		Address:  remote.Addr,
		Protocol: remote.Protocol,
	}

	if remote.AuthType == "tls" {
		// Load client certificate and key from config directory.
		clientCertPath := filepath.Join(configDir, "client.crt")
		clientCert, err := os.ReadFile(clientCertPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read client certificate %q: %v", clientCertPath, err)
		}

		clientKeyPath := filepath.Join(configDir, "client.key")
		clientKey, err := os.ReadFile(clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read client key %q: %v", clientKeyPath, err)
		}

		r.ClientCertificate = string(clientCert)
		r.ClientKey = string(clientKey)

		// Load server certificate and compute fingerprint if it exists.
		// If the certificate does not exist, continue without setting the fingerprint
		// and let the provider attempt to connect without it, as it might be trusted
		// by the system's CA store.
		serverCertPath := filepath.Join(configDir, "servercerts", remoteName+".crt")
		serverCert, err := shared.ReadCert(serverCertPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("Failed to read server certificate %q for remote %q: %v", serverCertPath, remoteName, err)
		}

		if serverCert != nil {
			r.ServerCertificateFingerprint = shared.CertFingerprint(serverCert)
		}
	}

	return r, nil
}

// loadLocalConfig loads the local LXD configuration from the specified directory or
// from the default location if no directory is provided.
// It returns the loaded configuration, the directory from which it was loaded.
func loadLocalConfig(configDir string) (*lxdConfig.Config, string, error) {
	// Determine LXD config directory.
	if configDir == "" {
		// Determine LXD configuration directory. First check for the presence
		// of the /var/snap/lxd directory. If the directory exists, return
		// snap's config path. Otherwise return the fallback path.
		_, err := os.Stat("/var/snap/lxd")
		if err == nil || os.IsExist(err) {
			configDir = "$HOME/snap/lxd/common/config"
		} else {
			configDir = "$HOME/.config/lxc"
		}
	}

	configDir = os.ExpandEnv(configDir)
	configPath := filepath.Join(configDir, "config.yml")

	config, err := lxdConfig.LoadConfig(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to load LXD config from %q: %v", configPath, err)
	}

	return config, configDir, nil
}
