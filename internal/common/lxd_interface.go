package common

import (
	"context"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InterfaceModel represents LXD instance network interface.
type InterfaceModel struct {
	// Real name of the interface within the instance. If config interface is
	// defined as "eth0", the real interface within a container will have the
	// same name. However, VMs will most likely have enp0sX.
	RealName types.String `tfsdk:"name"`

	// State of the interface [UP, DOWN].
	State types.String `tfsdk:"state"`

	// Type of the interface [broadcast, loopback].
	Type types.String `tfsdk:"type"`

	// List of interface addresses.
	Addresses types.List `tfsdk:"ips"`
}

// IPModel is a wrapper of the interface IP address.
type IPModel struct {
	// IP address.
	Address types.String `tfsdk:"address"`

	// Family [inet, inet6].
	Family types.String `tfsdk:"family"`

	// Scope [link, local, global].
	Scope types.String `tfsdk:"scope"`
}

// ToInterfaceMapType converts provided intance networks into types.Map. The function
// also accepts instance config which is used to determine configuration interface name.
func ToInterfaceMapType(ctx context.Context, instNetworks map[string]api.InstanceStateNetwork, instConfig map[string]string) (types.Map, diag.Diagnostics) {
	ipType := map[string]attr.Type{
		"address": types.StringType,
		"family":  types.StringType,
		"scope":   types.StringType,
	}

	infType := map[string]attr.Type{
		"name":  types.StringType,
		"state": types.StringType,
		"type":  types.StringType,
		"ips":   types.ListType{ElemType: types.ObjectType{AttrTypes: ipType}},
	}

	nilMap := types.MapNull(types.ObjectType{AttrTypes: infType})
	if len(instNetworks) == 0 {
		return nilMap, nil
	}

	interfaces := make(map[string]InterfaceModel, len(instNetworks))
	for name, net := range instNetworks {
		// Find volatile entry that contains mac address of the network
		// interface. If addresses match, extract the config name of the
		// interface from the config key (volatile.<if_name>.hwaddr).
		cfgInfName := ""
		for k, v := range instConfig {
			if v == net.Hwaddr {
				cfgInfName = strings.SplitN(k, ".", 3)[1]
				break
			}
		}

		if cfgInfName == "" {
			// We did not find a matching config interface, therefore
			// do not export it.
			continue
		}

		// Interface metadata.
		inf := InterfaceModel{
			RealName: types.StringValue(name),
			State:    types.StringValue(net.State),
			Type:     types.StringValue(net.Type),
		}

		// Interface addresses.
		netAddresses := make([]IPModel, 0, len(net.Addresses))
		for _, addr := range net.Addresses {
			addrType := IPModel{
				Address: types.StringValue(addr.Address),
				Family:  types.StringValue(addr.Family),
				Scope:   types.StringValue(addr.Scope),
			}

			netAddresses = append(netAddresses, addrType)
		}

		addresses, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipType}, netAddresses)
		if diags.HasError() {
			return nilMap, diags
		}

		inf.Addresses = addresses
		interfaces[cfgInfName] = inf
	}

	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: infType}, interfaces)
}
