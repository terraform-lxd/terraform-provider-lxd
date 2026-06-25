package image

import (
	"context"
	"fmt"
	"slices"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

// ImageModel resource data model that matches the schema.
type ImageModel struct {
	SourceImage    types.Object `tfsdk:"source_image"`
	SourceInstance types.Object `tfsdk:"source_instance"`
	Aliases        types.Set    `tfsdk:"aliases"`
	Project        types.String `tfsdk:"project"`
	Remote         types.String `tfsdk:"remote"`

	// Computed.
	ResourceID    types.String `tfsdk:"resource_id"`
	CreatedAt     types.Int64  `tfsdk:"created_at"`
	Fingerprint   types.String `tfsdk:"fingerprint"`
	CopiedAliases types.Set    `tfsdk:"copied_aliases"`
}

type SourceImageModel struct {
	Image        types.String `tfsdk:"image"`
	Type         types.String `tfsdk:"type"`
	Architecture types.String `tfsdk:"architecture"`
	CopyAliases  types.Bool   `tfsdk:"copy_aliases"`
}

type SourceInstanceModel struct {
	Name     types.String `tfsdk:"name"`
	Snapshot types.String `tfsdk:"snapshot"`
}

// ImageResource represent LXD image resource.
type ImageResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewImageResource return new image resource.
func NewImageResource() resource.Resource {
	return &ImageResource{}
}

func (r ImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (r ImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"source_image": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"image": schema.StringAttribute{
						Required: true,
					},
					"type": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("container"),
						Validators: []validator.String{
							stringvalidator.OneOf("container", "virtual-machine"),
						},
					},
					"architecture": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
						Validators: []validator.String{
							architectureValidator{},
						},
					},
					"copy_aliases": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},

			"source_instance": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"snapshot": schema.StringAttribute{
						Optional: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},

			"aliases": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.Set{
					// Prevent empty values.
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},

			"project": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(provider_config.DefaultProject),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			// Computed attributes.

			"resource_id": schema.StringAttribute{
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

			"fingerprint": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"copied_aliases": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ImageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r ImageResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if req.Config.Raw.IsNull() {
		return
	}

	var config ImageModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.SourceImage.IsNull() && config.SourceInstance.IsNull() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either source_image or source_instance must be set.",
		)
		return
	}

	if !config.SourceImage.IsNull() && !config.SourceInstance.IsNull() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Only source_image or source_instance can be set.",
		)
		return
	}
}

func (r ImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ImageModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.SourceImage.IsNull() {
		r.createImageFromSourceImage(ctx, resp, &plan)
		return
	} else if !plan.SourceInstance.IsNull() {
		r.createImageFromSourceInstance(ctx, resp, &plan)
		return
	}
}

func (r ImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ImageModel

	// Fetch resource model from Terraform state.
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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state, true)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ImageModel
	var state ImageModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	// Extract image metadata.
	_, imageFingerprint := splitImageResourceID(plan.ResourceID.ValueString())

	// Get info about the image.
	image, _, err := server.GetImage(imageFingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve image with fingerprint %q", imageFingerprint), err.Error())
		return
	}

	// Parse current (old) image aliases.
	oldAliases := make([]string, len(image.Aliases))
	for i, alias := range image.Aliases {
		oldAliases[i] = alias.Name
	}

	// Parse expected (new) image aliases.
	copiedAliases := make([]string, 0, len(plan.CopiedAliases.Elements()))
	diags := req.State.GetAttribute(ctx, path.Root("copied_aliases"), &copiedAliases)
	resp.Diagnostics.Append(diags...)

	newAliases, diags := ToAliasList(ctx, plan.Aliases)
	resp.Diagnostics.Append(diags...)

	newAliases = slices.Compact(append(newAliases, copiedAliases...))

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract removed and added image aliases.
	removed, added := utils.DiffSlices(oldAliases, newAliases)

	// Delete removed aliases.
	for _, alias := range removed {
		err := server.DeleteImageAlias(alias)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete alias %q for image with fingerprint %q", alias, imageFingerprint), err.Error())
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
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create alias %q for image with fingerprint %q", alias, imageFingerprint), err.Error())
			return
		}
	}

	plan.Fingerprint = types.StringValue(imageFingerprint)

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ImageModel

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

	_, imageFingerprint := splitImageResourceID(state.ResourceID.ValueString())

	opDelete, err := server.DeleteImage(imageFingerprint)
	if err == nil {
		err = opDelete.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove image with fingerprint %q", imageFingerprint), err.Error())
		return
	}
}

