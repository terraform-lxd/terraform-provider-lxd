package profile

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type LxdProfileDeviceModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Properties types.Map    `tfsdk:"properties"`
}

type LxdProfileResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Devices     types.Set    `tfsdk:"device"`
	Config      types.Map    `tfsdk:"config"`
	ConfigState types.Map    `tfsdk:"config_state"`
}

// Sync pulls profile data from the server and updates the model in-place.
// This should be called before updating Terraform state.
func (m *LxdProfileResourceModel) Sync(server lxd.InstanceServer, profileName string) diag.Diagnostics {
	profile, _, err := server.GetProfile(profileName)
	if err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve profile %q", profileName), err.Error()),
		}
	}

	config, diags := common.ToConfigMapType(context.Background(), profile.Config)
	if diags.HasError() {
		return diags
	}

	devices, diags := fromDeviceMap(context.Background(), profile.Devices)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(profile.Name)
	m.Description = types.StringValue(profile.Description)
	m.Devices = devices
	m.ConfigState = config

	return nil
}

// LxdProfileResource represent LXD profile resource.
type LxdProfileResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdProfileResource returns a new profile resource.
func NewLxdProfileResource() resource.Resource {
	return &LxdProfileResource{}
}

// Metadata for profile resource.
func (r LxdProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_profile", req.ProviderTypeName)
}

// Schema for profile resource.
func (r LxdProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *LxdProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LxdProfileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	common.ModifyConfigStatePlan(ctx, req, resp, nil)
}

func (r LxdProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *LxdProfileResourceModel

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

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Convert profile config to map.
	config, diag := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diag...)

	devices, diags := toDeviceMap(ctx, data.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	profile := api.ProfilesPost{
		Name: data.Name.ValueString(),
		ProfilePut: api.ProfilePut{
			Description: data.Description.ValueString(),
			Config:      config,
			Devices:     devices,
		},
	}

	err = server.CreateProfile(profile)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create profile %q", profile.Name), err.Error())
		return
	}

	diags = data.Sync(server, profile.Name)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *LxdProfileResourceModel

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

	profileName := data.Name.ValueString()

	diags = data.Sync(server, profileName)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *LxdProfileResourceModel

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

	// Set project if configured.
	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	profileName := data.Name.ValueString()
	_, etag, err := server.GetProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing profile %q", profileName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diags...)

	devices, diags := toDeviceMap(ctx, data.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update profile.
	profile := api.ProfilePut{
		Description: data.Description.ValueString(),
		Config:      config,
		Devices:     devices,
	}

	err = server.UpdateProfile(profileName, profile, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update profile %q", profileName), err.Error())
		return
	}

	diags = data.Sync(server, profileName)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *LxdProfileResourceModel

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

	profileName := data.Name.ValueString()
	err = server.DeleteProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove profile %q", profileName), err.Error())
	}
}

// toDeviceMap converts deviecs from types.Set into map[string]map[string]string.
func toDeviceMap(ctx context.Context, dataDevices types.Set) (map[string]map[string]string, diag.Diagnostics) {
	if dataDevices.IsNull() || dataDevices.IsUnknown() {
		return make(map[string]map[string]string), nil
	}

	// Convert types.Set into set of device models.
	modelDevices := make([]LxdProfileDeviceModel, len(dataDevices.Elements()))
	diags := dataDevices.ElementsAs(ctx, &modelDevices, false)
	if diags.HasError() {
		return nil, diags
	}

	devices := make(map[string]map[string]string, len(modelDevices))
	for _, d := range modelDevices {
		devName := d.Name.ValueString()
		devType := d.Type.ValueString()

		// Convert properties into map[string]string.
		device := make(map[string]string, len(d.Properties.Elements()))
		if !d.Properties.IsNull() && !d.Properties.IsUnknown() {
			diags := d.Properties.ElementsAs(ctx, &device, false)
			if diags.HasError() {
				return nil, diags
			}
		}

		device["type"] = devType
		devices[devName] = device
	}

	return devices, nil
}

// fromDeviceMap converts deviecs from map[string]map[string]string into types.Set.
func fromDeviceMap(ctx context.Context, devices map[string]map[string]string) (types.Set, diag.Diagnostics) {
	devModelTypes := map[string]attr.Type{
		"name":       types.StringType,
		"type":       types.StringType,
		"properties": types.MapType{types.StringType},
	}

	if len(devices) == 0 {
		return types.SetNull(types.ObjectType{devModelTypes}), nil
	}

	modelDevices := make([]LxdProfileDeviceModel, 0, len(devices))
	for key := range devices {
		props := devices[key]

		devName := types.StringValue(key)
		devType := types.StringValue(props["type"])

		// Remove type from properties, as we manage it
		// outside properties.
		delete(props, "type")

		devProps, diags := types.MapValueFrom(ctx, types.StringType, props)
		if diags.HasError() {
			return types.SetNull(types.ObjectType{devModelTypes}), diags
		}

		dev := LxdProfileDeviceModel{
			Name:       devName,
			Type:       devType,
			Properties: devProps,
		}

		modelDevices = append(modelDevices, dev)
	}

	return types.SetValueFrom(ctx, types.ObjectType{devModelTypes}, modelDevices)
}
