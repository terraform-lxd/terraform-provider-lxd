package storage

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

type StoragePoolDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	Project types.String `tfsdk:"project"`
	Remote  types.String `tfsdk:"remote"`

	// Computed.
	Description types.String `tfsdk:"description"`
	Driver      types.String `tfsdk:"driver"`
	Status      types.String `tfsdk:"status"`
	Config      types.Map    `tfsdk:"config"`
	Locations   types.Set    `tfsdk:"locations"`
}

func NewStoragePoolDataSource() datasource.DataSource {
	return &StoragePoolDataSource{}
}

type StoragePoolDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func (d *StoragePoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_pool", req.ProviderTypeName)
}

func (d *StoragePoolDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

			"driver": schema.StringAttribute{
				Computed: true,
			},

			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"status": schema.StringAttribute{
				Computed: true,
			},

			"locations": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *StoragePoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *StoragePoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state StoragePoolDataSourceModel

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

	poolName := state.Name.ValueString()
	pool, _, err := server.GetStoragePool(poolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage pool %q", poolName), err.Error())
		return
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(pool.Config), state.Config)
	resp.Diagnostics.Append(diags...)

	locations, diags := types.SetValueFrom(ctx, types.StringType, pool.Locations)
	resp.Diagnostics.Append(diags...)

	state.Name = types.StringValue(pool.Name)
	state.Description = types.StringValue(pool.Description)
	state.Driver = types.StringValue(pool.Driver)
	state.Status = types.StringValue(pool.Status)
	state.Config = config
	state.Locations = locations

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
