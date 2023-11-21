package image

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// LxdPublishImageResourceModel resource data model that matches the schema.
type LxdPublishImageResourceModel struct {
	Instance       types.String `tfsdk:"instance"`
	Aliases        types.Set    `tfsdk:"aliases"`
	Properties     types.Map    `tfsdk:"properties"`
	Public         types.Bool   `tfsdk:"public"`
	Filename       types.String `tfsdk:"filename"`
	CompressionAlg types.String `tfsdk:"compression_algorithm"`
	Triggers       types.List   `tfsdk:"triggers"`
	Project        types.String `tfsdk:"project"`
	Remote         types.String `tfsdk:"remote"`

	// Computed.
	ResourceID   types.String `tfsdk:"resource_id"`
	Architecture types.String `tfsdk:"architecture"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
}

// LxdPublishImageResource represent LXD publish image resource.
type LxdPublishImageResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdPublishImageResource return new publish image resource.
func NewLxdPublishImageResource() resource.Resource {
	return &LxdPublishImageResource{}
}

// Metadata for publish image resource.
func (r LxdPublishImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_publish_image", req.ProviderTypeName)
}

// Schema for publish image resource.
func (r LxdPublishImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"aliases": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					// Prevent empty values.
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},

			"properties": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
					mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},

			"public": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"filename": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"compression_algorithm": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("gzip"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("bzip2", "gzip", "lzma", "xz", "none"),
				},
			},

			"triggers": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
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

			"resource_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"architecture": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"fingerprint": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"created_at": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *LxdPublishImageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r LxdPublishImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LxdPublishImageResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := data.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	instanceName := data.Instance.ValueString()
	ct, _, err := server.GetInstanceState(instanceName)
	if err != nil { // && errors.IsNotFoundError(err)
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return
	}

	if ct.StatusCode != api.Stopped {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot publish image because instance %q is running", instanceName), "")
		return
	}

	imageProps, diags := common.ToConfigMap(ctx, data.Properties)
	resp.Diagnostics.Append(diags...)

	aliases, diags := ToAliasList(ctx, data.Aliases)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	imageAliases := make([]api.ImageAlias, 0, len(aliases))
	for _, alias := range aliases {
		// Ensure image alias does not already exist.
		aliasTarget, _, _ := server.GetImageAlias(alias)
		if aliasTarget != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Image alias %q already exists", alias), "")
			return
		}

		ia := api.ImageAlias{
			Name: alias,
		}

		imageAliases = append(imageAliases, ia)
	}

	imageReq := api.ImagesPost{
		Aliases:              imageAliases,
		Filename:             data.Filename.ValueString(),
		CompressionAlgorithm: data.CompressionAlg.ValueString(),
		ImagePut: api.ImagePut{
			Public:     data.Public.ValueBool(),
			Properties: imageProps,
		},
		Source: &api.ImagesPostSource{
			Name: data.Instance.ValueString(),
			Type: "instance",
		},
	}

	// Publish image.
	op, err := server.CreateImage(imageReq, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to publish instance %q image", instanceName), err.Error())
		return
	}

	// Wait for create operation to finish.
	err = op.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to publish instance %q image", instanceName), err.Error())
		return
	}

	// Extract fingerprint from operation response.
	opResp := op.Get()
	imageFingerprint := opResp.Metadata["fingerprint"].(string)
	data.Fingerprint = types.StringValue(imageFingerprint)

	imageID := createImageResourceID(remote, imageFingerprint)
	data.ResourceID = types.StringValue(imageID)

	_, diags = data.SyncState(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdPublishImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LxdPublishImageResourceModel

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

	found, diags := data.SyncState(ctx, server)
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

func (r LxdPublishImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LxdPublishImageResourceModel

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

	_, imageFingerprint := splitImageResourceID(data.ResourceID.ValueString())

	imageProps, diags := common.ToConfigMap(ctx, data.Properties)
	resp.Diagnostics.Append(diags...)

	oldAliases, diags := ToAliasList(ctx, data.Aliases)
	resp.Diagnostics.Append(diags...)

	newAliases := make([]string, 0, len(data.Aliases.Elements()))
	diags = req.State.GetAttribute(ctx, path.Root("aliases"), &newAliases)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract removed and added image aliases.
	removed, added := utils.DiffSlices(oldAliases, newAliases)

	// Delete removed aliases.
	for _, alias := range removed {
		err := server.DeleteImageAlias(alias)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete alias %q for published image", alias), err.Error())
			return
		}
	}

	// Add new aliases.
	for _, alias := range added {
		req := api.ImageAliasesPost{}
		req.Name = alias
		req.Target = imageFingerprint

		err := server.CreateImageAlias(req)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create alias %q for published image", alias), err.Error())
			return
		}
	}

	imageReq := api.ImagePut{
		Properties: imageProps,
	}

	err = server.UpdateImage(imageFingerprint, imageReq, "")
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update publihsed image properties"), err.Error())
		return
	}

	_, diags = data.SyncState(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdPublishImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LxdPublishImageResourceModel

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

	_, imageFingerprint := splitImageResourceID(data.ResourceID.ValueString())
	opDelete, err := server.DeleteImage(imageFingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove published image"), err.Error())
		return
	}

	err = opDelete.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove published image"), err.Error())
		return
	}
}

// SyncState pulls published image data from the server and updates the model
// in-place.
// This should be called before updating Terraform state.
func (m *LxdPublishImageResourceModel) SyncState(ctx context.Context, server lxd.InstanceServer) (bool, diag.Diagnostics) {
	respDiags := diag.Diagnostics{}

	_, imageFingerprint := splitImageResourceID(m.ResourceID.ValueString())

	image, _, err := server.GetImage(imageFingerprint)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve published image"), err.Error())
		return true, respDiags
	}

	configAliases, diags := ToAliasList(ctx, m.Aliases)
	respDiags.Append(diags...)

	// Copy aliases from image state that are present in user defined
	// config.
	var aliases []string
	for _, a := range image.Aliases {
		if utils.ValueInSlice(a.Name, configAliases) {
			aliases = append(aliases, a.Name)
		}
	}

	aliasSet, diags := ToAliasSetType(ctx, aliases)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return true, respDiags
	}

	m.Fingerprint = types.StringValue(image.Fingerprint)
	m.Architecture = types.StringValue(image.Architecture)
	m.CreatedAt = types.Int64Value(image.CreatedAt.Unix())
	m.Public = types.BoolValue(image.Public)
	m.Aliases = aliasSet

	return true, nil
}
