package cluster_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccClusterGroup_basic(t *testing.T) {
	groupName := acctest.GenerateName(2, "-")
	resourceName := "lxd_cluster_group.group"

	// Checked before the test case, because the configurations below need a member name.
	members := acctest.PreCheckClustering(t, 1)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "clustering_groups")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// An empty member list must land in state as an empty set, not as null.
				Config: acctest.Provider() + testAccClusterGroup(groupName, "", nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "members.#", "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccClusterGroup(groupName, "x86-64 members", members[:1]),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", groupName),
					resource.TestCheckResourceAttr(resourceName, "description", "x86-64 members"),
					resource.TestCheckResourceAttr(resourceName, "members.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "members.*", members[0]),
				),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        groupName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			// The group still has a member here, so the final destroy must remove it before
			// deleting the group.
		},
	})
}

func testAccClusterGroup(name string, description string, members []string) string {
	return fmt.Sprintf(`
resource "lxd_cluster_group" "group" {
  name        = %q
  description = %q
  members     = [%s]
}
`, name, description, acctest.QuoteStrings(members))
}
