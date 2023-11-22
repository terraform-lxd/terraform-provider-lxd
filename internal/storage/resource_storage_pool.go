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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type StoragePoolModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Driver      types.String `tfsdk:"driver"`
	Project     types.String `tfsdk:"project"`
	Target      types.String `tfsdk:"target"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
}

// StoragePoolResource represent LXD storage pool resource.
type StoragePoolResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewStoragePoolResource returns a new storage pool resource.
func NewStoragePoolResource() resource.Resource {
	return &StoragePoolResource{}
}

func (r StoragePoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_pool", req.ProviderTypeName)
}

func (r StoragePoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"driver": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("dir", "zfs", "lvm", "btrfs", "ceph", "cephfs", "cephobject"),
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
		},
	}
}

func (r *StoragePoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r StoragePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StoragePoolModel

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

	// Set target if configured.
	target := plan.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	// Convert pool config to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool := api.StoragePoolsPost{
		Name:   plan.Name.ValueString(),
		Driver: plan.Driver.ValueString(),
		StoragePoolPut: api.StoragePoolPut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateStoragePool(pool)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage pool %q", pool.Name), err.Error())
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

func (r StoragePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StoragePoolModel

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

	// Set target if configured.
	target := state.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
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

func (r StoragePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StoragePoolModel

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

	// Set target if configured.
	target := plan.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	poolName := plan.Name.ValueString()
	pool, etag, err := server.GetStoragePool(poolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage pool %q", poolName), err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge pool config state and user defined config.
	config := common.MergeConfig(pool.Config, userConfig, plan.ComputedKeys(pool.Driver))

	// Update pool.
	newPool := api.StoragePoolPut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateStoragePool(poolName, newPool, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage pool %q", poolName), err.Error())
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

func (r StoragePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StoragePoolModel

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

	// Set target if configured.
	target := state.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	poolName := state.Name.ValueString()
	err = server.DeleteStoragePool(poolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove storage pool %q", poolName), err.Error())
	}
}

func (r StoragePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	remote, project, name, diag := common.SplitImportID(req.ID, "storage_pool")
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

// Sync pulls storage pool data from the server and updates the model
// in-place. It returns a boolean indicating whether resource is found and
// diagnostics that contain potential errors.
// This should be called before updating Terraform state.
func (m *StoragePoolModel) Sync(ctx context.Context, server lxd.InstanceServer) (bool, diag.Diagnostics) {
	respDiags := diag.Diagnostics{}

	poolName := m.Name.ValueString()
	pool, _, err := server.GetStoragePool(poolName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage pool %q", poolName), err.Error())
		return true, respDiags
	}

	// Extract user defined config and merge it with current config state.
	userConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)

	stateConfig := common.StripConfig(pool.Config, userConfig, m.ComputedKeys(pool.Driver))

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(pool.Name)
	m.Description = types.StringValue(pool.Description)
	m.Driver = types.StringValue(pool.Driver)
	m.Config = config

	return true, nil
}

// ComputedKeys returns list of computed config keys.
func (_ StoragePoolModel) ComputedKeys(driver string) []string {
	var keys []string

	switch driver {
	case "dir":
		keys = []string{
			"source",
		}
	case "zfs":
		keys = []string{
			"source",
			"size",
			"zfs.pool_name",
		}
	case "lvm":
		keys = []string{
			"source",
			"size",
			"lvm.vg_name",
			"lvm.thinpool_name",
		}
	case "btrfs":
		keys = []string{
			"source",
			"size",
		}
	case "ceph":
		// TODO
	case "cephfs":
		// TODO
	case "cephobject":
		// TODO
	}

	return append(keys, "volatile.")
}