// TaintState marks the state with identity fields required to target the image.
func (m ImageModel) TaintState(ctx context.Context, tfState *tfsdk.State) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Append(tfState.SetAttribute(ctx, path.Root("resource_id"), m.ResourceID.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("project"), m.Project.ValueString())...)
	diags.Append(tfState.SetAttribute(ctx, path.Root("remote"), m.Remote.ValueString())...)

	return diags
}

// SyncState fetches the server's current state for an image and updates the provided model.
// It then applies this updated model as the new state in Terraform.
func (r ImageResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ImageModel, forgetOnNotFound bool) diag.Diagnostics {
	var respDiags diag.Diagnostics

	_, imageFingerprint := splitImageResourceID(m.ResourceID.ValueString())

	image, _, err := server.GetImage(imageFingerprint)
	if err != nil {
		if forgetOnNotFound && errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to sync state for image with fingerprint %q", imageFingerprint), err.Error())
		return respDiags
	}

	if !m.SourceImage.IsNull() {
		var sourceImageModel SourceImageModel
		respDiags = m.SourceImage.As(ctx, &sourceImageModel, basetypes.ObjectAsOptions{})
		if respDiags.HasError() {
			return respDiags
		}

		// Store architecture if computed
		if sourceImageModel.Architecture.IsNull() || sourceImageModel.Architecture.IsUnknown() {
			sourceImageModel.Architecture = types.StringValue(image.Architecture)
			m.SourceImage, respDiags = types.ObjectValue(m.SourceImage.AttributeTypes(ctx), map[string]attr.Value{
				"image":        sourceImageModel.Image,
				"type":         sourceImageModel.Type,
				"architecture": sourceImageModel.Architecture,
				"copy_aliases": sourceImageModel.CopyAliases,
			})
			if respDiags.HasError() {
				return respDiags
			}
		}
	}

	configAliases, diags := ToAliasList(ctx, m.Aliases)
	respDiags.Append(diags...)

	copiedAliases, diags := ToAliasList(ctx, m.CopiedAliases)
	respDiags.Append(diags...)

	// Extract aliases from image that are either present in user defined
	// config or are not copied from initial remote image.
	var aliases []string
	for _, a := range image.Aliases {
		if utils.ValueInSlice(a.Name, configAliases) || !utils.ValueInSlice(a.Name, copiedAliases) {
			aliases = append(aliases, a.Name)
		}
	}

	aliasSet, diags := ToAliasSetType(ctx, aliases)
	respDiags.Append(diags...)

	m.Fingerprint = types.StringValue(image.Fingerprint)
	m.CreatedAt = types.Int64Value(image.CreatedAt.Unix())
	m.Aliases = aliasSet

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

func (r ImageResource) createImageFromSourceImage(ctx context.Context, resp *resource.CreateResponse, plan *ImageModel) {
	var sourceImageModel SourceImageModel

	diags := plan.SourceImage.As(ctx, &sourceImageModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	imageType := sourceImageModel.Type.ValueString()

	var imageRemote string

	image := sourceImageModel.Image.ValueString()
	imageParts := strings.SplitN(image, ":", 2)
	if len(imageParts) == 2 {
		imageRemote = imageParts[0]
		image = imageParts[1]
	}

	imageServer, err := r.provider.ImageServer(imageRemote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewImageServerError(err))
		return
	}

	// Determine the correct image for the specified architecture.
	architecture := sourceImageModel.Architecture.ValueString()
	if architecture != "" {
		availableArchitectures, err := imageServer.GetImageAliasArchitectures(imageType, image)
		if err == nil {
			// Find the image alias that matches the specified architecture.
			found := false
			for imageArchitecture, imageAlias := range availableArchitectures {
				if imageArchitecture == architecture {
					image = imageAlias.Target
					found = true
				}
			}

			if !found {
				keys := make([]string, 0, len(availableArchitectures))
				for key := range availableArchitectures {
					keys = append(keys, key)
				}

				keyList := strings.Join(keys, ", ")
				resp.Diagnostics.AddError(fmt.Sprintf("No image alias found for architecture %q. Available architectures: %s ", architecture, keyList), "")
				return
			}
		} else if !errors.IsNotFoundError(err) {
			resp.Diagnostics.AddError("Failed to get image alias architectures", err.Error())
			return
		}
	}

	// Determine whether the user has provided a fingerprint or an alias.
	aliasTarget, _, _ := imageServer.GetImageAliasType(imageType, image)
	if aliasTarget != nil {
		image = aliasTarget.Target
	}

	aliases, diags := ToAliasList(ctx, plan.Aliases)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
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

	// Get data about remote image (also checks if image exists).
	imageInfo, _, err := imageServer.GetImage(image)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve info about image %q", image), err.Error())
		return
	}

	// Store remote aliases that we've copied, so we can filter them out later.
	copied := make([]string, 0)
	if sourceImageModel.CopyAliases.ValueBool() {
		for _, a := range imageInfo.Aliases {
			copied = append(copied, a.Name)

			// Skip aliases already defined by the user.
			if slices.Contains(aliases, a.Name) {
				continue
			}

			imageAliases = append(imageAliases, api.ImageAlias{Name: a.Name})
		}
	}

	// Copy image.
	args := lxd.ImageCopyArgs{
		Aliases: imageAliases,
		Public:  false,
	}

	opCopy, err := server.CopyImage(imageServer, *imageInfo, &args)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy image %q", image), err.Error())
		return
	}

	// Wait for copy operation to finish.
	err = opCopy.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy image %q", image), err.Error())
		return
	}

	copiedAliases, diags := types.SetValueFrom(ctx, types.StringType, copied)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	imageID := createImageResourceID(remote, imageInfo.Fingerprint)
	plan.ResourceID = types.StringValue(imageID)

	plan.CopiedAliases = copiedAliases

	diags = plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, *plan, false)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) createImageFromSourceInstance(ctx context.Context, resp *resource.CreateResponse, plan *ImageModel) {
	var sourceInstanceModel SourceInstanceModel

	diags := plan.SourceInstance.As(ctx, &sourceInstanceModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := sourceInstanceModel.Name.ValueString()
	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return
	}

	if sourceInstanceModel.Snapshot.IsNull() && instanceState.StatusCode != api.Stopped {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot publish image because instance %q is running", instanceName), "")
		return
	}

	aliases, diags := ToAliasList(ctx, plan.Aliases)
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

	var source *api.ImagesPostSource
	if !sourceInstanceModel.Snapshot.IsNull() {
		snapshotName := sourceInstanceModel.Snapshot.ValueString()
		source = &api.ImagesPostSource{
			Name: fmt.Sprintf("%s/%s", instanceName, snapshotName),
			Type: "snapshot",
		}
	} else {
		source = &api.ImagesPostSource{
			Name: instanceName,
			Type: "instance",
		}
	}

	imageReq := api.ImagesPost{
		Aliases:  imageAliases,
		ImagePut: api.ImagePut{},
		Source:   source,
	}

	// Publish image.
	op, err := server.CreateImage(imageReq, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to publish instance %q image", instanceName), err.Error())
		return
	}

	// Wait for create operation to finish.
	err = op.WaitContext(ctx)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to publish instance %q image", instanceName), err.Error())
		return
	}

	// Extract fingerprint from operation response.
	opResp := op.Get()
	imageFingerprint, ok := opResp.Metadata["fingerprint"].(string)
	if !ok {
		resp.Diagnostics.AddError("Failed to determine fingerprint of the published image", "")
		return
	}

	imageID := createImageResourceID(remote, imageFingerprint)

	plan.Fingerprint = types.StringValue(imageFingerprint)
	plan.ResourceID = types.StringValue(imageID)
	plan.CopiedAliases = types.SetValueMust(types.StringType, []attr.Value{})

	diags = plan.TaintState(ctx, &resp.State)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, *plan, false)
	resp.Diagnostics.Append(diags...)
}

