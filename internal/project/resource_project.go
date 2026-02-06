package project

import (
	"context"
	"fmt"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// ProjectModel resource data model that matches the schema.
type ProjectModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`

	// Used to remove unused images from a project on delete.
	// This is because Terraform does not track images cached by LXD, but if they
	// are not removed, the project cannot be deleted. Setting this to false will
	// throw an error if images are still present in the project on delete.
	CleanupImagesOnDestroy types.Bool `tfsdk:"cleanup_images_on_destroy"`
}

// ProjectResource represent LXD project resource.
type ProjectResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewProjectResource return new project resource.
func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// Metadata for project resource.
func (r ProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema for project resource.
func (r ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"cleanup_images_on_destroy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert project config schema to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	projectName := plan.Name.ValueString()
	server, err := r.provider.InstanceServer(remote, projectName, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	projectReq := api.ProjectsPost{
		Name: projectName,
		ProjectPut: api.ProjectPut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateProject(projectReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create project %q", projectName), err.Error())
		return
	}

	// Partially update state to make Terraform aware of the created resource.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), projectName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	projectName := state.Name.ValueString()
	server, err := r.provider.InstanceServer(remote, projectName, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	projectName := plan.Name.ValueString()
	server, err := r.provider.InstanceServer(remote, projectName, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	project, etag, err := server.GetProject(projectName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing project %q", projectName), err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge project state and user defined configuration.
	config := common.MergeConfig(project.Config, userConfig, plan.ComputedKeys())

	// Update project.
	newProject := api.ProjectPut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateProject(projectName, newProject, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update project %q", projectName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	projectName := state.Name.ValueString()
	server, err := r.provider.InstanceServer(remote, projectName, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// If the provider is allowed to cleanup images on destroy, ensure they are removed
	// before deleting the project. Images are removed only if they are the only resources
	// left in the project.
	if state.CleanupImagesOnDestroy.ValueBool() {
		project, _, err := server.GetProject(projectName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve project %q", projectName), err.Error())
			return
		}

		// Determine if images need to be removed based on project's UsedBy field.
		// This ensures images are not removed if project has "features.images" disabled
		// and prevents image deletion if images are not the only resources left.
		onlyImagesLeft := true
		hasImages := false
		for _, usedBy := range project.UsedBy {
			if usedBy == "/1.0/profiles/default?project="+projectName {
				// Ignore default profile which cannot be removed.
				continue
			}

			if !strings.HasPrefix(usedBy, "/1.0/images/") {
				onlyImagesLeft = false
				break
			}

			hasImages = true
		}

		if hasImages && onlyImagesLeft {
			images, err := server.GetImages()
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve images from project %q", projectName), err.Error())
				return
			}

			for _, img := range images {
				op, err := server.DeleteImage(img.Fingerprint)
				if err == nil {
					err = op.WaitContext(ctx)
				}

				if err != nil {
					resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove unused image %q from project %q", img.Fingerprint, projectName), err.Error())
					return
				}
			}
		}
	}

	err = server.DeleteProject(projectName, false)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove project %q", projectName), err.Error())
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "project",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		// Attribute "project" is parsed by default, but we use
		// attribute "name" instead.
		if k == "project" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid import ID %q", req.ID),
				"Valid import format:\nimport lxd_project.<resource> [remote:]<name>",
			)
			break
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// SyncState fetches the server's current state for a project and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r ProjectResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ProjectModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	projectName := m.Name.ValueString()
	project, _, err := server.GetProject(projectName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve project %q", projectName), err.Error())
		return respDiags
	}

	// Extract user defined config and merge it with current config state.
	stateConfig := common.StripConfig(project.Config, m.Config, m.ComputedKeys())

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	respDiags.Append(diags...)

	m.Name = types.StringValue(project.Name)
	m.Description = types.StringValue(project.Description)
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// ComputedKeys returns list of computed config keys.
func (m ProjectModel) ComputedKeys() []string {
	return []string{
		"features.images",
		"features.profiles",
		"features.storage.volumes",
		"features.storage.buckets",
	}
}
