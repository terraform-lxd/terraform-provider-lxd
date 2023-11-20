package storage

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type LxdStorageVolumeCopyResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Pool         types.String `tfsdk:"pool"`
	SourceName   types.String `tfsdk:"source_name"`
	SourcePool   types.String `tfsdk:"source_pool"`
	SourceRemote types.String `tfsdk:"source_remote"`
	Project      types.String `tfsdk:"project"`
	Target       types.String `tfsdk:"target"`
	Remote       types.String `tfsdk:"remote"`
}

// LxdStorageVolumeCopyResource represent LXD storage volume copy resource.
type LxdStorageVolumeCopyResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdStorageVolumeCopyResource returns a new storage volume copy resource.
func NewLxdStorageVolumeCopyResource() resource.Resource {
	return &LxdStorageVolumeCopyResource{}
}

func (r LxdStorageVolumeCopyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_volume_copy", req.ProviderTypeName)
}

func (r LxdStorageVolumeCopyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *LxdStorageVolumeCopyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r LxdStorageVolumeCopyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LxdStorageVolumeCopyResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dstServer, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	dstProject := data.Project.ValueString()
	dstTarget := data.Target.ValueString()

	if dstProject != "" {
		dstServer = dstServer.UseProject(dstProject)
	}

	if dstTarget != "" {
		dstServer = dstServer.UseTarget(dstTarget)
	}

	srcServer, err := r.provider.InstanceServer(data.SourceRemote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	dstName := data.Name.ValueString()
	dstPool := data.Pool.ValueString()
	srcName := data.SourceName.ValueString()
	srcPool := data.SourcePool.ValueString()

	dstVolID := fmt.Sprintf("%s/%s", dstPool, dstName)
	srcVolID := fmt.Sprintf("%s/%s", srcPool, srcName)

	srcVol := api.StorageVolume{
		Name: srcName,
		Type: "custom",
	}

	args := lxd.StoragePoolVolumeCopyArgs{
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
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdStorageVolumeCopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r LxdStorageVolumeCopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r LxdStorageVolumeCopyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
