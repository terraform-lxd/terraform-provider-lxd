package network_test

import (
	"fmt"
	"maps"
	"net"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/acctest"
)

// isCIDR is a custom check that verifies the given value represents a valid
// CIDR notation IP address.
func isCIDR(value string) error {
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		return fmt.Errorf("Value %q is not a valid CIDR: %s", value, err)
	}

	return nil
}

func TestAccNetwork_basic(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_basic(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "description", ""),
					resource.TestCheckResourceAttr("lxd_network.network", "config.%", "0"),
					resource.TestCheckResourceAttr("lxd_network.network", "managed", "true"),
					resource.TestCheckResourceAttrWith("lxd_network.network", "ipv4_address", isCIDR),
					resource.TestCheckResourceAttrWith("lxd_network.network", "ipv6_address", isCIDR),
				),
			},
		},
	})
}

func TestAccNetwork_description(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_desc(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "description", "My network"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.%", "2"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.address", "10.150.10.1/24"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv6.address", "fd42:474b:622d:259d::1/64"),
				),
			},
		},
	})
}

func TestAccNetwork_nullable(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_nullable(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.%", "2"),
					resource.TestCheckNoResourceAttr("lxd_network.network", "config.ipv4.address"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv6.address", "none"),
				),
			},
		},
	})
}

func TestAccNetwork_attach(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")
	profileName := acctest.GenerateName(2, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_attach(networkName, profileName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "name", profileName),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.name", "eth1"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.type", "nic"),
					resource.TestCheckResourceAttr("lxd_profile.profile1", "device.0.properties.parent", networkName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.#", "2"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.0", "default"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "profiles.1", profileName),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv6_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "mac_address"),
				),
			},
		},
	})
}

func TestAccNetwork_updateConfig(t *testing.T) {
	networkName := acctest.GenerateName(1, "-")
	instanceName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_updateConfig_1(networkName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.address", "10.150.30.1/24"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.nat", "true"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.parent", networkName),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "ipv4_address"),
					resource.TestCheckResourceAttrSet("lxd_instance.instance1", "mac_address"),
				),
			},
			{
				Config: acctest.Provider() + testAccNetwork_updateConfig_2(networkName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.address", "10.150.40.1/24"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.ipv4.nat", "false"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "status", "Running"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("lxd_instance.instance1", "device.0.properties.parent", networkName),
				),
			},
		},
	})
}

func TestAccNetwork_typeMacvlan(t *testing.T) {
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_typeMacvlan(networkName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "macvlan"),
					resource.TestCheckResourceAttr("lxd_network.network", "config.parent", "lxdbr0"),
					resource.TestCheckResourceAttr("lxd_network.network", "ipv4_address", ""),
					resource.TestCheckResourceAttr("lxd_network.network", "ipv6_address", ""),
				),
			},
		},
	})
}

func TestAccNetwork_clusterConfigNoMemberOverrides(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, "bridge", nil, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", "bridge"),
					resource.TestCheckResourceAttr("lxd_network.network", "members.%", fmt.Sprintf("%d", len(targets))),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", targets[1]), "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, "bridge", nil, nil),
			},
		},
	})
}

func TestAccNetwork_clusterConfigEmptyMemberOverrides(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	networkName := acctest.GenerateName(2, "-")

	step1Overrides := map[string]map[string]string{
		targets[0]: {},
	}

	step2Overrides := map[string]map[string]string{}
	for _, target := range targets[1:] {
		step2Overrides[target] = map[string]string{}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, "bridge", nil, step1Overrides),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "members.%", fmt.Sprintf("%d", len(targets))),
					resource.TestCheckResourceAttr("lxd_network.network", "member_overrides.%", "1"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", targets[1]), "0"),
				),
			},
			{
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, "bridge", nil, step2Overrides),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "members.%", fmt.Sprintf("%d", len(targets))),
					resource.TestCheckResourceAttr("lxd_network.network", "member_overrides.%", fmt.Sprintf("%d", len(targets)-1)),
					resource.TestCheckNoResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s", targets[0])),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", targets[0]), "0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", targets[1]), "0"),
				),
			},
			{
				// Reapply the same config to ensure there is no drift in terraform state.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, "bridge", nil, step2Overrides),
			},
		},
	})
}

