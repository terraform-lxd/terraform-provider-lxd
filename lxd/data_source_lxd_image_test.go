package lxd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/lxc/lxd/shared"
)

func TestAccLxdImageLookupBasic(t *testing.T) {
	var container shared.ContainerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: lxdImageTestLookupBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLxdImageId("data.lxd_image.test"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "architecture", "amd64"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "os", "1"),
					resource.TestMatchResourceAttr("data.lxd_image.test", "creation_date", regexp.MustCompile("^20[0-9]{2}-")),
					resource.TestMatchResourceAttr("data.lxd_image.test", "description", regexp.MustCompile("^Amazon Linux AMI")),
				),
			},
		},
	})
}

func testAccCheckLxdImageId(n string) resource.TestCheckFunc {
	// Wait for IAM role
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find AMI data source: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("AMI data source ID not set")
		}
		return nil
	}
}

const lxdImageTestLookupBasic = `
data "lxd_image" "test" {
    remote      = "Images"
    
    arch        = "amd64"
    filter {
        name = ""
        value = "Ubuntu xenial arm64*"
    } 
}
`
