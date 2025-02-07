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
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type StorageBucketModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Pool        types.String `tfsdk:"pool"`
	Project     types.String `tfsdk:"project"`
	Target      types.String `tfsdk:"target"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`

	// Computed.
	Location types.String `tfsdk:"location"`
}

// StorageBucketResource represent LXD storage bucket resource.
type StorageBucketResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewStorageBucketResource return a new storage bucket resource.
func NewStorageBucketResource() resource.Resource {
	return &StorageBucketResource{}
}

func (r StorageBucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_bucket"
}

func (r StorageBucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *StorageBucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r StorageBucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageBucketModel

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

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolName := plan.Pool.ValueString()
	bucketName := plan.Name.ValueString()

	bucket := api.StorageBucketsPost{
		Name: bucketName,
		StorageBucketPut: api.StorageBucketPut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	_, err = server.CreateStoragePoolBucket(poolName, bucket)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage bucket %q", bucketName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageBucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageBucketModel

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

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r StorageBucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageBucketModel

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
	bucketName := plan.Name.ValueString()
	_, etag, err := server.GetStoragePoolBucket(poolName, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve storage bucket %q", bucketName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newBucket := api.StorageBucketPut{
		Config:      config,
		Description: plan.Description.ValueString(),
	}

	err = server.UpdateStoragePoolBucket(poolName, bucketName, newBucket, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage bucket %q", bucketName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageBucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageBucketModel

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

	poolName := state.Pool.ValueString()
	bucketName := state.Name.ValueString()
	err = server.DeleteStoragePoolBucket(poolName, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete storage bucket %q", bucketName), err.Error())
		return
	}
}

func (r StorageBucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "storage_bucket",
		RequiredFields: []string{"pool", "name"},
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

// SyncState fetches the server's current state for a storage bucket and
// updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r StorageBucketResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m StorageBucketModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	poolName := m.Pool.ValueString()
	bucketName := m.Name.ValueString()
	bucket, _, err := server.GetStoragePoolBucket(poolName, bucketName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage bucket %q", bucketName), err.Error())
		return respDiags
	}

	// Extract user defined config and merge it with current config state.
	stateConfig := common.StripConfig(bucket.Config, m.Config, m.ComputedKeys())

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	respDiags.Append(diags...)

	m.Name = types.StringValue(bucket.Name)
	m.Location = types.StringValue(bucket.Location)
	m.Description = types.StringValue(bucket.Description)
	m.Config = config

	m.Target = types.StringValue("")
	if server.IsClustered() || bucket.Location != "none" {
		m.Target = types.StringValue(bucket.Location)
	}

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// ComputedKeys returns list of computed config keys.
func (m StorageBucketModel) ComputedKeys() []string {
	return []string{
		"block.filesystem",
		"block.mount_options",
		"volatile.",
	}
}