// ToAliasList converts aliases of type types.Set into a slice of strings.
func ToAliasList(ctx context.Context, aliasSet types.Set) ([]string, diag.Diagnostics) {
	if aliasSet.IsNull() || aliasSet.IsUnknown() {
		return []string{}, nil
	}

	aliases := make([]string, 0, len(aliasSet.Elements()))
	diags := aliasSet.ElementsAs(ctx, &aliases, false)
	return aliases, diags
}

// ToAliasSetType converts slice of strings into aliases of type types.Set.
func ToAliasSetType(ctx context.Context, aliases []string) (types.Set, diag.Diagnostics) {
	if len(aliases) == 0 {
		// Prevent null value if slice is empty.
		return types.SetValueMust(types.StringType, []attr.Value{}), nil
	}

	return types.SetValueFrom(ctx, types.StringType, aliases)
}

// createImageResourceID creates new image ID by concatenating remote and
// image fingerprint using colon.
func createImageResourceID(remote string, fingerprint string) string {
	return fmt.Sprintf("%s:%s", remote, fingerprint)
}

// splitImageResourceID splits an image ID into remote and fingerprint strings.
func splitImageResourceID(id string) (remote string, fingerprint string) {
	pieces := strings.SplitN(id, ":", 2)
	if len(pieces) != 2 {
		return "", id
	}

	return pieces[0], pieces[1]
}
