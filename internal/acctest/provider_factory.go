package acctest

import (
	"sync"

	lxd_config "github.com/canonical/lxd/lxc/config"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/provider"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// TestImage is a constant that specifies the default image used in all tests.
const TestImage = "ubuntu-minimal-daily:22.04"
// TestCachedImage is a constant that specifies the default image used in image caching tests.
const TestCachedImage = "ubuntu-minimal-daily:20.04"

var testProviderConfig *provider_config.LxdProviderConfig
var testProviderMutex sync.Mutex

// testProvider returns a LxdProviderConfig that is initialized with default
// LXD config.
//
// NOTE: This means this provider can differ from the actual provider used
// within the test. Therefore, it should be used exclusively for test prechecks
// because we assume all tests are run locally.
func testProvider() *provider_config.LxdProviderConfig {
	testProviderMutex.Lock()
	defer testProviderMutex.Unlock()

	if testProviderConfig == nil {
		config := lxd_config.DefaultConfig()
		acceptClientCert := true
		testProviderConfig = provider_config.NewLxdProvider(config, acceptClientCert)
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
