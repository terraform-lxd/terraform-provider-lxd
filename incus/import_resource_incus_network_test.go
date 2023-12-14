package incus

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIncusNetwork_importBasic(t *testing.T) {
	resourceName := "incus_network.eth1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_basic(),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestIncusNetwork_importDescription(t *testing.T) {
	resourceName := "incus_network.eth1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccNetwork_desc(),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
