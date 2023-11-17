package storage

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

type LxdStoragePoolResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Driver      types.String `tfsdk:"driver"`
	Project     types.String `tfsdk:"project"`
	Target      types.String `tfsdk:"target"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
	ConfigState types.Map    `tfsdk:"config_state"`
}

// Sync pulls storage pool data from the server and updates the model in-place.
// It returns a boolean indicating whether resource is found and diagnostics
// that contain potential errors.
// This should be called before updating Terraform state.
func (m *LxdStoragePoolResourceModel) Sync(server lxd.InstanceServer, poolName string) (bool, diag.Diagnostics) {
	pool, _, err := server.GetStoragePool(poolName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		return true, diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve storage pool %q", poolName), err.Error()),
		}
	}

	config, diags := common.ToConfigMapType(context.Background(), pool.Config)
	if diags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(pool.Name)
	m.Description = types.StringValue(pool.Description)
	m.Driver = types.StringValue(pool.Driver)
	m.ConfigState = config

	return true, nil
}

// LxdStoragePoolResource represent LXD storage pool resource.
type LxdStoragePoolResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdStoragePoolResource returns a new storage pool resource.
func NewLxdStoragePoolResource() resource.Resource {
	return &LxdStoragePoolResource{}
}

// Metadata for storage pool resource.
func (r LxdStoragePoolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_pool", req.ProviderTypeName)
}

// Schema for storage pool resource.
func (r LxdStoragePoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			// Config represents user defined LXD config file.
			"config": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},

			// Config state represents actual LXD resource state.
			// It is managed solely by the provider. User config
			// is merged into it.
			"config_state": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *LxdStoragePoolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LxdStoragePoolResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If resource is being destroyed req.Config will be null.
	// In such case there is no need for plan modification.
	if req.Config.Raw.IsNull() {
		return
	}

	var driver string

	diags := req.Config.GetAttribute(ctx, path.Root("driver"), &driver)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	common.ModifyConfigStatePlan(ctx, req, resp, r.ComputedKeys(driver))
}

func (r LxdStoragePoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *LxdStoragePoolResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	// Convert pool config to map.
	config, diag := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool := api.StoragePoolsPost{
		Name:   data.Name.ValueString(),
		Driver: data.Driver.ValueString(),
		StoragePoolPut: api.StoragePoolPut{
			Description: data.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateStoragePool(pool)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage pool %q", pool.Name), err.Error())
		return
	}

	found, diags := data.Sync(server, pool.Name)
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
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdStoragePoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *LxdStoragePoolResourceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	poolName := data.Name.ValueString()

	found, diags := data.Sync(server, poolName)
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
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdStoragePoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *LxdStoragePoolResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	poolName := data.Name.ValueString()
	_, etag, err := server.GetStoragePool(poolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage pool %q", poolName), err.Error())
		return
	}

	// Merge LXD state and user configs.
	userConfig, diags := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diags...)

	stateConfig, diags := common.ToConfigMap(ctx, data.ConfigState)
	resp.Diagnostics.Append(diags...)

	driver := data.Driver.ValueString()
	config := common.MergeConfig(stateConfig, userConfig, r.ComputedKeys(driver))

	if resp.Diagnostics.HasError() {
		return
	}

	// Update pool.
	pool := api.StoragePoolPut{
		Description: data.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateStoragePool(poolName, pool, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage pool %q", poolName), err.Error())
		return
	}

	found, diags := data.Sync(server, poolName)
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
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdStoragePoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *LxdStoragePoolResourceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	poolName := data.Name.ValueString()
	err = server.DeleteStoragePool(poolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove storage pool %q", poolName), err.Error())
	}
}

func (r *LxdStoragePoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

// ComputedKeys returns list of computed LXD config keys.
func (r LxdStoragePoolResource) ComputedKeys(driver string) []string {
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

	// TODO: Add regex support to ignore all keys
	// of certain type.
	return append(keys, "volatile.*")
}
