package lxd

import (
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestLXDProfile_importBasic(t *testing.T) {
	profileName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_basic(profileName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestLXDProfile_importConfig(t *testing.T) {
	profileName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_config(profileName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestLXDProfile_importDevice(t *testing.T) {
	profileName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccProfile_device_1(profileName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
