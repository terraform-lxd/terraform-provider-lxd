package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

// NetworkLBModel resource data model that matches the schema.
type NetworkLBModel struct {
	Network       types.String `tfsdk:"network"`
	ListenAddress types.String `tfsdk:"listen_address"`
	Ports         types.Set    `tfsdk:"port"`
	Backends      types.Set    `tfsdk:"backend"`
	Description   types.String `tfsdk:"description"`
	Project       types.String `tfsdk:"project"`
	Remote        types.String `tfsdk:"remote"`
	Config        types.Map    `tfsdk:"config"`
}

// IncusNetworkLBResource represent Incus network load balancer resource.
type IncusNetworkLBResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewNetworkLBResource returns a new network load balancer resource.
func NewNetworkLBResource() resource.Resource {
	return &IncusNetworkLBResource{}
}

func (r IncusNetworkLBResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_lb", req.ProviderTypeName)
}

func (r IncusNetworkLBResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		},

		Blocks: map[string]schema.Block{
			"backend": schema.SetNestedBlock{
				Description: "Network load balancer backend",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "LB backend name",
						},

						"description": schema.StringAttribute{
							Optional:    true,
							Description: "LB backend description",
						},

						"target_address": schema.StringAttribute{
							Required:    true,
							Description: "LB backend target address",
						},

						"target_port": schema.StringAttribute{
							Optional:    true,
							Description: "LB backend target port",
						},
					},
				},
			},

			"port": schema.SetNestedBlock{
				Description: "Network load balancer port",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"description": schema.StringAttribute{
							Optional:    true,
							Description: "Port description",
						},

						"protocol": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("tcp"),
							Description: "Port protocol",
							Validators: []validator.String{
								stringvalidator.OneOf("tcp", "udp"),
							},
						},

						"listen_port": schema.StringAttribute{
							Required:    true,
							Description: "Port to listen to",
						},

						"target_backend": schema.SetAttribute{
							Required:    true,
							Description: "List of target LB backends",
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

func (r *IncusNetworkLBResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.IncusProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	r.provider = provider
}

func (r IncusNetworkLBResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkLBModel

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

	backends, diag := ToLBBackendList(ctx, plan.Backends)
	resp.Diagnostics.Append(diag...)

	ports, diag := ToLBPortList(ctx, plan.Ports)
	resp.Diagnostics.Append(diag...)

	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)

	if resp.Diagnostics.HasError() {
		return
	}

	networkName := plan.Network.ValueString()
	listenAddr := plan.ListenAddress.ValueString()
	lbName := toLBName(networkName, listenAddr)

	lbReq := api.NetworkLoadBalancersPost{
		ListenAddress: listenAddr,
		NetworkLoadBalancerPut: api.NetworkLoadBalancerPut{
			Description: plan.Description.ValueString(),
			Ports:       ports,
			Backends:    backends,
			Config:      config,
		},
	}

	// Create LB.
	err = server.CreateNetworkLoadBalancer(networkName, lbReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network load balancer %q", lbName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r IncusNetworkLBResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkLBModel

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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r IncusNetworkLBResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkLBModel

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

	backends, diag := ToLBBackendList(ctx, plan.Backends)
	resp.Diagnostics.Append(diag...)

	ports, diag := ToLBPortList(ctx, plan.Ports)
	resp.Diagnostics.Append(diag...)

	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)

	if resp.Diagnostics.HasError() {
		return
	}

	networkName := plan.Network.ValueString()
	listenAddr := plan.ListenAddress.ValueString()
	lbName := toLBName(networkName, listenAddr)

	lbReq := api.NetworkLoadBalancerPut{
		Description: plan.Description.ValueString(),
		Backends:    backends,
		Ports:       ports,
		Config:      config,
	}

	// Update network LB.
	_, etag, err := server.GetNetworkLoadBalancer(networkName, listenAddr)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network load balancer %q", lbName), err.Error())
		return
	}

	err = server.UpdateNetworkLoadBalancer(networkName, listenAddr, lbReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network load balancer %q", lbName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r IncusNetworkLBResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkLBModel

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
	listenAddr := state.ListenAddress.ValueString()
	lbName := toLBName(networkName, listenAddr)

	err = server.DeleteNetworkLoadBalancer(networkName, listenAddr)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network load balancer %q", lbName), err.Error())
	}
}

// SyncState fetches the server's current state for an network load balancer
// and updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r IncusNetworkLBResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m NetworkLBModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	networkName := m.Network.ValueString()
	listenAddr := m.ListenAddress.ValueString()
	lb, _, err := server.GetNetworkLoadBalancer(networkName, listenAddr)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		lbName := toLBName(networkName, listenAddr)
		respDiags.AddError(fmt.Sprintf("Failed to retrieve network load balancer %q", lbName), err.Error())
		return respDiags
	}

	backends, diags := ToLBBackendSetType(ctx, lb.Backends)
	respDiags.Append(diags...)

	ports, diags := ToLBPortSetType(ctx, lb.Ports)
	respDiags.Append(diags...)

	config, diags := common.ToConfigMapType(ctx, lb.Config)
	respDiags.Append(diags...)

	m.Description = types.StringValue(lb.Description)
	m.Backends = backends
	m.Ports = ports
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

