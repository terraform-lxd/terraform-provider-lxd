package lxd

import (
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccContainer_importBasic(t *testing.T) {
	containerName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_container.container1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_basic(containerName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"wait_for_network",
				},
				ImportStateId: containerName + "/images:alpine/3.9/amd64",
			},
		},
	})
}

func TestAccContainer_importConfig(t *testing.T) {
	containerName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_container.container1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccContainer_config(containerName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"image",
					"wait_for_network",
				},
			},
		},
	})
}
