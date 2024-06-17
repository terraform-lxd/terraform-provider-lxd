package acctest

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// PreCheck is a precheck that ensures test requirements, such as existing
// environment variables, are met. It should be included in every acc test.
func PreCheck(t *testing.T) {
	// if os.Getenv("TEST_LXD_REQUIRED_VAR") == "" {
	// 	t.Fatal("TEST_LXD_REQUIRED_VAR must be set for acceptance tests")
	// }
}

// PreCheckLxdVersion skips the test if the server's version does not satisfy
// the provided version constraints. The version constraints are detailed at:
// https://pkg.go.dev/github.com/hashicorp/go-version#readme-version-constraints
func PreCheckLxdVersion(t *testing.T, versionConstraint string) {
	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		t.Fatal(err)
	}

	serverVersion := apiServer.Environment.ServerVersion
	ok, err := utils.CheckVersion(serverVersion, versionConstraint)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Skipf("Test %q skipped. LXD server version %q does not satisfy the version constraint %q", t.Name(), serverVersion, versionConstraint)
	}
}

// PreCheckAPIExtensions skips the test if the LXD server does not support
// the required extensions.
func PreCheckAPIExtensions(t *testing.T, extensions ...string) {
	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	missing := []string{}
	for _, e := range extensions {
		if !server.HasExtension(e) {
			missing = append(missing, e)
		}
	}

	if len(missing) > 0 {
		t.Skipf("Test %q skipped. LXD server is missing required extensions: %v", t.Name(), missing)
	}
}

// PreCheckVirtualization skips the test if the LXD server does not
// support virtualization.
func PreCheckVirtualization(t *testing.T) {
	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that LXD server supports qemu driver which is required for virtualization.
	if !strings.Contains(apiServer.Environment.Driver, "qemu") {
		t.Skipf("Test %q skipped. LXD server does not support virtualization.", t.Name())
	}
}

// PreCheckClustering skips the test if LXD server is not running
// in clustered mode.
func PreCheckClustering(t *testing.T) {
	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if !server.IsClustered() {
		t.Skipf("Test %q skipped. LXD server is not running in clustered mode.", t.Name())
	}
}

// PreCheckRoot skips the test if the user cannot escalate privileges without a password.
// Root is required for certain tests, such as creating a loopback device for storage.
// This ensures tests do not stop midway asking for password.
func PreCheckRoot(t *testing.T) {
	err := exec.Command("sudo", "-n", "true").Run()
	if err != nil {
		t.Skipf("Test %q skipped. Cannot escalate privilege without a password.", t.Name())
	}
}

// PrintResourceState is a test check function that prints the entire state
// of a resource with the given name. This check should be used only for
// debuging purposes.
//
// Example resource name: lxd_profile.profile2.
func PrintResourceState(t *testing.T, resName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Resource %q not found", resName)
		}

		fmt.Println(utils.ToPrettyJSON(rs))
		return nil
	}
}
