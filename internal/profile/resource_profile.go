package profile

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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

// ProfileModel represents a LXD profile.
type ProfileModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Devices     types.Set    `tfsdk:"device"`
	Config      types.Map    `tfsdk:"config"`
}

// ProfileResource represent LXD profile resource.
type ProfileResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewProfileResource returns a new profile resource.
func NewProfileResource() resource.Resource {
	return &ProfileResource{}
}

// Metadata for profile resource.
func (r ProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_profile", req.ProviderTypeName)
}

// Schema for profile resource.
func (r ProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
		},

		Blocks: map[string]schema.Block{
			"device": schema.SetNestedBlock{
				Description: "Profile device",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Device name",
						},

						"type": schema.StringAttribute{
							Required:    true,
							Description: "Device type",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"none", "disk", "nic", "unix-char",
									"unix-block", "usb", "gpu", "infiniband",
									"proxy", "unix-hotplug", "tpm", "pci",
								),
							},
						},

						"properties": schema.MapAttribute{
							Required:    true,
							Description: "Device properties",
							ElementType: types.StringType,
							Validators: []validator.Map{
								// Prevent empty values.
								mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r ProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfileModel

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

	// Convert profile config and devices to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	profileName := plan.Name.ValueString()

	profile := api.ProfilesPost{
		Name: profileName,
		ProfilePut: api.ProfilePut{
			Description: plan.Description.ValueString(),
			Config:      config,
			Devices:     devices,
		},
	}

	if profileName == "default" {
		// Resolve empty project name.
		projectName := project
		if projectName == "" {
			projectName = "default"
		}

		err := checkDefaultProject(server, projectName, profileName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Cannot import existing profile %q from project %q", profileName, projectName), err.Error())
			return
		}

		// Default profile is automatically created in each project and we cannot remove it.
		// However, if default profile is added, instead of creating it, we need fetch the
		// existing one and update it.
		_, etag, err := server.GetProfile(profileName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing profile %q", profileName), err.Error())
			return
		}

		err = server.UpdateProfile(profileName, profile.ProfilePut, etag)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to update profile %q", profile.Name), err.Error())
			return
		}
	} else {
		err = server.CreateProfile(profile)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create profile %q", profile.Name), err.Error())
			return
		}
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfileModel

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
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfileModel

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

	profileName := plan.Name.ValueString()
	_, etag, err := server.GetProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing profile %q", profileName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update profile.
	profile := api.ProfilePut{
		Description: plan.Description.ValueString(),
		Config:      config,
		Devices:     devices,
	}

	if profileName == "default" {
		// Resolve empty project name.
		projectName := project
		if projectName == "" {
			projectName = "default"
		}

		// Ensure default profile is not located within the default project. This can
		// occur if the profiles's project feature `feature.profiles` was manually
		// changed after the default profile was made managed.
		err := checkDefaultProject(server, projectName, profileName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf(`Cannot update profile "default" in project "%q"`, projectName), err.Error())
			return
		}
	}

	err = server.UpdateProfile(profileName, profile, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update profile %q", profileName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfileModel

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

	profileName := state.Name.ValueString()

	// Default profile cannot be removed.
	if profileName == "default" {
		// If default profile is located in the default project, simply remove it
		// from the state.
		if checkDefaultProject(server, project, profileName) != nil {
			return
		}

		// Otherwise, try to empty the profile's configuration to ensure the profile
		// is not being used by any resource.
		profile := api.ProfilePut{
			Description: "",
			Config:      nil,
			Devices:     nil,
		}

		// Also ignore the not found error, which may occur if a project where
		// the profile is located is already removed.
		err = server.UpdateProfile(profileName, profile, "")
		if err != nil && !errors.IsNotFoundError(err) {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to empty configuration of the profile %q", profileName), err.Error())
		}
	} else {
		err = server.DeleteProfile(profileName)
		if err != nil && !errors.IsNotFoundError(err) {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove profile %q", profileName), err.Error())
		}
	}
}

func (r ProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "profile",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// SyncState fetches the server's current state for a profile and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r ProfileResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m ProfileModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	profileName := m.Name.ValueString()
	profile, _, err := server.GetProfile(profileName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve profile %q", profileName), err.Error())
		return respDiags
	}

	// Convert config state and devices into schema types.
	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(profile.Config), m.Config)
	respDiags.Append(diags...)

	devices, diags := common.ToDeviceSetType(ctx, profile.Devices)
	respDiags.Append(diags...)

	m.Name = types.StringValue(profile.Name)
	m.Description = types.StringValue(profile.Description)
	m.Devices = devices
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// checkDefaultProject returns an error if default profile is located within the default
// project or if the project does not exist.
func checkDefaultProject(server lxd.InstanceServer, projectName string, profileName string) error {
	if profileName != "default" {
		// Nothing to check for.
		return nil
	}

	// Default profile in default project cannot be managed as there may be resources
	// linked to that profile that cannot be managed by Terraform.
	if projectName == "" || projectName == "default" {
		return fmt.Errorf(`Profile "default" cannot be managed in project "default"`)
	}

	project, _, err := server.GetProject(projectName)
	if err != nil {
		return err
	}

	// Ensure project has "features.profiles" disabled. If this feature is enabled,
	// project's profiles are located in default project, which we cannot manage.
	feature, ok := project.Config["features.profiles"]
	if !ok || shared.IsFalse(feature) {
		return fmt.Errorf(`Project %q has "features.profiles" disabled which means the profile "default" is located in project "default". This profile cannot be managed.`, projectName)
	}

	return nil
}
