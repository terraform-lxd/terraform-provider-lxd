package acctest

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// PreCheck is a precheck that ensures test requirements, such as existing
// environment variables, are met. It should be included in every acc test.
func PreCheck(t *testing.T) {
	// if os.Getenv("TEST_LXD_IS_VERSION") == "" {
	// 	t.Fatal("TEST_LXD_IS_VERSION must be set for acceptance tests")
	// }

	// if os.Getenv("TEST_LXD_HAS_VIRTUALIZATION") == "" {
	// 	t.Fatal("TEST_LXD_HAS_VIRTUALIZATION must be set for acceptance tests")
	// }

	// if os.Getenv("TEST_LXD_IS_CLUSTERED") == "" {
	// 	t.Fatal("TEST_LXD_IS_CLUSTERED must be set for acceptance tests")
	// }

	// if os.Getenv("TEST_LXD_EXTENSIONS") == "" {
	// 	t.Fatal("TEST_LXD_EXTENSIONS must be set for acceptance tests")
	// }
}

// PrintResourceState is a test check function that prints the entire state
// of a resource with the given name. This check should be used only for
// debuging purposes.
//
// Example resource name: lxd_profile.profile2
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
