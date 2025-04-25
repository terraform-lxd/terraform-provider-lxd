package device

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

type DeviceDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	InstanceName types.String `tfsdk:"instance_name"`
	Project      types.String `tfsdk:"project"`
	Remote       types.String `tfsdk:"remote"`

	// Computed.
	Type       types.String `tfsdk:"type"`
	Properties types.Map    `tfsdk:"properties"`
}

type DeviceDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

func (d *DeviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *DeviceDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"instance_name": schema.StringAttribute{
				Required: true,
			},

			"project": schema.StringAttribute{
				Optional: true,
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed.

			"type": schema.StringAttribute{
				Computed: true,
			},

			"properties": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *DeviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DeviceDataSourceModel

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

	instanceName := state.InstanceName.ValueString()
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve an existing instance %q", instanceName), err.Error())
		return
	}

	deviceName := state.Name.ValueString()
	device, ok := instance.Devices[deviceName]
	if !ok || device == nil {
		msg := fmt.Sprintf("Failed to retrieve device %q for instance %q", deviceName, instanceName)
		resp.Diagnostics.AddError(msg, msg)
		return
	}

	if device["type"] == "" {
		resp.Diagnostics.AddError(
			"Device is missing type",
			fmt.Sprintf("Device %q for instance %q is missing type field", deviceName, instanceName),
		)
		return
	}

	deviceType := types.StringValue(device["type"])
	delete(device, "type")

	deviceProps, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(device), state.Properties)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Type = deviceType
	state.Properties = deviceProps

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
