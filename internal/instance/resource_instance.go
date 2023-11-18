package instance

import (
	"context"
	"fmt"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

type LxdInstanceFileModel struct {
	Content    types.String `tfsdk:"content"`
	Source     types.String `tfsdk:"source"`
	TargetFile types.String `tfsdk:"target_file"`
	UserID     types.Int64  `tfsdk:"uid"`
	GroupID    types.Int64  `tfsdk:"gid"`
	Mode       types.String `tfsdk:"mode"`
	CreateDirs types.String `tfsdk:"create_directories"`
}

type LxdInstanceResourceModel struct {
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	Image          types.String `tfsdk:"image"`
	Ephemeral      types.Bool   `tfsdk:"ephemeral"`
	StartOnCreate  types.Bool   `tfsdk:"start_on_create"`
	WaitForNetwork types.Bool   `tfsdk:"wait_for_network"`
	Profiles       types.List   `tfsdk:"profiles"`
	Devices        types.Set    `tfsdk:"device"`
	Files          types.Set    `tfsdk:"file"`
	Limits         types.Map    `tfsdk:"limits"`
	Config         types.Map    `tfsdk:"config"`
	Project        types.String `tfsdk:"project"`
	Remote         types.String `tfsdk:"remote"`
	Target         types.String `tfsdk:"target"`

	// Computed.
	IPv4   types.String `tfsdk:"ipv4_address"`
	IPv6   types.String `tfsdk:"ipv6_address"`
	MAC    types.String `tfsdk:"mac_address"`
	Status types.String `tfsdk:"status"`
}

// LxdInstanceResource represent LXD instance resource.
type LxdInstanceResource struct {
	provider      *provider_config.LxdProviderConfig
	updateTimeout int
}

// NewLxdInstanceResource returns a new instance resource.
func NewLxdInstanceResource() resource.Resource {
	return &LxdInstanceResource{}
}

// Metadata for instance resource.
func (r LxdInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_instance", req.ProviderTypeName)
}

// Schema for instance resource.
func (r LxdInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("container"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("container", "virtual-machine"),
				},
			},

			"image": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"ephemeral": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"start_on_create": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},

			"wait_for_network": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},

			// If profiles are null, use "default" profile.
			// If profiles lengeth is 0, no profiles are applied.
			"profiles": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					// Prevent empty values.
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},

			"limits": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Validators: []validator.Map{
					// Prevent empty keys or values.
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
					mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
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

			// Output.
			"ipv4_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Output.
			"ipv6_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Output.
			"mac_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Output.
			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Config represents user defined LXD config file.
			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Validators: []validator.Map{
					mapvalidator.KeysAre(configKeyValidator{}),
				},
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

			"file": schema.SetNestedBlock{
				Description: "Upload file to instance",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"content": schema.StringAttribute{
							Optional: true,
						},

						"source": schema.StringAttribute{
							Optional: true,
						},

						"target_file": schema.StringAttribute{
							Required: true,
						},

						"uid": schema.Int64Attribute{
							Optional: true,
						},

						"gid": schema.Int64Attribute{
							Optional: true,
						},

						"mode": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString("0775"),
						},

						"create_directories": schema.BoolAttribute{
							Optional: true,
						},

						"append": schema.BoolAttribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func (r *LxdInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Set instance update timeout (for starting/stopping the instance).
	r.updateTimeout = int(time.Duration(time.Minute * 5).Seconds())

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

func (r *LxdInstanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If resource is being destroyed req.Config will be null.
	// In such case there is no need for plan modification.
	if req.Config.Raw.IsNull() {
		return
	}

	var profiles types.List
	req.Config.GetAttribute(ctx, path.Root("profiles"), &profiles)

	// If profiles are null, set "default" profile.
	if profiles.IsNull() {
		resp.Plan.SetAttribute(ctx, path.Root("profiles"), []string{"default"})
	}

	common.ModifyConfigStatePlan(ctx, req, resp, r.ComputedKeys())
}

func (r LxdInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LxdInstanceResourceModel

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

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	// Evaluate image remote.
	image := data.Image.ValueString()
	imageRemote := remote
	imageParts := strings.SplitN(image, ":", 2)
	if len(imageParts) == 2 {
		imageRemote = imageParts[0]
		image = imageParts[1]
	}

	imageServer, err := r.provider.ImageServer(imageRemote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewImageServerError(err))
	}

	// Extract profiles, devices, config and limits.
	profiles, diags := ToProfileList(ctx, data.Profiles)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, data.Devices)
	resp.Diagnostics.Append(diags...)

	config, diags := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diags...)

	limits, diags := common.ToConfigMap(ctx, data.Limits)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Merge limits into instance config.
	for k, v := range limits {
		key := fmt.Sprintf("limits.%s", k)
		config[key] = v
	}

	// Prepare instance request.
	instance := api.InstancesPost{
		Name: data.Name.ValueString(),
		Type: api.InstanceType(data.Type.ValueString()),
		InstancePut: api.InstancePut{
			Description: data.Description.ValueString(),
			Ephemeral:   data.Ephemeral.ValueBool(),
			Config:      config,
			Profiles:    profiles,
			Devices:     devices,
		},
	}

	var imageInfo *api.Image

	// Gather info about source image.
	conn, _ := imageServer.GetConnectionInfo()
	if conn.Protocol == "simplestreams" {
		// Optimisation for simplestreams.
		imageInfo = &api.Image{}
		imageInfo.Public = true
		imageInfo.Fingerprint = image
		instance.Source.Alias = image
	} else {
		// Attempt to resolve an image alias.
		alias, _, err := imageServer.GetImageAlias(image)
		if err == nil {
			image = alias.Target
			instance.Source.Alias = image
		}

		// Get the image info.
		imageInfo, _, err = imageServer.GetImage(image)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retireve image info for instance %q", instance.Name), err.Error())
			return
		}
	}

	// Create instance. It will not be running after this API call.
	opCreate, err := server.CreateInstanceFromImage(imageServer, *imageInfo, instance)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create instance %q", instance.Name), err.Error())
		return
	}

	// Wait for the instance to be created.
	err = opCreate.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create instance %q", instance.Name), err.Error())
		return
	}

	// Partially update state, to make terraform aware of
	// an existing instance.
	diags = resp.State.SetAttribute(ctx, path.Root("name"), instance.Name)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Upload files.
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		files, diags := common.ToFileMap(ctx, data.Files)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, f := range files {
			err := common.InstanceFileUpload(server, instance.Name, f)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to instance %q", instance.Name), err.Error())
				return
			}
		}
	}

	if data.StartOnCreate.ValueBool() {
		// Start instance.
		startReq := api.InstanceStatePut{
			Action:  "start",
			Force:   false,
			Timeout: r.updateTimeout,
		}

		opStart, err := server.UpdateInstanceState(instance.Name, startReq, "")
		if err != nil {
			// Instance has been created, but daemon rejected start request.
			resp.Diagnostics.AddError(fmt.Sprintf("LXD server rejected request to start instance %q", instance.Name), err.Error())
			return
		}

		err = opStart.Wait()
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to start instance %q", instance.Name), err.Error())
			return
		}

		// Even though op.Wait has completed, wait until we can see
		// the instance is running via a new API call.
		stateConf := &retry.StateChangeConf{
			Target:     []string{"Running"},
			Timeout:    3 * time.Minute,
			MinTimeout: 3 * time.Second,
			Delay:      r.provider.RefreshInterval(),
			Refresh:    instanceStateRefreshFunc(server, instance.Name),
		}

		_, err = stateConf.WaitForStateContext(ctx)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to wait for instance %q to become active", instance.Name), err.Error())
			return
		}

		if data.WaitForNetwork.ValueBool() {
			// LXD will return "Running" even if "inet" has not yet
			// been set. Therefore wait until we see an "inet" IP
			// before reading the state.
			networkConf := &retry.StateChangeConf{
				Target:     []string{"OK"},
				Timeout:    3 * time.Minute,
				MinTimeout: 3 * time.Second,
				Delay:      r.provider.RefreshInterval(),
				Refresh:    instanceNetworkStateRefreshFunc(server, instance.Name),
			}

			_, err = networkConf.WaitForStateContext(ctx)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to wait for instance %q network information", instance.Name), err.Error())
				return
			}
		}
	}

	_, diags = data.Sync(server, instance.Name)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LxdInstanceResourceModel

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

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	instanceName := data.Name.ValueString()

	found, diags := data.Sync(server, instanceName)
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

