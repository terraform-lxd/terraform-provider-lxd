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
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccProfile_basic(profileName),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testLXDProfileDefault = `
resource "lxd_profile" "default" {
	name = "default"
}
`
