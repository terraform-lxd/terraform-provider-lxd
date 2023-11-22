package storage

import (
	"context"
	"fmt"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type StorageVolumeModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Pool        types.String `tfsdk:"pool"`
	Type        types.String `tfsdk:"type"`
	ContentType types.String `tfsdk:"content_type"`
	Project     types.String `tfsdk:"project"`
	Target      types.String `tfsdk:"target"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`

	// Computed.
	Location       types.String `tfsdk:"location"`
	ExpandedConfig types.Map    `tfsdk:"expanded_config"`
}

// StorageVolumeResource represent LXD storage volume resource.
type StorageVolumeResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewStorageVolumeResource returns a new storage volume resource.
func NewStorageVolumeResource() resource.Resource {
	return &StorageVolumeResource{}
}

func (r StorageVolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_volume", req.ProviderTypeName)
}

func (r StorageVolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"pool": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("custom"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"content_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("filesystem"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("filesystem", "block"),
				},
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

			"target": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},

			// Computed.

			"location": schema.StringAttribute{
				Optional: true,
			},

			"expanded_config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *StorageVolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r StorageVolumeResource) Setup(_ context.Context, data StorageVolumeModel) (lxd.InstanceServer, diag.Diagnostic) {
	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		return nil, errors.NewInstanceServerError(err)
	}

	project := data.Project.ValueString()
	target := data.Target.ValueString()

	if project != "" {
		server = server.UseProject(project)
	}

	if target != "" {
		server = server.UseTarget(target)
	}

	return server, nil
}

func (r StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageVolumeModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, plan)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	// Convert volume config to map.
	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	volName := plan.Name.ValueString()
	poolName := plan.Pool.ValueString()

	vol := api.StorageVolumesPost{
		Name:        plan.Name.ValueString(),
		Type:        plan.Type.ValueString(),
		ContentType: plan.ContentType.ValueString(),
		StorageVolumePut: api.StorageVolumePut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err := server.CreateStoragePoolVolume(poolName, vol)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage volume %q", volName), err.Error())
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

func (r StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageVolumeModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, state)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
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

func (r StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageVolumeModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, plan)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	poolName := plan.Pool.ValueString()
	volName := plan.Name.ValueString()
	volType := plan.Type.ValueString()

	vol, etag, err := server.GetStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage volume %q", volName), err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge volume config and user defined config.
	config := common.MergeConfig(vol.Config, userConfig, plan.ComputedKeys())

	volReq := api.StorageVolumePut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	// Update volume.
	err = server.UpdateStoragePoolVolume(poolName, volType, volName, volReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage volume %q", volName), err.Error())
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

func (r StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageVolumeModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, state)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	poolName := state.Pool.ValueString()
	volName := state.Name.ValueString()
	volType := state.Type.ValueString()

	err := server.DeleteStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove storage pool %q", poolName), err.Error())
	}
}

func (r StorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	remote, project, name, diag := common.SplitImportID(req.ID, "volume")
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	// Split name into pool and volume name.
	split := strings.SplitN(name, "/", 2)
	if len(split) != 2 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid import format: %q", req.ID),
			"Valid import:\nimport lxd_volume.<resource_name> [<remote>:][<project>]/<pool_name>/<volume_name>",
		)
		return
	}

	poolName := split[0]
	volName := split[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pool"), poolName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), volName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), "custom")...)

	if remote != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	}

	if project != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	}
}

// Sync pulls storage volume data from the server and updates the model
// in-place. It returns a boolean indicating whether resource is found
// and diagnostics that contain potential errors.
// This should be called before updating Terraform state.
func (m *StorageVolumeModel) Sync(ctx context.Context, server lxd.InstanceServer) (bool, diag.Diagnostics) {
	respDiags := diag.Diagnostics{}

	poolName := m.Pool.ValueString()
	volName := m.Name.ValueString()
	volType := m.Type.ValueString()

	vol, _, err := server.GetStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage volume %q", volName), err.Error())
		return true, respDiags
	}

	// Extract user defined config and merge it with current config state.
	userConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)

	stateConfig := common.StripConfig(vol.Config, userConfig, m.ComputedKeys())

	config, diags := common.ToConfigMapType(ctx, stateConfig)
	respDiags.Append(diags...)

	expandedConfig, diags := common.ToConfigMapType(ctx, vol.Config)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(vol.Name)
	m.Description = types.StringValue(vol.Description)
	m.ExpandedConfig = expandedConfig
	m.Config = config

	if vol.Location != "" && vol.Location != "none" {
		m.Location = types.StringValue(vol.Location)
	}

	return true, nil
}

// ComputedKeys returns list of computed config keys.
func (_ StorageVolumeModel) ComputedKeys() []string {
	return []string{
		"block.filesystem",
		"block.mount_options",
		"volatile.",
	}
}
