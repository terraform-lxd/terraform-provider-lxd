package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccNetwork_DS_basic(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_DS_basic(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("data.lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("data.lxd_network.network", "description", "Terraform provider test network"),
					resource.TestCheckResourceAttr("data.lxd_network.network", "managed", "true"),
					resource.TestCheckResourceAttrWith("data.lxd_network.network", "ipv4_address", isCIDR),
					resource.TestCheckResourceAttrWith("data.lxd_network.network", "ipv6_address", isCIDR),
				),
			},
		},
	})
}

func TestAccNetwork_DS_config(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_DS_config(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("data.lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("data.lxd_network.network", "description", ""),
					resource.TestCheckResourceAttr("data.lxd_network.network", "managed", "true"),
					resource.TestCheckResourceAttr("data.lxd_network.network", "config.ipv4.nat", "false"),
					resource.TestCheckResourceAttr("data.lxd_network.network", "config.ipv6.nat", "false"),
				),
			},
		},
	})
}

func testAccNetwork_DS_basic(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name        = %q
  description = "Terraform provider test network"
}

data "lxd_network" "network" {
  name = lxd_network.network.name
}
  `, networkName)
}

func testAccNetwork_DS_config(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = %q

  config = {
    "ipv4.nat" = false
    "ipv6.nat" = false
  }
}

data "lxd_network" "network" {
  name = lxd_network.network.name
}
  `, networkName)
}
