package storage

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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

type StorageBucketKeyModel struct {
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Pool          types.String `tfsdk:"pool"`
	StorageBucket types.String `tfsdk:"storage_bucket"`
	Role          types.String `tfsdk:"role"`
	Project       types.String `tfsdk:"project"`
	Remote        types.String `tfsdk:"remote"`

	// Computed.
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

// StorageBucketKeyResource represent Incus storage bucket key resource.
type StorageBucketKeyResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewStorageBucketKeyResource return a new storage bucket key resource.
func NewStorageBucketKeyResource() resource.Resource {
	return &StorageBucketKeyResource{}
}

func (r StorageBucketKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_bucket_key", req.ProviderTypeName)
}

// TODO: setup proper schema for storage bucket key like volume for pool!
func (r StorageBucketKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"storage_bucket": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"role": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("read-only"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("admin", "read-only"),
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

			// Computed.

			"access_key": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},

			"secret_key": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func (r *StorageBucketKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r StorageBucketKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageBucketKeyModel

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

	poolName := plan.Pool.ValueString()
	bucketName := plan.StorageBucket.ValueString()

	// Ensure storage bucket exists.
	_, _, err = server.GetStoragePoolBucket(poolName, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve storage bucket %q", bucketName), err.Error())
		return
	}

	keyName := plan.Name.ValueString()

	key := api.StorageBucketKeysPost{
		StorageBucketKeyPut: api.StorageBucketKeyPut{
			Description: plan.Description.ValueString(),
			Role:        plan.Role.ValueString(),
		},
		Name: keyName,
	}

	_, err = server.CreateStoragePoolBucketKey(poolName, bucketName, key)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage bucket key %q of %q", keyName, bucketName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageBucketKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageBucketKeyModel

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

func (r StorageBucketKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageBucketKeyModel

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

	poolName := plan.Pool.ValueString()
	bucketName := plan.StorageBucket.ValueString()

	// Ensure strorage bucket exists.
	_, _, err = server.GetStoragePoolBucket(poolName, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve storage bucket %q", bucketName), err.Error())
		return
	}

	keyName := plan.Name.ValueString()
	key, etag, err := server.GetStoragePoolBucketKey(poolName, bucketName, keyName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve storage bucket key %q of bucket %q", keyName, bucketName), err.Error())
		return
	}

	newKey := api.StorageBucketKeyPut{
		Description: plan.Description.ValueString(),
		Role:        plan.Role.ValueString(),
		// As we do not want to update the access key and the secret key, we provide the existing values for the update.
		AccessKey: key.AccessKey,
		SecretKey: key.SecretKey,
	}

	err = server.UpdateStoragePoolBucketKey(poolName, bucketName, keyName, newKey, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage bucket key %q of bucket %q", keyName, bucketName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageBucketKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageBucketKeyModel

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
	bucketName := state.StorageBucket.ValueString()

	// Ensure storage bucket exists.
	_, _, err = server.GetStoragePoolBucket(poolName, bucketName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve storage bucket %q", bucketName), err.Error())
		return
	}

	keyName := state.Name.ValueString()
	err = server.DeleteStoragePoolBucketKey(poolName, bucketName, keyName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete storage bucket key %q of bucket %q", keyName, bucketName), err.Error())
		return
	}
}

func (r StorageBucketKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "storage_bucket_key",
		RequiredFields: []string{"pool", "storage_bucket", "name"},
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

// SyncState fetches the server's current state for a storage bucket key and
// updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r StorageBucketKeyResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m StorageBucketKeyModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	poolName := m.Pool.ValueString()
	bucketName := m.StorageBucket.ValueString()
	keyName := m.Name.ValueString()
	key, _, err := server.GetStoragePoolBucketKey(poolName, bucketName, keyName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage bucket key %q of bucket %q", keyName, bucketName), err.Error())
		return respDiags
	}

	m.Name = types.StringValue(key.Name)
	m.Description = types.StringValue(key.Description)
	m.Role = types.StringValue(key.Role)
	m.AccessKey = types.StringValue(key.AccessKey)
	m.SecretKey = types.StringValue(key.SecretKey)

	return tfState.Set(ctx, &m)
}
