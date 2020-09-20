package lxd

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"strconv"

	"github.com/lxc/lxd/shared/api"
)

func TestAccCachedImage_basic(t *testing.T) {
	var img api.Image

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img1", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img1", &img),
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
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
		},
	})
}

func TestAccCachedImage_copiedAlias(t *testing.T) {
	var img api.Image
	alias1 := strings.ToLower(petname.Generate(2, "-"))
	alias2 := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases2(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img3", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img3", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
					testAccCachedImageContainsAlias(&img, "alpine/3.9"),
				),
			},
		},
	})
}

func TestAccCachedImage_aliasCollision(t *testing.T) {
	var img api.Image

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliasCollision(),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img4", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img4", &img),
					testAccCachedImageContainsAlias(&img, "alpine/3.9/amd64"),
				),
			},
		},
	})
}

func TestAccCachedImage_aliasAlreadyExists(t *testing.T) {
	var img api.Image
	alias := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.exists1", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.exists1", &img),
					testAccCachedImageContainsAlias(&img, alias),
				),
			},
			{
				Config:      testAccCachedImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(`.*Image alias already exists on destination.*`),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.exists1", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.exists1", &img),
					testAccCachedImageContainsAlias(&img, alias),
				),
			},
		},
	})
}

func TestAccCachedImage_addRemoveAlias(t *testing.T) {
	var img api.Image
	alias1 := strings.ToLower(petname.Generate(2, "-"))
	alias2 := strings.ToLower(petname.Generate(2, "-"))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img2", &img),
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

		id := newCachedImageIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*lxdProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		img, _, err := client.GetImage(id.fingerprint)
		if err != nil {
			return err
		}

		if img != nil {
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

func resourceAccCachedImageCheckAttributes(n string, img *api.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found in state: %s", n)
		}

		if rs.Primary.Attributes["architecture"] != img.Architecture {
			return fmt.Errorf("architecture doesn't match: %s / %s", rs.Primary.Attributes["architecture"], img.Architecture)
		}

		if rs.Primary.Attributes["fingerprint"] != img.Fingerprint {
			return fmt.Errorf("fingerprint doesn't match")
		}

		if rs.Primary.Attributes["created_at"] != strconv.FormatInt(img.CreatedAt.Unix(), 10) {
			return fmt.Errorf("created_at doesn't match")
		}

		return nil

	}
}

func testAccCachedImage_basic() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img1" {
  source_remote = "images"
  source_image = "alpine/3.9"

  copy_aliases = true
}
	`)
}

func testAccCachedImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img2" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["%s"]
  copy_aliases = false
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["%s"]
  copy_aliases = false
}
	`, alias)
}

func testAccCachedImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["%s"]
  copy_aliases = false
}

resource "lxd_cached_image" "exists2" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["%s"]
  copy_aliases = false
}
	`, alias, alias)
}

func testAccCachedImage_aliases2(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img3" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["alpine/3.9","%s"]
  copy_aliases = true
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasCollision() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img4" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["alpine/3.9/amd64"]
  copy_aliases = true
}
	`)
}
