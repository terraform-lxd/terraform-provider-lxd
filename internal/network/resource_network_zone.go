package network

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkZoneModel resource data model that matches the schema.
type NetworkZoneModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
}

// NetworkZoneResource represent LXD network zone resource.
type NetworkZoneResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewNetworkZoneResource returns a new network zone resource.
func NewNetworkZoneResource() resource.Resource {
	return &NetworkZoneResource{}
}

func (r NetworkZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_zone", req.ProviderTypeName)
}

func (r NetworkZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
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
	}
}

func (r *NetworkZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r NetworkZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkZoneModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(plan.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := plan.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Convert network zone config to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := plan.Name.ValueString()
	zoneReq := api.NetworkZonesPost{
		Name: zoneName,
		NetworkZonePut: api.NetworkZonePut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateNetworkZone(zoneReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network zone %q", zoneName), err.Error())
		return
	}

	_, diags = plan.Sync(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkZoneModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(state.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := state.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	found, diags := state.Sync(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Remove resource state if resource is not found.
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkZoneModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(plan.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := plan.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	zoneName := plan.Name.ValueString()
	_, etag, err := server.GetNetworkZone(zoneName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network zone %q", zoneName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update network zone.
	zoneReq := api.NetworkZonePut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateNetworkZone(zoneName, zoneReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network zone %q", zoneName), err.Error())
		return
	}

	_, diags = plan.Sync(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkZoneModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(state.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := state.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	zoneName := state.Name.ValueString()
	err = server.DeleteNetworkZone(zoneName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network zone %q", zoneName), err.Error())
	}
}

func (r NetworkZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	remote, project, name, diag := common.SplitImportID(req.ID, "network_zone")
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	if remote != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	}

	if project != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

// Sync pulls network zone data from the server and updates the model in-place.
// It returns a boolean indicating whether resource is found and diagnostics
// that contain potential errors.
// This should be called before updating Terraform state.
func (m *NetworkZoneModel) Sync(ctx context.Context, server lxd.InstanceServer) (bool, diag.Diagnostics) {
	zoneName := m.Name.ValueString()
	zone, _, err := server.GetNetworkZone(zoneName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		return true, diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve network zone %q", zoneName), err.Error()),
		}
	}

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, zone.Config)
	if diags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(zone.Name)
	m.Description = types.StringValue(zone.Description)
	m.Config = config

	return true, nil
}
