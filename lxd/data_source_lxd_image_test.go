package lxd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccLxdImageLookupBasic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: lxdImageTestLookupBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLxdImageID("data.lxd_image.test"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "architecture", "amd64"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "os", "ubuntu"),
				),
			},
		},
	})
}

func testAccCheckLxdImageID(n string) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("data source didn't find matching LXD image: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("LXD Image data source ID not set")
		}
		return nil
	}
}

const lxdImageTestLookupBasic = `
data "lxd_image" "test" {
    remote      = "ubuntu"
    
    architecture= "amd64"
	release     = "xenial"

}
`
