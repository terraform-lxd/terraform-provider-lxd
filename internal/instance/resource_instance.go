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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type InstanceModel struct {
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

// InstanceResource represent LXD instance resource.
type InstanceResource struct {
	provider      *provider_config.LxdProviderConfig
	updateTimeout int
}

// NewInstanceResource returns a new instance resource.
func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

func (r InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_instance", req.ProviderTypeName)
}

func (r InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Validators: []validator.Map{
					mapvalidator.KeysAre(configKeyValidator{}),
				},
			},

			// Computed.

			"ipv4_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"ipv6_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"mac_address": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},

						"create_directories": schema.BoolAttribute{
							Optional: true,
						},

						// Append is here just to satisfy the LxdFile model.
						"append": schema.BoolAttribute{
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func (r *InstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Set instance update timeout (for starting/stopping the instance).
	r.updateTimeout = int(time.Duration(5 * time.Minute).Seconds())

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

func (r *InstanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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
}

func (r InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Evaluate image remote.
	image := plan.Image.ValueString()
	imageRemote := remote
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

	// Extract profiles, devices, config and limits.
	profiles, diags := ToProfileList(ctx, plan.Profiles)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	limits, diags := common.ToConfigMap(ctx, plan.Limits)
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
		Name: plan.Name.ValueString(),
		Type: api.InstanceType(plan.Type.ValueString()),
		InstancePut: api.InstancePut{
			Description: plan.Description.ValueString(),
			Ephemeral:   plan.Ephemeral.ValueBool(),
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
	if !plan.Files.IsNull() && !plan.Files.IsUnknown() {
		files, diags := common.ToFileMap(ctx, plan.Files)
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

	if plan.StartOnCreate.ValueBool() {
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

		if plan.WaitForNetwork.ValueBool() {
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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceModel
	var state InstanceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := plan.Name.ValueString()
	instance, etag, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", instanceName), err.Error())
		return
	}

	// First extract profiles, devices, limits, config and config state.
	// Then merge user defined config with instance config (state).
	profiles, diags := ToProfileList(ctx, plan.Profiles)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	limits, diag := common.ToConfigMap(ctx, plan.Limits)
	resp.Diagnostics.Append(diag...)

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	config := common.MergeConfig(instance.Config, userConfig, plan.ComputedKeys())

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
		Description:  plan.Description.ValueString(),
		Ephemeral:    plan.Ephemeral.ValueBool(),
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

	oldFiles, diags := common.ToFileMap(ctx, state.Files)
	resp.Diagnostics.Append(diags...)

	newFiles, diags := common.ToFileMap(ctx, plan.Files)
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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := state.Name.ValueString()

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

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "instance",
		RequiredFields: []string{"name"},
		AllowedOptions: []string{"image"},
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

// SyncState fetches the server's current state for an instance and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r InstanceResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m InstanceModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	instanceName := m.Name.ValueString()
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve instance %q", instanceName), err.Error())
		return respDiags
	}

	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		respDiags.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return respDiags
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

	// Extract user defined config and merge it with current resource config.
	usrConfig, diags := common.ToConfigMap(ctx, m.Config)
	respDiags.Append(diags...)

	stateConfig := common.StripConfig(instance.Config, usrConfig, m.ComputedKeys())

	// Extract enteries with "limits." prefix.
	instanceLimits := make(map[string]string)
	for k, v := range stateConfig {
		key, ok := strings.CutPrefix(k, "limits.")
		if ok {
			instanceLimits[key] = v
			delete(stateConfig, k)
		}
	}

	// Convert config, limits, profiles, and devices into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig)
	respDiags.Append(diags...)

	limits, diags := common.ToConfigMapType(ctx, instanceLimits)
	respDiags.Append(diags...)

	profiles, diags := ToProfileListType(ctx, instance.Profiles)
	respDiags.Append(diags...)

	devices, diags := common.ToDeviceSetType(ctx, instance.Devices)
	respDiags.Append(diags...)

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

	m.Target = types.StringValue("")
	if server.IsClustered() || instance.Location != "none" {
		m.Target = types.StringValue(instance.Location)
	}

	// Ensure default value is set (to prevent plan diff on import).
	if m.StartOnCreate.IsNull() {
		m.StartOnCreate = types.BoolValue(true)
	}

	// Ensure default value is set (to prevent plan diff on import).
	if m.WaitForNetwork.IsNull() {
		m.WaitForNetwork = types.BoolValue(true)
	}

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// ComputedKeys returns list of computed config keys.
func (_ InstanceModel) ComputedKeys() []string {
	return []string{
		"image.",
		"volatile.",
	}
}

// ToProfileList converts profiles of type types.List into []string.
//
// If profiles are null, use "default" profile.
// If profiles lengeth is 0, no profiles are applied.
func ToProfileList(ctx context.Context, profileList types.List) ([]string, diag.Diagnostics) {
	if profileList.IsNull() {
		return []string{"default"}, nil
	}

	profiles := make([]string, 0, len(profileList.Elements()))
	diags := profileList.ElementsAs(ctx, &profiles, false)

	return profiles, diags
}

// ToProfileListType converts []string into profiles of type types.List.
func ToProfileListType(ctx context.Context, profiles []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, profiles)
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
