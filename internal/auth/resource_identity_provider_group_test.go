package auth_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccAuthIdentityProviderGroup_groups(t *testing.T) {
	idpGroup := acctest.GenerateName(2, "-")
	group1 := acctest.GenerateName(2, "-")
	group2 := acctest.GenerateName(2, "-")
	groups := []string{group1, group2}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Identity provider group with no mapped groups.
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups(idpGroup, groups, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "name", idpGroup),
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "groups.#", "0"),
				),
			},
			{
				// Map one group.
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups(idpGroup, groups, []string{group1}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "name", idpGroup),
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "groups.#", "1"),
					resource.TestCheckTypeSetElemAttr("lxd_auth_identity_provider_group.idp_group", "groups.*", group1),
				),
			},
			{
				// Map two groups.
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups(idpGroup, groups, groups),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "name", idpGroup),
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "groups.#", "2"),
					resource.TestCheckTypeSetElemAttr("lxd_auth_identity_provider_group.idp_group", "groups.*", group1),
					resource.TestCheckTypeSetElemAttr("lxd_auth_identity_provider_group.idp_group", "groups.*", group2),
				),
			},
			{
				// Remove all mapped groups.
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups(idpGroup, groups, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "name", idpGroup),
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "groups.#", "0"),
				),
			},
		},
	})
}

func TestAccAuthIdentityProviderGroup_unknownGroup(t *testing.T) {
	idpGroup := acctest.GenerateName(2, "-")
	group := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// LXD rejects a mapping to a group that does not exist.
				// LXD 5.21 says "Failed to write", newer versions say "Failed writing",
				// so match "expected number of rows" instead of the full error message.
				Config:      acctest.Provider() + testAccAuthIdentityProviderGroup_unknownGroup(idpGroup, group),
				ExpectError: regexp.MustCompile("expected number of rows"),
			},
			{
				// The failed create left nothing behind, so the same name can be created.
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups(idpGroup, []string{group}, []string{group}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "name", idpGroup),
					resource.TestCheckResourceAttr("lxd_auth_identity_provider_group.idp_group", "groups.#", "1"),
					resource.TestCheckTypeSetElemAttr("lxd_auth_identity_provider_group.idp_group", "groups.*", group),
				),
			},
		},
	})
}

func TestAccAuthIdentityProviderGroup_importEmpty(t *testing.T) {
	resourceName := "lxd_auth_identity_provider_group.idp_group"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups("idp-group-test", nil, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentityValue(resourceName, tfjsonpath.New("name"), knownvalue.StringExact("idp-group-test")),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccAuthIdentityProviderGroup_importWithGroups(t *testing.T) {
	resourceName := "lxd_auth_identity_provider_group.idp_group"
	groups := []string{"idp-group-test-group1", "idp-group-test-group2"}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups("idp-group-test", groups, groups),
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccAuthIdentityProviderGroup_importIDRejected(t *testing.T) {
	resourceName := "lxd_auth_identity_provider_group.idp_group"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "access_management")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_groups("idp-group-test", nil, nil),
			},
			{
				ResourceName:  resourceName,
				ImportStateId: "idp-group-test",
				ImportState:   true,
				ExpectError:   regexp.MustCompile(`does not support import using ID`),
			},
		},
	})
}

// testAccAuthIdentityProviderGroup_groups creates an auth group for each name in groups and
// an identity provider group mapped to the auth groups named in mapped.
func testAccAuthIdentityProviderGroup_groups(name string, groups []string, mapped []string) string {
	var authGroups []string
	for _, group := range groups {
		authGroups = append(authGroups, fmt.Sprintf(`
		resource "lxd_auth_group" %[1]q {
		  name = %[1]q
		}
		`, group))
	}

	refs := make([]string, 0, len(mapped))
	for _, group := range mapped {
		refs = append(refs, fmt.Sprintf("lxd_auth_group.%s.name", group))
	}

	return strings.Join(authGroups, "") + fmt.Sprintf(`
		resource "lxd_auth_identity_provider_group" "idp_group" {
		  name   = %q
		  groups = [%s]
		}
	`, name, strings.Join(refs, ", "))
}

func testAccAuthIdentityProviderGroup_unknownGroup(name string, group string) string {
	return fmt.Sprintf(`
		resource "lxd_auth_identity_provider_group" "idp_group" {
		  name   = %q
		  groups = [%q]
		}
	`, name, group)
}