type IncusNetworkLBBackendModel struct {
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	TargetAddress types.String `tfsdk:"target_address"`
	TargetPort    types.String `tfsdk:"target_port"`
}

// ToLBBackendList converts network LB backend from types.Set into
// list of API backends.
func ToLBBackendList(ctx context.Context, backendsSet types.Set) ([]api.NetworkLoadBalancerBackend, diag.Diagnostics) {
	if backendsSet.IsNull() || backendsSet.IsUnknown() {
		return []api.NetworkLoadBalancerBackend{}, nil
	}

	modelBackends := make([]IncusNetworkLBBackendModel, 0, len(backendsSet.Elements()))
	diags := backendsSet.ElementsAs(ctx, &modelBackends, false)
	if diags.HasError() {
		return nil, diags
	}

	// Convert into API network LB backends.
	backends := make([]api.NetworkLoadBalancerBackend, 0, len(modelBackends))
	for _, b := range modelBackends {
		backend := api.NetworkLoadBalancerBackend{
			Name:          b.Name.ValueString(),
			Description:   b.Description.ValueString(),
			TargetAddress: b.TargetAddress.ValueString(),
			TargetPort:    b.TargetPort.ValueString(),
		}

		backends = append(backends, backend)
	}

	return backends, nil
}

// ToLBBackendList converts list of API network LB backends into types.Set.
func ToLBBackendSetType(ctx context.Context, backends []api.NetworkLoadBalancerBackend) (types.Set, diag.Diagnostics) {
	backendList := make([]IncusNetworkLBBackendModel, 0, len(backends))
	for _, b := range backends {
		backend := IncusNetworkLBBackendModel{
			Name:          types.StringValue(b.Name),
			Description:   types.StringValue(b.Description),
			TargetAddress: types.StringValue(b.TargetAddress),
			TargetPort:    types.StringValue(b.TargetPort),
		}

		backendList = append(backendList, backend)
	}

	backendType := map[string]attr.Type{
		"name":           types.StringType,
		"description":    types.StringType,
		"target_address": types.StringType,
		"target_port":    types.StringType,
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: backendType}, backendList)
}

type NetworkLBPortModel struct {
	Description   types.String `tfsdk:"description"`
	Protocol      types.String `tfsdk:"protocol"`
	ListenPort    types.String `tfsdk:"listen_port"`
	TargetBackend types.Set    `tfsdk:"target_backend"`
}

// ToLBPortList converts network LB backend from types.Set into
// list of API ports.
func ToLBPortList(ctx context.Context, portSet types.Set) ([]api.NetworkLoadBalancerPort, diag.Diagnostics) {
	if portSet.IsNull() || portSet.IsUnknown() {
		return []api.NetworkLoadBalancerPort{}, nil
	}

	portList := make([]NetworkLBPortModel, 0, len(portSet.Elements()))
	diags := portSet.ElementsAs(ctx, &portList, false)
	if diags.HasError() {
		return nil, diags
	}

	// Convert into API network LB ports.
	ports := make([]api.NetworkLoadBalancerPort, 0, len(portList))
	for _, p := range portList {
		// Convert target backends string slice.
		backends := make([]string, 0, len(p.TargetBackend.Elements()))
		if !p.TargetBackend.IsNull() && !p.TargetBackend.IsUnknown() {
			diags := p.TargetBackend.ElementsAs(ctx, &backends, false)
			if diags.HasError() {
				return nil, diags
			}
		}

		port := api.NetworkLoadBalancerPort{
			Description:   p.Description.ValueString(),
			Protocol:      p.Protocol.ValueString(),
			ListenPort:    p.ListenPort.ValueString(),
			TargetBackend: backends,
		}

		ports = append(ports, port)
	}

	return ports, nil
}

// ToLBPortList converts list of API network LB ports into types.Set.
func ToLBPortSetType(ctx context.Context, ports []api.NetworkLoadBalancerPort) (types.Set, diag.Diagnostics) {
	portType := map[string]attr.Type{
		"description":    types.StringType,
		"protocol":       types.StringType,
		"listen_port":    types.StringType,
		"target_backend": types.SetType{ElemType: types.StringType},
	}

	portList := make([]NetworkLBPortModel, 0, len(ports))
	for _, p := range ports {
		backends, diags := types.SetValueFrom(ctx, types.StringType, p.TargetBackend)
		if diags.HasError() {
			return types.SetNull(types.ObjectType{AttrTypes: portType}), diags
		}

		port := NetworkLBPortModel{
			Description:   types.StringValue(p.Description),
			Protocol:      types.StringValue(p.Protocol),
			ListenPort:    types.StringValue(p.ListenPort),
			TargetBackend: backends,
		}

		portList = append(portList, port)
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: portType}, portList)
}

// toLBName creates a unique load balancer name (id).
func toLBName(networkName string, listenAddr string) string {
	return fmt.Sprintf("%s/%s", networkName, listenAddr)
}
