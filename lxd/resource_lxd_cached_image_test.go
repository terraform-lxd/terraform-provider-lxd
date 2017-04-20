package lxd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/davecgh/go-spew/spew"
	"github.com/lxc/lxd/shared/api"
)

func TestAccCachedImage_basic(t *testing.T) {
	var img *api.Image

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCachedImage_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img1", img),
					resource.TestCheckResourceAttr("lxd_cached_image.img1", "fingerprint", img.Fingerprint),
				),
			},
		},
	})
}

func TestAccCachedImage_alias(t *testing.T) {
	var img api.Image
	alias1 := strings.ToLower(petname.Generate(2, "-"))
	alias2 := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
		},
	})
}

func testAccCachedImageExists(t *testing.T, n string, image *api.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found in state: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id := newCachedImageIdFromResourceId(rs.Primary.ID)
		client := testAccProvider.Meta().(*LxdProvider).Client
		img, err := client.GetImageInfo(id.fingerprint)
		if err != nil {
			return err
		}

		if img != nil {
			return fmt.Errorf("Image Info: %s", spew.Sdump(img))
			*image = *img
			return nil
		}

		return fmt.Errorf("Image not found: %s", rs.Primary.ID)
	}
}

func testAccCachedImageContainsAlias(img *api.Image, alias string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if img.Aliases == nil || len(img.Aliases) == 0 {
			return fmt.Errorf("No aliases")
		}

		for _, a := range img.Aliases {
			if a.Name != alias {
				continue
			}

			if a.Name == alias {
				return nil
			}
		}

		return fmt.Errorf("Alias not found: %s", alias)
	}
}

func testAccCachedImage_basic() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img1" {
  source_remote = "ubuntu"
  source_image = "t/amd64"

  copy_aliases = true
}
	`)
}

func testAccCachedImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img2" {
  source_remote = "ubuntu"
  source_image = "x/i386"

  alias = ["%s"]
  copy_aliases = false
}
	`, strings.Join(aliases, `","`))
}