func (r LxdInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *LxdInstanceResourceModel

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

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	instanceName := data.Name.ValueString()
	instance, etag, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", instanceName), err.Error())
		return
	}

	// First extract profiles, devices, limits, config and config state.
	// Then merge user defined config with instance config (state).
	profiles, diags := ToProfileList(ctx, data.Profiles)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, data.Devices)
	resp.Diagnostics.Append(diags...)

	limits, diag := common.ToConfigMap(ctx, data.Limits)
	resp.Diagnostics.Append(diag...)

	userConfig, diags := common.ToConfigMap(ctx, data.Config)
	resp.Diagnostics.Append(diags...)

	config := common.MergeConfig(instance.Config, userConfig, r.ComputedKeys())

	if resp.Diagnostics.HasError() {
		return
	}

	// Merge limits into instance config.
	for k, v := range limits {
		key := fmt.Sprintf("limits.%s", k)
		config[key] = v
	}

	// Update instance.
	newInstance := api.InstancePut{
		Description:  data.Description.ValueString(),
		Ephemeral:    data.Ephemeral.ValueBool(),
		Architecture: instance.Architecture,
		Restore:      instance.Restore,
		Stateful:     instance.Stateful,
		Config:       config,
		Profiles:     profiles,
		Devices:      devices,
	}

	opUpdate, err := server.UpdateInstance(instanceName, newInstance, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update instance %q", instance.Name), err.Error())
		return
	}

	err = opUpdate.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update instance %q", instance.Name), err.Error())
		return
	}

	// Fetch old files from state.
	var oldFileSet types.Set
	diags = req.State.GetAttribute(ctx, path.Root("file"), &oldFileSet)
	resp.Diagnostics.Append(diags...)

	oldFiles, diags := common.ToFileMap(ctx, oldFileSet)
	resp.Diagnostics.Append(diags...)

	newFiles, diags := common.ToFileMap(ctx, data.Files)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Remove files that are no longer present in newFiles.
	for k, f := range oldFiles {
		_, ok := newFiles[k]
		if ok {
			continue
		}

		targetFile := f.TargetFile.ValueString()
		err := common.InstanceFileDelete(server, instanceName, targetFile)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file from instance %q", instanceName), err.Error())
			return
		}
	}

	// Upload new files.
	for k, f := range newFiles {
		_, ok := oldFiles[k]
		if ok {
			continue
		}

		err := common.InstanceFileUpload(server, instanceName, f)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to instance %q", instanceName), err.Error())
			return
		}
	}

	_, diags = data.Sync(server, instanceName)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LxdInstanceResourceModel

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

	// Set target if configured.
	target := data.Target.ValueString()
	if target != "" {
		server = server.UseTarget(target)
	}

	instanceName := data.Name.ValueString()

	ct, etag, _ := server.GetInstanceState(instanceName)
	if ct.Status == "Running" {
		stopReq := api.InstanceStatePut{
			Action:  "stop",
			Timeout: r.updateTimeout,
		}

		opStop, err := server.UpdateInstanceState(instanceName, stopReq, etag)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to stop instance %q", instanceName), err.Error())
			return
		}

		err = opStop.Wait()
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed waiting for instance %q to stop", instanceName), err.Error())
			return
		}

		// Even though op.Wait has completed, wait until we can see the instance has
		// stopped via a new API call.
		stateConf := &retry.StateChangeConf{
			Target:     []string{"Stopped"},
			Timeout:    3 * time.Minute,
			MinTimeout: 3 * time.Second,
			Delay:      r.provider.RefreshInterval(),
			Refresh:    instanceStateRefreshFunc(server, instanceName),
		}

		_, err = stateConf.WaitForStateContext(ctx)
		if err != nil {
			if errors.IsNotFoundError(err) {
				// Ephemeral instances will be deleted when they are stopped
				// so we can just return here and end the Delete call early.
				return
			}

			resp.Diagnostics.AddError(fmt.Sprintf("Failed waiting for instance %q to stop", instanceName), err.Error())
			return
		}
	}

	opDelete, err := server.DeleteInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove instance %q", instanceName), err.Error())
		return
	}

	// Wait for the instance to be deleted.
	err = opDelete.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove instance %q", instanceName), err.Error())
	}
}

