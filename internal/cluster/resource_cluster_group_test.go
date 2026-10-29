package cluster_test

import (
	"fmt"
	"regexp"
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

func TestAccClusterGroup_inUse(t *testing.T) {
	groupName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")
	resourceName := "lxd_cluster_group.group"

	members := acctest.PreCheckClustering(t, 1)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "clustering_groups", "clustering_groups_used_by")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A project restricted to the group keeps LXD from deleting the group.
				Config: acctest.Provider() + testAccClusterGroup_inUse(groupName, projectName, members[0], true, true),
				Check:  resource.TestCheckResourceAttr(resourceName, "members.#", "1"),
			},
			{
				// Removing the group while the project still references it must fail.
				Config:      acctest.Provider() + testAccClusterGroup_inUse(groupName, projectName, members[0], false, true),
				ExpectError: regexp.MustCompile(`in\s+use`),
			},
			{
				// The failed destroy must have left the member in the group, so the original
				// configuration plans no changes.
				Config:   acctest.Provider() + testAccClusterGroup_inUse(groupName, projectName, members[0], true, true),
				PlanOnly: true,
			},
			{
				// Remove the project first, so that the group can be destroyed at the end.
				Config: acctest.Provider() + testAccClusterGroup_inUse(groupName, projectName, members[0], true, false),
			},
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

func testAccClusterGroup_inUse(groupName string, projectName string, member string, withGroup bool, withProject bool) string {
	var config string

	// The project refers to the group by resource when both exist, so that the group is created
	// first, and by name otherwise, which is the same value and plans no change.
	groupRef := fmt.Sprintf("%q", groupName)

	if withGroup {
		groupRef = "lxd_cluster_group.group.name"
		config += fmt.Sprintf(`
resource "lxd_cluster_group" "group" {
  name    = %q
  members = [%q]
}
`, groupName, member)
	}

	if withProject {
		config += fmt.Sprintf(`
resource "lxd_project" "project" {
  name = %q
  config = {
    "restricted"                = "true"
    "restricted.cluster.groups" = %s
  }
}
`, projectName, groupRef)
	}

	return config
}
