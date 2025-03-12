package server_test

import (
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccInfo_standalone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInfo(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.lxd_info.self", "remote"),
					resource.TestCheckResourceAttr("data.lxd_info.self", "cluster_members.%", "0"),
					resource.TestCheckResourceAttr("data.lxd_info.self", "api_extensions.6", "etag"),      // A randomly selected API extension. Its position must never change.
					resource.TestCheckResourceAttr("data.lxd_info.self", "instance_types.0", "container"), // Containers are always supported and reported first in the list.
				),
			},
		},
	})
}

func TestAccInfo_cluster(t *testing.T) {
	members := acctest.PreCheckClustering(t, 1)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInfo(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.lxd_info.self", "remote"),
					resource.TestCheckResourceAttr("data.lxd_info.self", "cluster_members.%", strconv.Itoa(len(members))),
					resource.TestCheckResourceAttr("data.lxd_info.self", "api_extensions.6", "etag"),      // A randomly selected API extension. Its position must never change.
					resource.TestCheckResourceAttr("data.lxd_info.self", "instance_types.0", "container"), // Containers are always supported and reported first in the list.
				),
			},
		},
	})
}

func testAccInfo() string {
	return `data "lxd_info" "self" {}`
}
