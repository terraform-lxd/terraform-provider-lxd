package server_test

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

func TestAccServer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test": "foo",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", "foo"),
				),
			},
			{
				// Revert the managed key to its default.
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", ""),
				),
			},
		},
	})
}

func TestAccServer_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test": "foo",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", "foo"),
				),
			},
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test":  "bar",
					"user.terraform-provider-test2": "baz",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", "bar"),
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test2", "baz"),
				),
			},
			{
				// Revert the managed keys to their defaults.
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"user.terraform-provider-test":  "",
					"user.terraform-provider-test2": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test", ""),
					resource.TestCheckResourceAttr("lxd_server.global", "config.user.terraform-provider-test2", ""),
				),
			},
		},
	})
}

func TestAccServer_localKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"core.bgp_routerid": "127.0.0.1",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.core.bgp_routerid", "127.0.0.1"),
				),
			},
			{
				// Revert the managed key to its default.
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"core.bgp_routerid": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config.core.bgp_routerid", ""),
				),
			},
		},
	})
}

func TestAccServer_invalidKeyRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccServer_config(map[string]string{
					"not.a.real.key": "value",
				}),
				ExpectError: regexp.MustCompile("is not a valid server configuration key"),
			},
		},
	})
}

func TestAccServer_clusterMemberOverrides(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	overrideTarget := targets[0]
	defaultTarget := targets[1]

	// core.bgp_routerid is a local key, so it is applied per cluster member.
	configKey := "core.bgp_routerid"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Set the local key for every member through the global config.
				Config: acctest.Provider() + testAccServer_memberOverrides(map[string]string{configKey: "127.0.0.1"}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config."+configKey, "127.0.0.1"),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "127.0.0.1"),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "127.0.0.1"),
				),
			},
			{
				// Override the local key on a single member.
				Config: acctest.Provider() + testAccServer_memberOverrides(map[string]string{configKey: "127.0.0.1"}, map[string]map[string]string{overrideTarget: {configKey: "127.0.0.2"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", "config."+configKey, "127.0.0.1"),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "127.0.0.2"),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "127.0.0.2"),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "127.0.0.1"),
				),
			},
			{
				// Set a different value on each member.
				Config: acctest.Provider() + testAccServer_memberOverrides(map[string]string{configKey: "127.0.0.1"}, map[string]map[string]string{overrideTarget: {configKey: "127.0.0.2"}, defaultTarget: {configKey: "127.0.0.3"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "127.0.0.2"),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "127.0.0.3"),
				),
			},
			{
				// Drop the overrides and revert every member to the global value.
				Config: acctest.Provider() + testAccServer_memberOverrides(map[string]string{configKey: ""}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("lxd_server.global", fmt.Sprintf("member_overrides.%s", overrideTarget)),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), ""),
					resource.TestCheckResourceAttr("lxd_server.global", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), ""),
				),
			},
		},
	})
}

func TestAccServer_clusterMemberOverridesGlobalKeyRejected(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckServerTests(t)
			acctest.PreCheckAPIExtensions(t, "metadata_configuration")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// images.auto_update_interval is a global key, so it cannot be set per member.
				Config:      acctest.Provider() + testAccServer_memberOverrides(nil, map[string]map[string]string{targets[0]: {"images.auto_update_interval": "15"}}),
				ExpectError: regexp.MustCompile(`Only member-specific keys are allowed in per-member\s+configuration`),
			},
			{
				Config:      acctest.Provider() + testAccServer_memberOverrides(nil, map[string]map[string]string{"non-existent-member": {"core.bgp_routerid": "127.0.0.1"}}),
				ExpectError: regexp.MustCompile(`non-existent\s+cluster member`),
			},
		},
	})
}

func testAccServer_config(config map[string]string) string {
	return testAccServer_memberOverrides(config, nil)
}

func testAccServer_memberOverrides(config map[string]string, memberOverrides map[string]map[string]string) string {
	var b strings.Builder
	b.WriteString(`
resource "lxd_server" "global" {
  config = {
`)

	for _, key := range slices.Sorted(maps.Keys(config)) {
		fmt.Fprintf(&b, "    %q = %q\n", key, config[key])
	}

	b.WriteString("  }\n")

	if len(memberOverrides) > 0 {
		b.WriteString("\n  member_overrides = {\n")

		for _, member := range slices.Sorted(maps.Keys(memberOverrides)) {
			overrideConfig := memberOverrides[member]

			fmt.Fprintf(&b, "    %q = {\n", member)
			b.WriteString("      config = {\n")

			for _, key := range slices.Sorted(maps.Keys(overrideConfig)) {
				fmt.Fprintf(&b, "        %q = %q\n", key, overrideConfig[key])
			}

			b.WriteString("      }\n")
			b.WriteString("    }\n")
		}

		b.WriteString("  }\n")
	}

	b.WriteString("}\n")

	return b.String()
}
