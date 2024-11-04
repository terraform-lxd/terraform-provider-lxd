package clustering_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccClusterGroupAssignment_basic(t *testing.T) {
	clusterGroupName := petname.Generate(2, "-")
	clusterGroupMemberName := "node-1"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterGroupAssignment_basic(clusterGroupName, clusterGroupMemberName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_cluster_group_assignment.group1_node1", "cluster_group", clusterGroupName),
					resource.TestCheckResourceAttr("incus_cluster_group_assignment.group1_node1", "member", clusterGroupMemberName),
				),
			},
		},
	})
}

func testAccClusterGroupAssignment_basic(clusterGroupName string, clusterGroupMemberName string) string {
	return fmt.Sprintf(`
resource "incus_cluster_group" "group1" {
  name   = "%[1]s"
}

resource "incus_cluster_group_assignment" "group1_node1" {
  cluster_group = incus_cluster_group.group1.name
  member        = "%[2]s"
}

node-1
`, clusterGroupName, clusterGroupMemberName)
}