func TestAccNetwork_clusterConfigLifecycle(t *testing.T) {
	targets := acctest.PreCheckClustering(t, 2)
	networkName := acctest.GenerateName(2, "-")
	overrideTarget := targets[0]
	defaultTarget := targets[1]

	// Use a node-specific bridge config key so the test does not depend on
	// physical network interfaces being present on the cluster members.
	networkType := "bridge"
	configKey := "bridge.external_interfaces"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Set config key in global settings.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{configKey: "nosuchint0"}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "type", networkType),
					resource.TestCheckResourceAttr("lxd_network.network", "config."+configKey, "nosuchint0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "nosuchint0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "nosuchint0"),
				),
			},
			{
				// Set config key in global settings and one member override.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{configKey: "nosuchint0"}, map[string]map[string]string{overrideTarget: {configKey: "nosuchint1"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "config."+configKey, "nosuchint0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "nosuchint1"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "nosuchint1"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "nosuchint0"),
				),
			},
			{
				// Change the value of one member override.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{configKey: "nosuchint0"}, map[string]map[string]string{overrideTarget: {configKey: "nosuchint2"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "config."+configKey, "nosuchint0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "nosuchint2"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "nosuchint2"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "nosuchint0"),
				),
			},
			{
				// Change global config while keeping the member override.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{configKey: "nosuchint3"}, map[string]map[string]string{overrideTarget: {configKey: "nosuchint2"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "config."+configKey, "nosuchint3"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "nosuchint2"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "nosuchint2"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "nosuchint3"),
				),
			},
			{
				// Set different member override values for both members.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{configKey: "nosuchint3"}, map[string]map[string]string{overrideTarget: {configKey: "nosuchint1"}, defaultTarget: {configKey: "nosuchint2"}}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "config."+configKey, "nosuchint3"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s.config.%s", overrideTarget, configKey), "nosuchint1"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s.config.%s", defaultTarget, configKey), "nosuchint2"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "nosuchint1"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "nosuchint2"),
				),
			},
			{
				// Remove the config key from member overrides and use the global value.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{configKey: "nosuchint3"}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s", overrideTarget)),
					resource.TestCheckNoResourceAttr("lxd_network.network", fmt.Sprintf("member_overrides.%s", defaultTarget)),
					resource.TestCheckResourceAttr("lxd_network.network", "config."+configKey, "nosuchint3"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", overrideTarget, configKey), "nosuchint3"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%s", defaultTarget, configKey), "nosuchint3"),
				),
			},
			{
				// Remove the global config key and ensure it is no longer managed.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, map[string]string{}, nil),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("lxd_network.network", "config."+configKey),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", overrideTarget), "0"),
					resource.TestCheckResourceAttr("lxd_network.network", fmt.Sprintf("members.%s.config.%%", defaultTarget), "0"),
				),
			},
			{
				// Reapply final state to ensure no error and no drift.
				Config: acctest.Provider() + testAccNetwork_memberOverrides(networkName, networkType, nil, nil),
			},
		},
	})
}

func TestAccNetwork_project(t *testing.T) {
	projectName := acctest.GenerateName(2, "-")
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_project(networkName, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("lxd_network.network", "name", networkName),
					resource.TestCheckResourceAttr("lxd_network.network", "project", projectName),
					resource.TestCheckResourceAttr("lxd_project.project1", "name", projectName),
				),
			},
		},
	})
}

func TestAccNetwork_importBasic(t *testing.T) {
	resourceName := "lxd_network.network"
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_basic(networkName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        networkName,
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccNetwork_importDesc(t *testing.T) {
	resourceName := "lxd_network.network"
	networkName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_desc(networkName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        networkName,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    false, // State of "config" will be always empty.
				ImportState:                          true,
			},
		},
	})
}

