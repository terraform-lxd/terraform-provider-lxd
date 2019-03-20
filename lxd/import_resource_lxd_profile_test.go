package lxd

import (
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestLXDProfile_importBasic(t *testing.T) {
	profileName := strings.ToLower(petname.Generate(2, "-"))
	resourceName := "lxd_profile.profile1"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfileBasicConfig(profileName),
			},

			resource.TestStep{
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
			resource.TestStep{
				Config: testAccProfileSetConfig(profileName),
			},

			resource.TestStep{
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
			resource.TestStep{
				Config: testAccProfileDevice1Config(profileName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
