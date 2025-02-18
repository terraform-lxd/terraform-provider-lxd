package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

type InstanceDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	Project types.String `tfsdk:"project"`
	Remote  types.String `tfsdk:"remote"`

	// Computed
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	IPv4        types.String `tfsdk:"ipv4_address"`
	IPv6        types.String `tfsdk:"ipv6_address"`
	MAC         types.String `tfsdk:"mac_address"`
	Location    types.String `tfsdk:"location"`
	Status      types.String `tfsdk:"status"`
	Ephemeral   types.Bool   `tfsdk:"ephemeral"`
	Running     types.Bool   `tfsdk:"running"`
	Profiles    types.List   `tfsdk:"profiles"`
	Devices     types.Map    `tfsdk:"devices"`
	Limits      types.Map    `tfsdk:"limits"`
	Config      types.Map    `tfsdk:"config"`
	Interfaces  types.Map    `tfsdk:"interfaces"`
}

type InstanceDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func NewInstanceDataSource() datasource.DataSource {
	return &InstanceDataSource{}
}

func (d *InstanceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_instance", req.ProviderTypeName)
}

func (d *InstanceDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"project": schema.StringAttribute{
				Optional: true,
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed.

			"description": schema.StringAttribute{
				Computed: true,
			},

			"type": schema.StringAttribute{
				Computed: true,
			},

			"ephemeral": schema.BoolAttribute{
				Computed: true,
			},

			"running": schema.BoolAttribute{
				Computed: true,
			},

			"profiles": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"limits": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"devices": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed: true,
						},

						"properties": schema.MapAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},

			"interfaces": schema.MapNestedAttribute{
				Computed:    true,
				Description: "Map of the instance network interfaces",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed: true,
						},

						"type": schema.StringAttribute{
							Computed: true,
						},

						"state": schema.StringAttribute{
							Computed: true,
						},

						"ips": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"address": schema.StringAttribute{
										Computed: true,
									},

									"family": schema.StringAttribute{
										Computed: true,
									},

									"scope": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},

			"ipv4_address": schema.StringAttribute{
				Computed: true,
			},

			"ipv6_address": schema.StringAttribute{
				Computed: true,
			},

			"mac_address": schema.StringAttribute{
				Computed: true,
			},

			"location": schema.StringAttribute{
				Computed: true,
			},

			"status": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *InstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.LxdProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	d.provider = provider
}

func (d *InstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state InstanceDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var respDiags diag.Diagnostics

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := d.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := state.Name.ValueString()
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve an existing instance %q", instanceName), err.Error())
		return
	}

	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return
	}

	// Set null values for IPv4, IPv6, and MAC addresses to ensure
	// the computed value is always set.
	state.IPv4 = types.StringNull()
	state.IPv6 = types.StringNull()
	state.MAC = types.StringNull()

	accIface, ok := instance.ExpandedConfig["user.access_interface"]
	if ok {
		// If there is an user.access_interface set, extract IPv4, IPv6 and
		// MAC addresses from that network interface.
		net, ok := instanceState.Network[accIface]
		if ok {
			state.MAC = types.StringValue(net.Hwaddr)
			ipv4, ipv6 := findGlobalIPAddresses(net)

			if ipv4 != "" {
				state.IPv4 = types.StringValue(ipv4)
			}

			if ipv6 != "" {
				state.IPv6 = types.StringValue(ipv6)
			}
		}
	} else {
		// Search for the first interface (alphabetically sorted) that has
		// global IPv4 or IPv6 address.
		for _, iface := range utils.SortMapKeys(instanceState.Network) {
			if iface == "lo" {
				continue
			}

			net := instanceState.Network[iface]
			ipv4, ipv6 := findGlobalIPAddresses(net)
			if ipv4 != "" || ipv6 != "" {
				state.MAC = types.StringValue(net.Hwaddr)

				if ipv4 != "" {
					state.IPv4 = types.StringValue(ipv4)
				}

				if ipv6 != "" {
					state.IPv6 = types.StringValue(ipv6)
				}

				break
			}
		}
	}

	// Extract limits (the rest is simply config).
	instanceLimits := make(map[string]string)
	instanceConfig := make(map[string]string)
	for k, v := range instance.Config {
		key, ok := strings.CutPrefix(k, "limits.")
		if ok {
			instanceLimits[key] = v
			continue
		}

		instanceConfig[k] = v
	}

	// Convert config, limits, profiles, and devices into schema type.
	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(instanceConfig), state.Config)
	resp.Diagnostics.Append(diags...)

	limits, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(instanceLimits), state.Limits)
	resp.Diagnostics.Append(diags...)

	profiles, diags := ToProfileListType(ctx, instance.Profiles)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMapType(ctx, instance.Devices)
	resp.Diagnostics.Append(diags...)

	interfaces, diags := common.ToInterfaceMapType(ctx, instanceState.Network, instance.Config)
	resp.Diagnostics.Append(diags...)

	if respDiags.HasError() {
		return
	}

	state.Name = types.StringValue(instance.Name)
	state.Type = types.StringValue(instance.Type)
	state.Description = types.StringValue(instance.Description)
	state.Ephemeral = types.BoolValue(instance.Ephemeral)
	state.Running = types.BoolValue(instanceState.Status == api.Running.String())
	state.Location = types.StringValue(instance.Location)
	state.Status = types.StringValue(instance.Status)
	state.Profiles = profiles
	state.Limits = limits
	state.Devices = devices
	state.Interfaces = interfaces
	state.Config = config

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