func TestAccNetwork_importProject(t *testing.T) {
	resourceName := "lxd_network.network"
	networkName := acctest.GenerateName(2, "-")
	projectName := acctest.GenerateName(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.Provider() + testAccNetwork_project(networkName, projectName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/%s", projectName, networkName),
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccNetwork_basic(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
}
`, networkName)
}

func testAccNetwork_desc(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name        = "%s"
  description = "My network"
  config = {
    "ipv4.address" = "10.150.10.1/24"
    "ipv6.address" = "fd42:474b:622d:259d::1/64"
  }
}
`, networkName)
}

func testAccNetwork_nullable(networkName string) string {
	return fmt.Sprintf(`
locals {
  foo = "bar"
}

resource "lxd_network" "network" {
  name = "%s"

  config = {
    "ipv4.address" = local.foo == "bar" ? null : "10.0.0.1/24"
    "ipv6.address" = "none"
  }
}
`, networkName)
}

func testAccNetwork_attach(networkName string, profileName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
  config = {
    "ipv4.address" = "10.150.20.1/24"
  }
}

resource "lxd_profile" "profile1" {
  name = "%s"

  device {
    name = "eth1"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.network.name
    }
  }
}

resource "lxd_instance" "instance1" {
  name     = "%s"
  image    = "%s"
  profiles = ["default", lxd_profile.profile1.name]
}
`, networkName, profileName, instanceName, acctest.TestImage)
}

func testAccNetwork_updateConfig_1(networkName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
  config = {
    "ipv4.address" = "10.150.30.1/24"
    "ipv4.nat"     = true
  }
}

# We do need an instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "lxd_instance" "instance1" {
  name             = "%s"
  image            = "%s"

  wait_for {
    type = "ipv4"
  }

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.network.name
    }
  }
}
`, networkName, instanceName, acctest.TestImage)
}

func testAccNetwork_updateConfig_2(networkName string, instanceName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"

  config = {
    "ipv4.address" = "10.150.40.1/24"
    "ipv4.nat"     = false
  }
}

# We do need an instance here to ensure the network cannot
# be deleted, but must be updated in-place.
resource "lxd_instance" "instance1" {
  name             = "%s"
  image            = "%s"

  device {
    name = "eth0"
    type = "nic"
    properties = {
      nictype = "bridged"
      parent  = lxd_network.network.name
    }
  }
}
`, networkName, instanceName, acctest.TestImage)
}

func testAccNetwork_typeMacvlan(networkName string) string {
	return fmt.Sprintf(`
resource "lxd_network" "network" {
  name = "%s"
  type = "macvlan"

  config = {
    "parent" = "lxdbr0"
  }
}
`, networkName)
}

func testAccNetwork_memberOverrides(name string, networkType string, config map[string]string, memberOverrides map[string]map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `
resource "lxd_network" "network" {
  name = %q
  type = %q
`, name, networkType)

	// Config.
	configKeys := slices.Collect(maps.Keys(config))
	sort.Strings(configKeys)

	b.WriteString("  config = {\n")

	for _, key := range configKeys {
		fmt.Fprintf(&b, "    %q = %q\n", key, config[key])
	}

	b.WriteString("  }\n")

	// Member overrides.
	memberKeys := slices.Collect(maps.Keys(memberOverrides))
	sort.Strings(memberKeys)

	b.WriteString("  member_overrides = {\n")
	for _, member := range memberKeys {
		overrideConfig := memberOverrides[member]
		overrideKeys := slices.Collect(maps.Keys(overrideConfig))
		sort.Strings(overrideKeys)

		fmt.Fprintf(&b, "    %q = {\n", member)
		b.WriteString("      config = {\n")

		for _, key := range overrideKeys {
			fmt.Fprintf(&b, "        %q = %q\n", key, overrideConfig[key])
		}

		b.WriteString("      }\n")
		b.WriteString("    }\n")
	}

	b.WriteString("  }\n")
	b.WriteString("}\n")

	return b.String()
}

func testAccNetwork_project(networkName string, projectName string) string {
	return fmt.Sprintf(`
resource "lxd_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
}

resource "lxd_network" "network" {
  name    = "%s"
  type    = "bridge"
  project = lxd_project.project1.name
}
`, projectName, networkName)
}
