package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkDataSourceModel resource data model that matches the schema.
type NetworkDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	Project types.String `tfsdk:"project"`
	Remote  types.String `tfsdk:"remote"`

	// Computed.
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Managed     types.Bool   `tfsdk:"managed"`
	Config      types.Map    `tfsdk:"config"`
	IPv4        types.String `tfsdk:"ipv4_address"`
	IPv6        types.String `tfsdk:"ipv6_address"`
}

func NewNetworkDataSource() datasource.DataSource {
	return &NetworkDataSource{}
}

type NetworkDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func (d *NetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network", req.ProviderTypeName)
}

func (d *NetworkDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

			"description": schema.StringAttribute{
				Computed: true,
			},

			"type": schema.StringAttribute{
				Computed: true,
			},

			"managed": schema.BoolAttribute{
				Computed: true,
			},

			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"ipv4_address": schema.StringAttribute{
				Computed: true,
			},

			"ipv6_address": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *NetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state NetworkDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := d.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkName := state.Name.ValueString()
	network, _, err := server.GetNetwork(networkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network %q", networkName), err.Error())
		return
	}

	networkState, err := server.GetNetworkState(networkName)
	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of network %q", networkName), err.Error())
		return
	}

	var ipv4, ipv6 string
	if networkState != nil {
		ipv4, ipv6 = findGlobalCIDRs(networkState.Addresses)
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(network.Config), state.Config)
	resp.Diagnostics.Append(diags...)

	state.Name = types.StringValue(network.Name)
	state.Description = types.StringValue(network.Description)
	state.Type = types.StringValue(network.Type)
	state.Managed = types.BoolValue(network.Managed)
	state.Config = config

	state.IPv4 = types.StringValue(ipv4)
	state.IPv6 = types.StringValue(ipv6)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