func (r *LxdInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	remote, project, name, diag := common.SplitImportID(req.ID, "instance")
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	if remote != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	}

	if project != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

// Sync pulls instance data from the server and updates the model in-place.
// It returns a boolean indicating whether resource is found and diagnostics
// that contain potential errors.
// This should be called before updating Terraform state.
func (m *LxdInstanceResourceModel) Sync(server lxd.InstanceServer, instanceName string) (bool, diag.Diagnostics) {
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		return true, diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve instance %q", instanceName), err.Error()),
		}
	}

	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		return true, diag.Diagnostics{
			diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error()),
		}
	}

	// Reset IPv4, IPv6, and MAC addresses. If case instance has lost
	// network connectivity, we should reflect that in state.
	m.IPv4 = types.StringNull()
	m.IPv6 = types.StringNull()
	m.MAC = types.StringNull()

	// First there is an access_interface set, extract IPv4, IPv4, and
	// MAC addresses from it.
	var accIfaceFound bool
	accIface, ok := instance.Config["user.access_interface"]
	if ok {
		net := instanceState.Network[accIface]

		ipv4, mac, ok := findIPv4Address(net)
		if ok {
			m.IPv4 = types.StringValue(ipv4)
			m.MAC = types.StringValue(mac)
			accIfaceFound = true
		}

		ipv6, ok := findIPv6Address(net)
		if ok {
			m.IPv6 = types.StringValue(ipv6)
		}
	}

	// If the above wasn't successful, try to automatically determine
	// the IPv4, IPv6, and MAC addresses.
	if !accIfaceFound {
		for iface, net := range instanceState.Network {
			if iface == "lo" {
				continue
			}

			ipv4, mac, ok := findIPv4Address(net)
			if ok {
				m.IPv4 = types.StringValue(ipv4)
				m.MAC = types.StringValue(mac)
			}

			ipv6, ok := findIPv6Address(net)
			if ok {
				m.IPv6 = types.StringValue(ipv6)
			}
		}
	}

	instanceLimits := make(map[string]string)
	for k, v := range instance.Config {
		// Remove enteries with untracked prefixes.
		if utils.HasAnyPrefix(k, []string{"image.", "volatile."}) {
			delete(instance.Config, k)
			continue
		}

		// Extract enteries with "limits." prefix.
		key, ok := strings.CutPrefix(k, "limits.")
		if ok {
			instanceLimits[key] = v
			delete(instance.Config, k)
		}
	}

	config, diags := common.ToConfigMapType(context.Background(), instance.Config)
	if diags.HasError() {
		return true, diags
	}

	limits, diags := common.ToConfigMapType(context.Background(), instanceLimits)
	if diags.HasError() {
		return true, diags
	}

	profiles, diags := ToProfileListType(context.Background(), instance.Profiles)
	if diags.HasError() {
		return true, diags
	}

	devices, diags := common.ToDeviceSetType(context.Background(), instance.Devices)
	if diags.HasError() {
		return true, diags
	}

	m.Name = types.StringValue(instance.Name)
	m.Description = types.StringValue(instance.Description)
	m.Status = types.StringValue(instance.Status)
	m.Ephemeral = types.BoolValue(instance.Ephemeral)
	m.Profiles = profiles
	m.Limits = limits
	m.Devices = devices
	m.Config = config

	m.Type = types.StringValue(instance.Type)
	if instance.Type == "" {
		// If the LXD server does not support virtualization or the
		// instances API is not available, instance.Type might be a
		// blank string. In that case we fall back to "container"
		// to avoid constant changes to the resource definition.
		m.Type = types.StringValue("container")
	}

	if instance.Location != "" && instance.Location != "none" {
		m.Target = types.StringValue(instance.Location)
	}

	return true, nil
}

