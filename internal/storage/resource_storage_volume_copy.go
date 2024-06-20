package storage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"

	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type StorageVolumeCopyModel struct {
	Name         types.String `tfsdk:"name"`
	Pool         types.String `tfsdk:"pool"`
	SourceName   types.String `tfsdk:"source_name"`
	SourcePool   types.String `tfsdk:"source_pool"`
	SourceRemote types.String `tfsdk:"source_remote"`
	Project      types.String `tfsdk:"project"`
	Target       types.String `tfsdk:"target"`
	Remote       types.String `tfsdk:"remote"`
}

// StorageVolumeCopyResource represent Incus storage volume copy resource.
type StorageVolumeCopyResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewStorageVolumeCopyResource returns a new storage volume copy resource.
func NewStorageVolumeCopyResource() resource.Resource {
	return &StorageVolumeCopyResource{}
}

func (r StorageVolumeCopyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_volume_copy", req.ProviderTypeName)
}

func (r StorageVolumeCopyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"pool": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"source_pool": schema.StringAttribute{
				Required:    true,
				Description: "The source pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"source_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the source volume.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"source_remote": schema.StringAttribute{
				Optional:    true,
				Description: "The remote from which the source volume is copied.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *StorageVolumeCopyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r StorageVolumeCopyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageVolumeCopyModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dstProject := plan.Project.ValueString()
	dstTarget := plan.Target.ValueString()
	dstServer, err := r.provider.InstanceServer(plan.Remote.ValueString(), dstProject, dstTarget)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	srcServer, err := r.provider.InstanceServer(plan.SourceRemote.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	dstName := plan.Name.ValueString()
	dstPool := plan.Pool.ValueString()
	srcName := plan.SourceName.ValueString()
	srcPool := plan.SourcePool.ValueString()

	dstVolID := fmt.Sprintf("%s/%s", dstPool, dstName)
	srcVolID := fmt.Sprintf("%s/%s", srcPool, srcName)

	srcVol := api.StorageVolume{
		Name: srcName,
		Type: "custom",
	}

	args := incus.StoragePoolVolumeCopyArgs{
		Name:       dstName,
		VolumeOnly: true,
	}

	opCopy, err := dstServer.CopyStoragePoolVolume(dstPool, srcServer, srcPool, srcVol, &args)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy storage volume %q -> %q", srcVolID, dstVolID), err.Error())
		return
	}

	err = opCopy.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy storage volume %q -> %q", srcVolID, dstVolID), err.Error())
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeCopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r StorageVolumeCopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r StorageVolumeCopyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
