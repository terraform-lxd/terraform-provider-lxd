package cluster

import (
	"context"
	"fmt"

	"github.com/canonical/lxd/shared"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// ClusterLinkDataSourceModel is a data source data model that matches the schema.
type ClusterLinkDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Remote      types.String `tfsdk:"remote"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Config      types.Map    `tfsdk:"config"`
	Addresses   types.List   `tfsdk:"addresses"`
}

// ClusterLinkDataSource reads LXD cluster links.
type ClusterLinkDataSource struct {
	provider *provider_config.LxdProviderConfig
}

// NewClusterLinkDataSource returns a new cluster link data source.
func NewClusterLinkDataSource() datasource.DataSource {
	return &ClusterLinkDataSource{}
}

func (d *ClusterLinkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_link"
}

func (d *ClusterLinkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
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

			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"addresses": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *ClusterLinkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterLinkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ClusterLinkDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := d.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	linkName := state.Name.ValueString()

	link, _, err := server.GetClusterLink(linkName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster link %q", linkName), err.Error())
		return
	}

	// Volatile keys are managed by LXD and are not part of the configuration.
	config, diags := common.ToConfigMapType(ctx, common.StripConfig(link.Config, state.Config, []string{"volatile."}), state.Config)
	resp.Diagnostics.Append(diags...)

	// The list is empty rather than null while the link is pending, so that its length tells
	// whether the link is active.
	addresses := shared.SplitNTrimSpace(link.Config["volatile.addresses"], ",", -1, true)
	if addresses == nil {
		addresses = []string{}
	}

	addressesList, diags := types.ListValueFrom(ctx, types.StringType, addresses)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	state.Name = types.StringValue(link.Name)
	state.Description = types.StringValue(link.Description)
	state.Type = types.StringValue(link.Type)
	state.Config = config
	state.Addresses = addressesList

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