// ComputedKeys returns list of computed LXD config keys.
func (r LxdInstanceResource) ComputedKeys() []string {
	return []string{
		"image.",
		"volatile.",
	}
}

// ToProfileList converts profiles of type types.List into []string.
//
// If profiles are null, use "default" profile.
// If profiles lengeth is 0, no profiles are applied.
func ToProfileList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() {
		return []string{"default"}, nil
	}

	profiles := make([]string, 0, len(list.Elements()))
	diags := list.ElementsAs(ctx, &profiles, false)

	return profiles, diags
}

// ToProfileListType converts []string into profiles of type types.List.
func ToProfileListType(ctx context.Context, list []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, list)
}

// networkStateRefreshFunc returns function that refreshes instance's status.
func instanceStateRefreshFunc(server lxd.InstanceServer, instanceName string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		st, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return st, "Error", err
		}

		return st, st.Status, nil
	}
}

// instanceNetworkStateRefreshFunc returns function that checks
// whether instance has received an IP address.
func instanceNetworkStateRefreshFunc(server lxd.InstanceServer, instanceName string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		st, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return st, "Error", err
		}

		for iface, net := range st.Network {
			if iface == "lo" {
				continue
			}

			for _, ip := range net.Addresses {
				if ip.Family == "inet" {
					return st, "OK", nil
				}
			}
		}

		return st, "NOT FOUND", nil
	}
}

// findIPv4Address searches the network for last IPv4 address. If an IP address
// is found, interface's MAC address is also returned.
func findIPv4Address(network api.InstanceStateNetwork) (string, string, bool) {
	var ipv4, mac string
	for _, ip := range network.Addresses {
		if ip.Family == "inet" {
			ipv4 = ip.Address
			mac = network.Hwaddr
		}
	}

	return ipv4, mac, (ipv4 != "")
}

// Find last global IPv6 address or return any last IPv6 address
// if there is no global address. This works analog to the IPv4
// selection mechanism but favors global addresses.
func findIPv6Address(network api.InstanceStateNetwork) (string, bool) {
	var ipv6 string

	for _, ip := range network.Addresses {
		if ip.Family == "inet6" && ip.Scope == "global" {
			ipv6 = ip.Address
		}
	}

	if ipv6 != "" {
		return ipv6, true
	}

	for _, ip := range network.Addresses {
		if ip.Family == "inet6" {
			return ip.Address, true
		}
	}

	return ipv6, (ipv6 != "")
}
