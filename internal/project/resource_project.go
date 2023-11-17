package project

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// LxdProjectResourceModel resource data model that matches the schema.
type LxdProjectResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
	ConfigState types.Map    `tfsdk:"config_state"`
}

// Sync pulls project data from the server and updates the model in-place.
// This should be called before updating Terraform state.
func (m *LxdProjectResourceModel) Sync(server lxd.InstanceServer, projectName string) diag.Diagnostics {
	project, _, err := server.UseProject(projectName).GetProject(projectName)
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve project %q", projectName), err.Error()),
		}
	}

	config, diags := common.ToConfigMapType(context.Background(), project.Config)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(project.Name)
	m.Description = types.StringValue(project.Description)
	m.ConfigState = config

	return nil
}

// LxdProjectResource represent LXD project resource.
type LxdProjectResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdProjectResource return new project resource.
func NewLxdProjectResource() resource.Resource {
	return &LxdProjectResource{}
}

// Metadata for project resource.
func (r LxdProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_project", req.ProviderTypeName)
}

// Schema for project resource.
func (r LxdProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *LxdProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LxdProjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {

	// TODO: Always remove computed state from the plan.
	// TODO: Add changes to the plan that are being applied from user config.
	//    If user value is not set (null/unknown): ignore.
	//    If user value is an empty string       : unset specific value.
	//    If user value is set                   : overwrite this value.

	common.ModifyConfigStatePlan(ctx, req, resp, r.ComputedKeys())
}

func (r LxdProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *LxdProjectResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert project config schema to map.
	config, diag := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	project := api.ProjectsPost{
		Name: data.Name.ValueString(),
		ProjectPut: api.ProjectPut{
			Description: data.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateProject(project)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create project %q", project.Name), err.Error())
		return
	}

	diags = data.Sync(server, project.Name)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *LxdProjectResourceModel

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

	projectName := data.Name.ValueString()

	diags = data.Sync(server, projectName)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *LxdProjectResourceModel

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

	projectName := data.Name.ValueString()
	_, etag, err := server.UseProject(projectName).GetProject(projectName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing project %q", projectName), err.Error())
		return
	}

	// Merge LXD state and user configurations.
	userConfig, diags := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diags...)

	stateConfig, diags := common.ToConfigMap(ctx, data.ConfigState)
	resp.Diagnostics.Append(diags...)

	config := common.MergeConfig(stateConfig, userConfig, r.ComputedKeys())

	if resp.Diagnostics.HasError() {
		return
	}

	// Update project.
	newProject := api.ProjectPut{
		Description: data.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateProject(projectName, newProject, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update project %q", projectName), err.Error())
		return
	}

	diags = data.Sync(server, projectName)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *LxdProjectResourceModel

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

	projectName := data.Name.ValueString()
	err = server.UseProject(projectName).DeleteProject(projectName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove project %q", projectName), err.Error())
	}
}

// ComputedKeys returns list of compuuted LXD config keys.
func (r LxdProjectResource) ComputedKeys() []string {
	return []string{
		"features.images",
		"features.profiles",
		"features.storage.volumes",
		"features.storage.buckets",
	}
}
