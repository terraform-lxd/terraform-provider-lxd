package lxd

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/lxc/lxd"
	"testing"
)

func TestAccComputeInstance_basic1(t *testing.T) {
	//	var instance compute.Instance

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("lxd_container.tf-lxd-acctest-basic"),
				),
			},
		},
	})
}

func testAccContainerRunning(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*lxd.Client)
		ct := getContainerState(client, rs.Primary.ID)
		if ct != nil {
			fmt.Printf("%+v\n", ct)

			return nil
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

const testAccComputeInstance_basic = `
resource "lxd_container" "tf-lxd-acctest-basic" {
  name = "tf-lxd-acctest-basic"
  image = "ubuntu"
  profiles = ["default"]
}`

const testAccComputeInstance_ssh_provisioner = `
resource "lxd_container" "tf-lxd-acctest-basic" {
  name = "tf-lxd-acctest-basic"
  image = "ubuntu"
  profiles = ["default"]
}`
