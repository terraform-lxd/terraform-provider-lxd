package auth_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccAuthGroup_DS_description(t *testing.T) {
	authGroup := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthGroup_DS_description(authGroup, "Initial auth group description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "description", "Initial auth group description"),
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "permissions.#", "0"),
				),
			},
		},
	})
}

func TestAccAuthGroup_DS_withPermissions(t *testing.T) {
	authGroup := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthGroup_DS_permissions(authGroup, []permission{
					{
						Entitlement: "admin",
						EntityType:  "server",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "permissions.#", "1"),
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "permissions.0.entitlement", "admin"),
					resource.TestCheckResourceAttr("data.lxd_auth_group.group", "permissions.0.entity_type", "server"),
				),
			},
		},
	})
}

func testAccAuthGroup_DS_description(name string, description string) string {
	return testAccAuthGroup_description(name, description) + `
		data "lxd_auth_group" "group" {
		  name = lxd_auth_group.group.name
		}
	`
}

func testAccAuthGroup_DS_permissions(name string, permissions []permission) string {
	return testAccAuthGroup_permissions(name, permissions) + `
		data "lxd_auth_group" "group" {
		  name = lxd_auth_group.group.name
		}
	`
}
