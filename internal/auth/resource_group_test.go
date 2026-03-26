package auth_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

type permission struct {
	Entitlement string
	EntityType  string
	EntityArgs  map[string]string
}

func TestAccAuthGroup_description(t *testing.T) {
	authGroup := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthGroup_description(authGroup, "Initial auth group description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "description", "Initial auth group description"),
				),
			},
			{
				Config: acctest.Provider() + testAccAuthGroup_description(authGroup, "Updated auth group description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "description", "Updated auth group description"),
				),
			},
		},
	})
}

func TestAccAuthGroup_permissions(t *testing.T) {
	authGroup := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Auth group with no permissions.
				Config: acctest.Provider() + testAccAuthGroup_permissions(authGroup, []permission{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "0"),
				),
			},
			{
				// Add permission.
				Config: acctest.Provider() + testAccAuthGroup_permissions(authGroup, []permission{
					{
						Entitlement: "admin",
						EntityType:  "server",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "1"),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.0.entitlement", "admin"),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.0.entity_type", "server"),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.0.entity_args.%", "0"),
				),
			},
			{
				// Replace permissions.
				Config: acctest.Provider() + testAccAuthGroup_permissions(authGroup, []permission{
					{
						Entitlement: "can_view_projects",
						EntityType:  "server",
					},
					{
						Entitlement: "storage_pool_manager",
						EntityType:  "server",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement": "can_view_projects",
							"entity_type": "server",
						},
					),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement": "storage_pool_manager",
							"entity_type": "server",
						},
					),
				),
			},
			{
				// Add permission.
				Config: acctest.Provider() + testAccAuthGroup_permissions(authGroup, []permission{
					{
						Entitlement: "storage_pool_manager",
						EntityType:  "server",
					},
					{
						Entitlement: "can_view_projects",
						EntityType:  "server",
					},
					{
						Entitlement: "can_edit",
						EntityType:  "project",
						EntityArgs: map[string]string{
							"name": "default",
						},
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement":      "can_edit",
							"entity_type":      "project",
							"entity_args.name": "default",
						},
					),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement": "storage_pool_manager",
							"entity_type": "server",
						},
					),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement": "can_view_projects",
							"entity_type": "server",
						},
					),
				),
			},
			{
				// Remove permissions.
				Config: acctest.Provider() + testAccAuthGroup_permissions(authGroup, []permission{}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", authGroup),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "0"),
				),
			},
		},
	})
}

func TestAccAuthGroup_project(t *testing.T) {
	name := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthGroup_instanceInProject(name, "auth-group-test-project"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", name),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement":      "can_view",
							"entity_type":      "project",
							"entity_args.name": "auth-group-test-project",
						},
					),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement":         "can_view",
							"entity_type":         "instance",
							"entity_args.name":    name,
							"entity_args.project": "auth-group-test-project",
						},
					),
				),
			},
			{
				Config: acctest.Provider() + testAccAuthGroup_instanceInProject(name, "default"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_group.group", "name", name),
					resource.TestCheckResourceAttr("lxd_auth_group.group", "permissions.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement":      "can_view",
							"entity_type":      "project",
							"entity_args.name": "default",
						},
					),
					resource.TestCheckTypeSetElemNestedAttrs("lxd_auth_group.group", "permissions.*",
						map[string]string{
							"entitlement":         "can_view",
							"entity_type":         "instance",
							"entity_args.name":    name,
							"entity_args.project": "default",
						},
					),
				),
			},
		},
	})
}

func TestAccAuthGroup_importEmpty(t *testing.T) {
	resourceName := "lxd_auth_group.group"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthGroup_description("auth-group-test", "mydesc"),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "auth-group-test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccAuthGroup_importWithPermissions(t *testing.T) {
	resourceName := "lxd_auth_group.group"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthGroup_permissions("auth-group-test", []permission{
					{
						Entitlement: "admin",
						EntityType:  "server",
					},
				}),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "auth-group-test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccAuthGroup_description(name string, description string) string {
	return fmt.Sprintf(`
                resource "lxd_auth_group" "group" {
                  name        = %q
                  description = %q
                }
        `,
		name,
		description,
	)
}

func testAccAuthGroup_permissions(name string, permissions []permission) string {
	var perms []string
	for _, permission := range permissions {
		args := make([]string, 0, len(permission.EntityArgs))
		for k, v := range permission.EntityArgs {
			args = append(args, fmt.Sprintf("%q = %q", k, v))
		}

		perms = append(perms, fmt.Sprintf(`
                        {
                          entitlement = %q
                          entity_type = %q
                          entity_args = {
                            %s
                          }
                        }
                `,
			permission.Entitlement,
			permission.EntityType,
			strings.Join(args, "\n"),
		))
	}

	return fmt.Sprintf(`
                resource "lxd_auth_group" "group" {
                  name        = %q
                  permissions = [%s]
                }
        `,
		name,
		strings.Join(perms, ",\n"),
	)
}

func testAccAuthGroup_instanceInProject(name string, project string) string {
	resProject := fmt.Sprintf(`
		resource "lxd_project" "proj" {
		  count = %[1]q == "default" ? 0 : 1
		  name  = %[1]q
		  config = {
		    "features.images"   = false
		    "features.profiles" = false
		  }
		}
	`,
		project,
	)

	return resProject + fmt.Sprintf(`
                resource "lxd_instance" "inst" {
                  name    = %[1]q
                  project = try(lxd_project.proj[0].name, "default")
                  running = false
                }

                resource "lxd_auth_group" "group" {
                  name = %[1]q
                  permissions = [
                    {
                      entitlement = "can_view"
                      entity_type = "project"
                      entity_args = {
                        name = lxd_instance.inst.project
                      }
                    },
                    {
                      entitlement = "can_view"
                      entity_type = "instance"
                      entity_args = {
                        name    = lxd_instance.inst.name
                        project = lxd_instance.inst.project
                      }
                    }
                  ]
                }
        `,
		name,
	)
}
