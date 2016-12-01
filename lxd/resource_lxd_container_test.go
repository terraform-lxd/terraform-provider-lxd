package lxd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/lxc/lxd/shared"
)

func TestAccContainer_basic1(t *testing.T) {
	var container shared.ContainerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(t, "lxd_container.container1", &container),
				),
			},
		},
	})
}

func testAccContainerRunning(t *testing.T, n string, container *shared.ContainerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*LxdProvider).Client
		ct, err := client.ContainerInfo(rs.Primary.ID)
		if err != nil {
			return err
		}

		if ct != nil {
			*container = *ct
			return nil
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

var testAccContainer_basic = `
  resource "lxd_container" "container1" {
    name = "container1"
    image = "ubuntu"
    profiles = ["default"]
  }`
