package auth_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccAuthIdentityProviderGroup_DS_groups(t *testing.T) {
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
				Config: acctest.Provider() + testAccAuthIdentityProviderGroup_DS_groups(idpGroup, groups),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.lxd_auth_identity_provider_group.idp_group", "name", idpGroup),
					resource.TestCheckResourceAttr("data.lxd_auth_identity_provider_group.idp_group", "groups.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.lxd_auth_identity_provider_group.idp_group", "groups.*", group1),
					resource.TestCheckTypeSetElemAttr("data.lxd_auth_identity_provider_group.idp_group", "groups.*", group2),
				),
			},
		},
	})
}

func testAccAuthIdentityProviderGroup_DS_groups(name string, groups []string) string {
	return testAccAuthIdentityProviderGroup_groups(name, groups, groups) + `
		data "lxd_auth_identity_provider_group" "idp_group" {
		  name = lxd_auth_identity_provider_group.idp_group.name
		}
	`
}
