package lxd

import (
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccLxdContainer_importBasic(t *testing.T) {
	containerName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_container.container1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_basic(containerName),
			},

			resource.TestStep{
				ResourceName: resourceName,
				//				Config:            testAccContainer_basic(containerName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLxdContainer_importConfig(t *testing.T) {
	containerName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_container.container1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccContainer_config(containerName),
			},

			resource.TestStep{
				ResourceName: resourceName,
				//				Config:            testAccContainer_config(containerName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
