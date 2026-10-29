package cluster_test

import (
	"fmt"
	"testing"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
)

func TestAccClusterLink_DS_basic(t *testing.T) {
	linkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "cluster_links")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					testAccClusterLink_DS_createPending(t, linkName)
				},
				Config: acctest.Provider() + testAccClusterLink_DS_basic(linkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_cluster_link.link", "name", linkName),
					resource.TestCheckResourceAttr("data.lxd_cluster_link.link", "description", "Backup cluster"),
					resource.TestCheckResourceAttr("data.lxd_cluster_link.link", "type", "bidirectional"),
					resource.TestCheckResourceAttr("data.lxd_cluster_link.link", "config.%", "1"),
					resource.TestCheckResourceAttr("data.lxd_cluster_link.link", "config.user.env", "test"),
					resource.TestCheckResourceAttr("data.lxd_cluster_link.link", "addresses.#", "0"),
				),
			},
		},
	})
}

// testAccClusterLink_DS_createPending creates a pending bidirectional cluster link through the
// API, which issues a trust token for the linked cluster, and removes the link when the test ends.
func testAccClusterLink_DS_createPending(t *testing.T, name string) {
	server := acctest.InstanceServer(t)

	linkPost := api.ClusterLinksPost{
		ClusterLinkPut: api.ClusterLinkPut{
			Description: "Backup cluster",
			Config:      map[string]string{"user.env": "test"},
		},
		Name: name,
		Type: api.ClusterLinkTypeBidirectional,
	}

	_, err := server.CreateIdentityClusterLinkToken(linkPost)
	if err != nil {
		t.Fatalf("Failed to create cluster link %q: %v", name, err)
	}

	t.Cleanup(func() {
		err := server.DeleteClusterLink(name)
		if err != nil && !errors.IsNotFoundError(err) {
			t.Logf("Failed to delete cluster link %q during cleanup: %v", name, err)
		}
	})
}

func testAccClusterLink_DS_basic(name string) string {
	return fmt.Sprintf(`
data "lxd_cluster_link" "link" {
  name = %q
}
	`, name)
}
