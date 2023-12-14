package incus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/lxc/incus/shared/api"
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
					testAccCachedImageExists(t, "incus_cached_image.img1", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img1", &img),
				),
			},
		},
	})
}

func TestAccCachedImage_basicVM(t *testing.T) {
	var img api.Image

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.img1vm", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img1vm", &img),
					testAccCachedImageIsVM(&img),
				),
			},
		},
	})
}

func TestAccCachedImage_alias(t *testing.T) {
	var img api.Image
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
		},
	})
}

func TestAccCachedImage_copiedAlias(t *testing.T) {
	var img api.Image
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases2(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.img3", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img3", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
					testAccCachedImageContainsAlias(&img, "alpine/3.18"),
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
					testAccCachedImageExists(t, "incus_cached_image.img4", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img4", &img),
					testAccCachedImageContainsAlias(&img, "alpine/3.18/amd64"),
				),
			},
		},
	})
}

func TestAccCachedImage_aliasAlreadyExists(t *testing.T) {
	var img api.Image
	alias := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.exists1", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.exists1", &img),
					testAccCachedImageContainsAlias(&img, alias),
				),
			},
			{
				Config:      testAccCachedImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(`.*Image alias already exists on destination.*`),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.exists1", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.exists1", &img),
					testAccCachedImageContainsAlias(&img, alias),
				),
			},
		},
	})
}

func TestAccCachedImage_addRemoveAlias(t *testing.T) {
	var img api.Image
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_aliases(alias1),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias1),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
			{
				Config: testAccCachedImage_aliases(alias2),
				Check: resource.ComposeTestCheckFunc(
					testAccCachedImageExists(t, "incus_cached_image.img2", &img),
					resourceAccCachedImageCheckAttributes("incus_cached_image.img2", &img),
					testAccCachedImageContainsAlias(&img, alias2),
				),
			},
		},
	})
}

func TestAccCachedImage_project(t *testing.T) {
	var img api.Image
	var project api.Project
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCachedImage_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunning(t, "incus_project.project1", &project),
					testAccCachedImageExistsInProject(t, "incus_cached_image.img1", &img, projectName),
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
		client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
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

func testAccCachedImageExistsInProject(t *testing.T, n string, image *api.Image, project string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found in state: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		id := newCachedImageIDFromResourceID(rs.Primary.ID)
		client, err := testAccProvider.Meta().(*incusProvider).GetInstanceServer("")
		if err != nil {
			return err
		}
		client = client.UseProject(project)
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

func testAccCachedImageIsVM(img *api.Image) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if img.Type != "virtual-machine" {
			return fmt.Errorf("Cached image is not a virtual-machine as requested")
		}

		return nil
	}
}

func testAccCachedImage_basic() string {
	return `
resource "incus_cached_image" "img1" {
  source_remote = "images"
  source_image = "alpine/3.18"

  copy_aliases = true
}
	`
}

func testAccCachedImage_basicVM() string {
	return `
resource "incus_cached_image" "img1vm" {
  source_remote = "images"
  source_image = "alpine/3.18"
  type = "virtual-machine"

  copy_aliases = true
}
	`
}

func testAccCachedImage_aliases(aliases ...string) string {
	return fmt.Sprintf(`
resource "incus_cached_image" "img2" {
  source_remote = "images"
  source_image = "alpine/3.18"

  aliases = ["%s"]
  copy_aliases = false
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "incus_cached_image" "exists1" {
  source_remote = "images"
  source_image = "alpine/3.18"

  aliases = ["%s"]
  copy_aliases = false
}
	`, alias)
}

func testAccCachedImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "incus_cached_image" "exists1" {
  source_remote = "images"
  source_image = "alpine/3.18"

  aliases = ["%s"]
  copy_aliases = false
}

resource "incus_cached_image" "exists2" {
  source_remote = "images"
  source_image = "alpine/3.18"

  aliases = ["%s"]
  copy_aliases = false
}
	`, alias, alias)
}

func testAccCachedImage_aliases2(aliases ...string) string {
	return fmt.Sprintf(`
resource "incus_cached_image" "img3" {
  source_remote = "images"
  source_image = "alpine/3.18"

  aliases = ["alpine/3.18","%s"]
  copy_aliases = true
}
	`, strings.Join(aliases, `","`))
}

func testAccCachedImage_aliasCollision() string {
	return `
resource "incus_cached_image" "img4" {
  source_remote = "images"
  source_image = "alpine/3.18"

  aliases = ["alpine/3.18/amd64"]
  copy_aliases = true
}
	`
}

func testAccCachedImage_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.storage.volumes" = false
	"features.images" = false
	"features.storage.buckets" = false
	"features.profiles" = false
  }
}
resource "incus_cached_image" "img1" {
  source_remote = "images"
  source_image = "alpine/3.18"
  project = incus_project.project1.name
}
	`, project)
}
