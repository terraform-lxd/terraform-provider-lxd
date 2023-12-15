package project

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

// ProjectModel resource data model that matches the schema.
type ProjectModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
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
	resp.TypeName = fmt.Sprintf("%s_project", req.ProviderTypeName)
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

	err = server.DeleteProject(projectName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove project %q", projectName), err.Error())
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
	usrConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)

	stateConfig := common.StripConfig(project.Config, usrConfig, m.ComputedKeys())

	// Convert config state into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig)
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
func (_ ProjectModel) ComputedKeys() []string {
	return []string{
		"features.images",
		"features.profiles",
		"features.storage.volumes",
		"features.storage.buckets",
	}
}
