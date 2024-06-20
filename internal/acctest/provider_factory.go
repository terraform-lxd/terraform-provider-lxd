package acctest

import (
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	incus_config "github.com/lxc/incus/v6/shared/cliconfig"

	"github.com/lxc/terraform-provider-incus/internal/provider"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

// TestImage is a constant that specifies the default image used in all tests.
const TestImage = "images:alpine/edge/amd64"

var testProviderConfig *provider_config.IncusProviderConfig
var testProviderMutex sync.Mutex

// testProvider returns an IncusProviderConfig that is initialized with default
// Incus config.
//
// NOTE: This means this provider can differ from the actual provider used
// within the test. Therefore, it should be used exclusively for test prechecks
// because we assume all tests are run locally.
func testProvider() *provider_config.IncusProviderConfig {
	testProviderMutex.Lock()
	defer testProviderMutex.Unlock()

	if testProviderConfig == nil {
		config := incus_config.DefaultConfig()
		acceptClientCert := true
		testProviderConfig = provider_config.NewIncusProvider(config, acceptClientCert)
	}

	return testProviderConfig
}

// ProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"incus": providerserver.NewProtocol6WithError(provider.NewIncusProvider("test")()),
}
