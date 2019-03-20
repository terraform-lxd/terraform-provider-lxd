package lxd

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/dustinkirkland/golang-petname"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"strconv"

	"github.com/lxc/lxd/shared/api"
)

func TestAccCachedImage_basic(t *testing.T) {
	var img api.Image

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCachedImageBasicConfig(),
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
			resource.TestStep{
				Config: testAccCachedImageAliasesConfig(alias1, alias2),
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
			resource.TestStep{
				Config: testAccCachedImageAliasesConfig2(alias1, alias2),
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

func TestAccCachedImageAliasCollisionConfig(t *testing.T) {
	var img api.Image

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCachedImageAliasCollisionConfig(),
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
			resource.TestStep{
				Config: testAccCachedImageAliasExists1Config(alias),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.exists1", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.exists1", &img),
					testAccCachedImageContainsAlias(&img, alias),
				),
			},
			resource.TestStep{
				Config:      testAccCachedImageAliasExists2Config(alias),
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
			resource.TestStep{
				Config: testAccCachedImageAliasesConfig(alias1),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
				),
			},
			resource.TestStep{
				Config: testAccCachedImageAliasesConfig(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "lxd_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("lxd_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
			resource.TestStep{
				Config: testAccCachedImageAliasesConfig(alias2),
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
		client, err := testAccProvider.Meta().(*lxdProvider).GetContainerServer("")
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

func testAccCachedImageBasicConfig() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img1" {
  source_remote = "images"
  source_image = "alpine/3.9"

  copy_aliases = true
}
	`)
}

func testAccCachedImageAliasesConfig(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img2" {
  source_remote = "images"
  source_image = "alpine/3.9/i386"

  aliases = ["%s"]
  copy_aliases = false
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImageAliasExists1Config(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "images"
  source_image = "alpine/3.9/i386"

  aliases = ["%s"]
  copy_aliases = false
}
	`, alias)
}

func testAccCachedImageAliasExists2Config(alias string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "exists1" {
  source_remote = "images"
  source_image = "alpine/3.9/i386"

  aliases = ["%s"]
  copy_aliases = false
}

resource "lxd_cached_image" "exists2" {
  source_remote = "images"
  source_image = "alpine/3.9/amd64"

  aliases = ["%s"]
  copy_aliases = false
}
	`, alias, alias)
}

func testAccCachedImageAliasesConfig2(aliases ...string) string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img3" {
  source_remote = "images"
  source_image = "alpine/3.9"

  aliases = ["alpine/3.9","%s"]
  copy_aliases = true
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImageAliasCollisionConfig() string {
	return fmt.Sprintf(`
resource "lxd_cached_image" "img4" {
  source_remote = "images"
  source_image = "alpine/3.9/amd64"

  aliases = ["alpine/3.9/amd64"]
  copy_aliases = true
}
	`)
}
