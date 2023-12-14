package incus

import (
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIncusProfile_importBasic(t *testing.T) {
	profileName := petname.Generate(2, "-")
	resourceName := "incus_profile.profile1"

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

func TestIncusProfile_importConfig(t *testing.T) {
	profileName := petname.Generate(2, "-")
	resourceName := "incus_profile.profile1"

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

func TestIncusProfile_importDevice(t *testing.T) {
	profileName := petname.Generate(2, "-")
	resourceName := "incus_profile.profile1"

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
