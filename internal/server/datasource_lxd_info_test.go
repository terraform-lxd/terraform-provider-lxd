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
				),
			},
		},
	})
}

func testAccInfo() string {
	return `data "lxd_info" "self" {}`
}
