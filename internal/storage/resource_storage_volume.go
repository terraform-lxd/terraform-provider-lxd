package storage

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
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
	Location types.String `tfsdk:"location"`
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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
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
				Computed: true,
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

func (r StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageVolumeModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Convert volume config to map.
	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolName := plan.Pool.ValueString()
	volName := plan.Name.ValueString()

	vol := api.StorageVolumesPost{
		Name:        plan.Name.ValueString(),
		Type:        plan.Type.ValueString(),
		ContentType: plan.ContentType.ValueString(),
		StorageVolumePut: api.StorageVolumePut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateStoragePoolVolume(poolName, vol)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage volume %q", volName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageVolumeModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageVolumeModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageVolumeModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	poolName := state.Pool.ValueString()
	volName := state.Name.ValueString()
	volType := state.Type.ValueString()
	err = server.DeleteStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove storage pool %q", poolName), err.Error())
	}
}

func (r StorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "volume",
		RequiredFields: []string{"pool", "name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}

	// Currently, only "custom" volumes can be imported.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), "custom")...)
}

// SyncState fetches the server's current state for a storage volume and
// updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r StorageVolumeResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m StorageVolumeModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	poolName := m.Pool.ValueString()
	volName := m.Name.ValueString()
	volType := m.Type.ValueString()
	vol, _, err := server.GetStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage volume %q", volName), err.Error())
		return respDiags
	}

	// Extract user defined config and merge it with current config state.
	userConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)

	stateConfig := common.StripConfig(vol.Config, userConfig, m.ComputedKeys())

	config, diags := common.ToConfigMapType(ctx, stateConfig)
	respDiags.Append(diags...)

	m.Name = types.StringValue(vol.Name)
	m.Type = types.StringValue(vol.Type)
	m.Location = types.StringValue(vol.Location)
	m.Description = types.StringValue(vol.Description)
	m.ContentType = types.StringValue(vol.ContentType)
	m.Config = config

	m.Target = types.StringValue("")
	if server.IsClustered() || vol.Location != "none" {
		m.Target = types.StringValue(vol.Location)
	}

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// ComputedKeys returns list of computed config keys.
func (_ StorageVolumeModel) ComputedKeys() []string {
	return []string{
		"block.filesystem",
		"block.mount_options",
		"volatile.",
	}
}
