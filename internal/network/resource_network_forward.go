package network

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkForwardModel resource data model that matches the schema.
type NetworkForwardModel struct {
	Network       types.String `tfsdk:"network"`
	ListenAddress types.String `tfsdk:"listen_address"`
	Ports         types.Set    `tfsdk:"ports"`
	Description   types.String `tfsdk:"description"`
	Project       types.String `tfsdk:"project"`
	Remote        types.String `tfsdk:"remote"`
	Config        types.Map    `tfsdk:"config"`
}

// NetworkForwardModel resource data model that matches the schema.
type NetworkForwardPortModel struct {
	Description   types.String `tfsdk:"description"`
	Protocol      types.String `tfsdk:"protocol"`
	ListenPort    types.String `tfsdk:"listen_port"`
	TargetPort    types.String `tfsdk:"target_port"`
	TargetAddress types.String `tfsdk:"target_address"`
}

// NetworkForwardResource represent network forward resource.
type NetworkForwardResource struct {
	provider *provider_config.LxdProviderConfig
}

func NewNetworkForwardResource() resource.Resource {
	return &NetworkForwardResource{}
}

func (r *NetworkForwardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_forward", req.ProviderTypeName)
}

func (r *NetworkForwardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	portObjectType := portObjectType()

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"network": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"listen_address": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"project": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},

			"ports": schema.SetNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  setdefault.StaticValue(types.SetNull(portObjectType)),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Port description",
						},

						"protocol": schema.StringAttribute{
							Required:    true,
							Description: "Port protocol",
							Validators: []validator.String{
								stringvalidator.OneOf("tcp", "udp"),
							},
						},

						"listen_port": schema.StringAttribute{
							Required:    true,
							Description: "Listen port to forward",
						},

						"target_port": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Target port to forward listen port to. Defaults to the value of listen_port",
						},

						"target_address": schema.StringAttribute{
							Required:    true,
							Description: "Target address to forward listen port to",
						},
					},
				},
			},
		},
	}
}

func portObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"description":    types.StringType,
			"protocol":       types.StringType,
			"listen_port":    types.StringType,
			"target_port":    types.StringType,
			"target_address": types.StringType,
		},
	}
}

func (r *NetworkForwardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.LxdProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	r.provider = provider
}

func (r *NetworkForwardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkForwardModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	ports, diags := ToNetworkForwardPortList(ctx, plan.Ports)
	resp.Diagnostics.Append(diags...)

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	networkName := plan.Network.ValueString()
	listenAddress := plan.ListenAddress.ValueString()

	createRequest := api.NetworkForwardsPost{
		ListenAddress: listenAddress,
		NetworkForwardPut: api.NetworkForwardPut{
			Description: plan.Description.ValueString(),
			Ports:       ports,
			Config:      config,
		},
	}

	err = server.CreateNetworkForward(networkName, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network forward for %q", listenAddress), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkForwardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkForwardModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkForwardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkForwardModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	ports, diags := ToNetworkForwardPortList(ctx, plan.Ports)
	resp.Diagnostics.Append(diags...)

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	networkName := plan.Network.ValueString()
	listenAddress := plan.ListenAddress.ValueString()

	updateRequest := api.NetworkForwardPut{
		Description: plan.Description.ValueString(),
		Ports:       ports,
		Config:      config,
	}

	_, etag, err := server.GetNetworkForward(networkName, listenAddress)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve network forward for %q", listenAddress), err.Error())
	}

	err = server.UpdateNetworkForward(networkName, listenAddress, updateRequest, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network forward for %q", listenAddress), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkForwardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkForwardModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkName := state.Network.ValueString()
	listenAddress := state.ListenAddress.ValueString()

	err = server.DeleteNetworkForward(networkName, listenAddress)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete network forward for %q", listenAddress), err.Error())
	}
}

func (r *NetworkForwardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "network_forward",
		RequiredFields: []string{"network", "listen_address"},
	}

	fields, diags := meta.ParseImportID(req.ID)
	if diags != nil {
		resp.Diagnostics.Append(diags)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

func (r *NetworkForwardResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m NetworkForwardModel) diag.Diagnostics {
	networkName := m.Network.ValueString()
	listenAddress := m.ListenAddress.ValueString()
	networkForward, _, err := server.GetNetworkForward(networkName, listenAddress)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		return diag.Diagnostics{diag.NewErrorDiagnostic(
			fmt.Sprintf("Failed to retrieve network forward %q", listenAddress), err.Error(),
		)}
	}

	ports, diags := ToNetworkForwardPortSetType(ctx, networkForward.Ports)
	if diags.HasError() {
		return diags
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(networkForward.Config), m.Config)
	if diags.HasError() {
		return diags
	}

	m.Description = types.StringValue(networkForward.Description)
	m.Ports = ports
	m.Config = config

	return tfState.Set(ctx, &m)
}

// ToNetworkForwardPortList converts forward ports from type types.Set into []api.NetworkForwardPort.
func ToNetworkForwardPortList(ctx context.Context, portsSet types.Set) ([]api.NetworkForwardPort, diag.Diagnostics) {
	if portsSet.IsNull() || portsSet.IsUnknown() {
		return []api.NetworkForwardPort{}, nil
	}

	modelPorts := make([]NetworkForwardPortModel, 0, len(portsSet.Elements()))
	diags := portsSet.ElementsAs(ctx, &modelPorts, false)
	if diags.HasError() {
		return nil, diags
	}

	ports := make([]api.NetworkForwardPort, 0, len(modelPorts))
	for _, modelPort := range modelPorts {
		port := api.NetworkForwardPort{
			Description:   modelPort.Description.ValueString(),
			Protocol:      modelPort.Protocol.ValueString(),
			ListenPort:    modelPort.ListenPort.ValueString(),
			TargetPort:    modelPort.TargetPort.ValueString(),
			TargetAddress: modelPort.TargetAddress.ValueString(),
		}

		ports = append(ports, port)
	}

	return ports, nil
}

// ToNetworkForwardPortSetType converts []api.NetworkForwardPort into forward ports of type types.Set.
func ToNetworkForwardPortSetType(ctx context.Context, ports []api.NetworkForwardPort) (types.Set, diag.Diagnostics) {
	portObjectType := portObjectType()
	nilSet := types.SetNull(portObjectType)

	if len(ports) == 0 {
		return nilSet, nil
	}

	portList := make([]attr.Value, 0, len(ports))
	for _, port := range ports {
		portMap := map[string]attr.Value{
			"description":    types.StringValue(port.Description),
			"protocol":       types.StringValue(port.Protocol),
			"listen_port":    types.StringValue(port.ListenPort),
			"target_port":    types.StringValue(port.TargetPort),
			"target_address": types.StringValue(port.TargetAddress),
		}

		portObject, diags := types.ObjectValue(portObjectType.AttrTypes, portMap)
		if diags.HasError() {
			return nilSet, diags
		}

		portList = append(portList, portObject)
	}

	return types.SetValue(portObjectType, portList)
}
