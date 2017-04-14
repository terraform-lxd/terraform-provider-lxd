package lxd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccLxdImageLookup_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: lxdImageTestLookup_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLxdImageID("data.lxd_image.test"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "architecture", "amd64"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "os", "ubuntu"),
				),
			},
		},
	})
}

func TestAccLxdImageLookup_description(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: lxdImageTestLookup_description,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLxdImageID("data.lxd_image.test"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "architecture", "arm64"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "os", "ubuntu"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "release", "xenial"),
					resource.TestCheckResourceAttr("data.lxd_image.test", "version", "16.04"),
				),
			},
		},
	})
}

func TestAccLxdImageLookup_alias(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: lxdImageTestLookup_alias,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLxdImageID("data.lxd_image.alias_lookup"),
					resource.TestCheckResourceAttr("data.lxd_image.alias_lookup", "architecture", "amd64"),
					resource.TestCheckResourceAttr("data.lxd_image.alias_lookup", "os", "Centos"),
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

const lxdImageTestLookup_basic = `
data "lxd_image" "test" {
  remote       = "ubuntu"
  architecture = "amd64"
	release      = "xenial"
}
`

const lxdImageTestLookup_description = `
data "lxd_image" "test" {
  remote            = "ubuntu"
  description_regex = "ubuntu 16.04 LTS arm64 \\(release\\).*"
}
`

const lxdImageTestLookup_alias = `
data "lxd_image" "alias_lookup" {
  remote      = "Images"
  alias_regex = "centos/7/amd64"
}
`
