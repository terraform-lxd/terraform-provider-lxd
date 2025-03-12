package server

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type InfoModel struct {
	Remote        types.String `tfsdk:"remote"`
	APIExtensions types.List   `tfsdk:"api_extensions"`
	Members       types.Map    `tfsdk:"cluster_members"`
	InstanceTypes types.List   `tfsdk:"instance_types"`
}

type ClusterMemberModel struct {
	URL    types.String `tfsdk:"url"`
	Status types.String `tfsdk:"status"`
}

func NewInfoDataSource() datasource.DataSource {
	return &InfoDataSource{}
}

type InfoDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func (d *InfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_info", req.ProviderTypeName)
}

func (d *InfoDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"remote": schema.StringAttribute{
				Optional: true,
			},

			"api_extensions": schema.ListAttribute{
				Description: "List of API extensions supported by the LXD server",
				Computed:    true,
				ElementType: types.StringType,
			},

			"cluster_members": schema.MapNestedAttribute{
				Computed:    true,
				Description: "Map of cluster members, which is empty if LXD is not clustered. The map key represents a cluster member name.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							Description: "Cluster member URL",
							Computed:    true,
						},

						"status": schema.StringAttribute{
							Description: "Cluster member status",
							Computed:    true,
						},
					},
				},
			},

			"instance_types": schema.ListAttribute{
				Description: "List of supported instance types",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *InfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var diags diag.Diagnostics
	var config InfoModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := d.provider.SelectRemote(config.Remote.ValueString())
	server, err := d.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve the API server from remote %q", remote), err.Error())
		return
	}

	var clusterMembers map[string]ClusterMemberModel

	if server.IsClustered() {
		members, err := server.GetClusterMembers()
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster members from remote %q", remote), err.Error())
			return
		}

		clusterMembers = make(map[string]ClusterMemberModel, len(members))
		for _, member := range members {
			clusterMembers[member.ServerName] = ClusterMemberModel{
				URL:    types.StringValue(member.URL),
				Status: types.StringValue(member.Status),
			}
		}
	}

	config.Remote = types.StringValue(remote)

	config.APIExtensions, diags = types.ListValueFrom(ctx, types.StringType, apiServer.APIExtensions)
	resp.Diagnostics.Append(diags...)

	config.Members, diags = ToClusterMemberMapType(ctx, clusterMembers)
	resp.Diagnostics.Append(diags...)

	// Sort instance types to ensure they are always in the same order.
	instanceTypes := apiServer.Environment.InstanceTypes
	slices.Sort(instanceTypes)

	config.InstanceTypes, diags = types.ListValueFrom(ctx, types.StringType, instanceTypes)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

// ToClusterMemberMapType converts map[string]ClusterMemberModel into types.Map.
func ToClusterMemberMapType(ctx context.Context, members map[string]ClusterMemberModel) (types.Map, diag.Diagnostics) {
	objType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"url":    types.StringType,
			"status": types.StringType,
		},
	}

	if members == nil {
		members = map[string]ClusterMemberModel{}
	}

	return types.MapValueFrom(ctx, objType, members)
}
