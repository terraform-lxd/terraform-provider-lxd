package instance

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
	"github.com/terraform-lxd/terraform-provider-lxd/internal/utils"
)

type InstanceModel struct {
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Type               types.String `tfsdk:"type"`
	Image              types.String `tfsdk:"image"`
	Ephemeral          types.Bool   `tfsdk:"ephemeral"`
	Running            types.Bool   `tfsdk:"running"`
	AllowRestart       types.Bool   `tfsdk:"allow_restart"`
	WaitForNetwork     types.Bool   `tfsdk:"wait_for_network"`
	WaitForOperational types.Bool   `tfsdk:"wait_for_operational"`
	Profiles           types.List   `tfsdk:"profiles"`
	Devices            types.Set    `tfsdk:"device"`
	Files              types.Set    `tfsdk:"file"`
	Execs              types.Map    `tfsdk:"execs"`
	Limits             types.Map    `tfsdk:"limits"`
	Config             types.Map    `tfsdk:"config"`
	Project            types.String `tfsdk:"project"`
	Remote             types.String `tfsdk:"remote"`
	Target             types.String `tfsdk:"target"`

	// Computed.
	IPv4       types.String `tfsdk:"ipv4_address"`
	IPv6       types.String `tfsdk:"ipv6_address"`
	MAC        types.String `tfsdk:"mac_address"`
	Location   types.String `tfsdk:"location"`
	Status     types.String `tfsdk:"status"`
	Interfaces types.Map    `tfsdk:"interfaces"`

	// Timeouts.
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// InstanceResource represent LXD instance resource.
type InstanceResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewInstanceResource returns a new instance resource.
func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

func (r InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r InstanceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
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
				Optional: true,
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

			"running": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},

			"allow_restart": schema.BoolAttribute{
				Description: "Allow instance to be stopped and restarted if required by the provider for operations like migration or renaming.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			"wait_for_network": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},

			"wait_for_operational": schema.BoolAttribute{
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

			"execs": schema.MapNestedAttribute{
				Description: "Map of commands to run within the instance",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"command": schema.ListAttribute{
							Description: "Command to run within the instance",
							Required:    true,
							ElementType: types.StringType,
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
						},

						"environment": schema.MapAttribute{
							Description: "Map of additional environment variables",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
							Validators: []validator.Map{
								mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
								mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
							},
						},

						"working_dir": schema.StringAttribute{
							Description: "The directory in which the command should run",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},

						"trigger": schema.StringAttribute{
							Description: "Determines when the command should be executed",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(common.ON_CHANGE.String()),
							Validators: []validator.String{
								stringvalidator.OneOf(
									common.ON_CHANGE.String(),
									common.ON_START.String(),
									common.ONCE.String(),
								),
							},
						},

						"enabled": schema.BoolAttribute{
							Description: "Whether the command should be executed",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},

						"record_output": schema.BoolAttribute{
							Description: "Whether to record command's output (stdout and stderr)",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},

						"fail_on_error": schema.BoolAttribute{
							Description: "Whether to fail on command error",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},

						"uid": schema.Int64Attribute{
							Description: "The user ID for running command",
							Optional:    true,
						},

						"gid": schema.Int64Attribute{
							Description: "The group ID for running command",
							Optional:    true,
						},

						// Computed.

						"exit_code": schema.Int64Attribute{
							Description: "Exit code of the command",
							Computed:    true,
						},

						"stdout": schema.StringAttribute{
							Description: "Command standard output (if recorded)",
							Computed:    true,
						},

						"stderr": schema.StringAttribute{
							Description: "Command standard error (if recorded)",
							Computed:    true,
						},

						"run_count": schema.Int64Attribute{
							Description: "Internal run count indicating how many times the command was executed",
							Computed:    true,
						},
					},
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.LengthAtLeast(1),
						stringvalidator.RegexMatches(
							regexp.MustCompile("[a-z0-9_]+"),
							"Only underscore and alphanumeric characters are allowed",
						),
					),
				},
			},

			// Computed.

			"interfaces": schema.MapNestedAttribute{
				Computed:    true,
				Description: "Map of the instance network interfaces",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the network interface within an instance",
						},

						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Type of the network interface (link, local, global)",
						},

						"state": schema.StringAttribute{
							Computed:    true,
							Description: "State of the network interface (up, down)",
						},

						"ips": schema.ListNestedAttribute{
							Computed:    true,
							Description: "IP addresses assigned to the interface",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"address": schema.StringAttribute{
										Computed:    true,
										Description: "IP address",
									},

									"family": schema.StringAttribute{
										Computed:    true,
										Description: "IP family (inet, inet6)",
									},

									"scope": schema.StringAttribute{
										Computed:    true,
										Description: "Scope (local, global, link)",
									},
								},
							},
						},
					},
				},
			},

			"ipv4_address": schema.StringAttribute{
				Computed: true,
			},

			"ipv6_address": schema.StringAttribute{
				Computed: true,
			},

			"mac_address": schema.StringAttribute{
				Computed: true,
			},

			"location": schema.StringAttribute{
				Computed: true,
			},

			"status": schema.StringAttribute{
				Computed: true,
			},

			// Custom timeouts
			"timeouts": timeouts.AttributesAll(ctx),
		},

		Blocks: map[string]schema.Block{
			"device": schema.SetNestedBlock{
				Description: "Instance device",
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

						"source_path": schema.StringAttribute{
							Optional: true,
						},

						"target_path": schema.StringAttribute{
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
	var config *InstanceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If profiles in config are null, set "default" profile in plan.
	if !req.Config.Raw.IsNull() && config.Profiles.IsNull() {
		resp.Plan.SetAttribute(ctx, path.Root("profiles"), []string{"default"})
	}
}

func (r InstanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if req.Config.Raw.IsNull() {
		return
	}

	var config InstanceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	running := true
	ephemeral := false

	if !config.Ephemeral.IsNull() && !config.Ephemeral.IsUnknown() {
		ephemeral = config.Ephemeral.ValueBool()
	}

	if !config.Running.IsNull() && !config.Running.IsUnknown() {
		running = config.Running.ValueBool()
	}

	// Ephemeral instance cannot be stopped.
	if ephemeral && !running {
		resp.Diagnostics.AddAttributeError(
			path.Root("running"),
			fmt.Sprintf("Instance %q is ephemeral and cannot be stopped", config.Name.ValueString()),
			fmt.Sprintf("Ephemeral instances are removed when stopped, therefore attribute %q must be set to %q.", "running", "true"),
		)
	}

	// Ensure image is set for container instances
	if config.Image.IsNull() && config.Type.ValueString() == "container" {
		resp.Diagnostics.AddAttributeError(
			path.Root("image"),
			fmt.Sprintf("Instance %q is a container and requires image", config.Name.ValueString()),
			fmt.Sprintf("Container instances require a rootfs (image), therefore attribute %q must be set.", "image"),
		)
	}
}

func (r InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set creation timeout.
	timeout, diags := plan.Timeouts.Create(ctx, r.provider.DefaultTimeout())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	var imageRemote string
	var imageServer lxd.ImageServer

	// Evaluate image remote.
	image := plan.Image.ValueString()
	imageParts := strings.SplitN(image, ":", 2)
	if len(imageParts) == 2 {
		imageRemote = imageParts[0]
		image = imageParts[1]
	}

	if imageRemote == "" {
		// Use the instance server as an image server if image remote is empty.
		imageServer = server
	} else {
		imageServer, err = r.provider.ImageServer(imageRemote)
		if err != nil {
			resp.Diagnostics.Append(errors.NewImageServerError(err))
			return
		}
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
		key := "limits." + k
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

	if image == "" {
		instance.Source.Type = api.SourceTypeNone
	} else if conn.Protocol == "simplestreams" {
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
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve image info for instance %q", instance.Name), err.Error())
			return
		}
	}

	// In case the image is set, create the instance from it, otherwise create it without rootfs. Similar to the --empty CLI flag on lxc.
	if image != "" {
		var opCreateFromImage lxd.RemoteOperation
		opCreateFromImage, err = server.CreateInstanceFromImage(imageServer, *imageInfo, instance)
		if err == nil {
			err = opCreateFromImage.Wait()
		}
	} else {
		var opCreate lxd.Operation
		opCreate, err = server.CreateInstance(instance)
		if err == nil {
			err = opCreate.Wait()
		}
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create instance %q", instance.Name), err.Error())
		return
	}

	// Partially update state to make Terraform aware of the created resource.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), instance.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Running.ValueBool() {
		// Start the instance.
		diag := startInstance(ctx, server, instance.Name, plan.WaitForOperational.ValueBool())
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}

		// Wait for the instance to obtain an IP address if network
		// availability is requested by the user.
		if plan.WaitForNetwork.ValueBool() {
			diag := waitInstanceNetwork(ctx, server, instance.Name)
			if diag != nil {
				resp.Diagnostics.Append(diag)
				return
			}
		}
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

	// Extract exec commands.
	execs, diags := common.ToExecMap(ctx, plan.Execs)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Execute commands.
	for _, k := range utils.SortMapKeys(execs) {
		e := execs[k]

		if plan.Running.ValueBool() && e.IsTriggered(true) {
			diags := e.Execute(ctx, server, instance.Name)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	// Update the plan with exec computed values.
	plan.Execs, diags = common.ToExecMapType(ctx, execs)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
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

	// Set read timeout.
	timeout, diags := state.Timeouts.Read(ctx, r.provider.DefaultTimeout())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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

// Update updates the instance in the following order:
// - Ensure instance state (stopped/running)
// - Update configuration (config, limits, devices, profiles)
// - Upload files
// - Run exec commands.
func (r InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceModel
	var state InstanceModel

	// Fetch resource model from Terraform plan.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set update timeout.
	timeout, diags := plan.Timeouts.Update(ctx, r.provider.DefaultTimeout())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := state.Name.ValueString()
	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return
	}

	// Get instance.
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", instanceName), err.Error())
		return
	}

	// Indicates if the instance has been just started.
	instanceStarted := false
	instanceRunning := isInstanceOperational(*instanceState)
	newInstanceName := plan.Name.ValueString()

	requireInstanceMigration := false
	requireInstanceRename := instanceName != newInstanceName

	// Compare current instance location against the desired location.
	if server.IsClustered() {
		onExpectedLocation, err := checkInstanceLocation(server, instance.Location, target)
		if err != nil {
			resp.Diagnostics.AddError("Failed to verify suitability of current instance location", err.Error())
			return
		}

		// Require instance migration if instance is misplaced.
		requireInstanceMigration = !onExpectedLocation
	}

	// If migration or instance rename is required, ensure the instance is stopped.
	if instanceRunning && (requireInstanceRename || requireInstanceMigration) {
		// If the instance is currently running and is not planned to be stopped,
		// we need to reject the update in case the provider is not allowed to
		// temporarily stop the instance. Otherwise, we could render the instance
		// unavailable without user's permission.
		if plan.Running.ValueBool() && !plan.AllowRestart.ValueBool() {
			resp.Diagnostics.AddError(
				"Instance stop not allowed",
				fmt.Sprintf(`The provider must temporarily stop the instance %q for migration or renaming, but stopping is not allowed. Either stop the instance manually or set the "allow_restart" attribute to "true".`, instanceName),
			)
			return
		}

		_, diag := stopInstance(ctx, server, instanceName, false)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}

		instanceRunning = false
	}

	// Handle instance rename.
	if requireInstanceRename {
		err := renameInstance(ctx, server, instanceName, newInstanceName)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to rename instance %q", instanceName), err.Error())
			return
		}

		// Use new instance name for further operations.
		instanceName = newInstanceName
	}

	// Handle instance migration.
	if requireInstanceMigration {
		err := migrateInstance(ctx, server, instanceName, target)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to migrate instance %q to %q", instanceName, target), err.Error())
			return
		}
	}

	// Ensure the instance is in desired state (stopped/running).
	if plan.Running.ValueBool() && !instanceRunning {
		instanceStarted = true
		instanceRunning = true

		diag := startInstance(ctx, server, instanceName, plan.WaitForOperational.ValueBool())
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}

		// If instance is freshly started, we should also wait for
		// network (if user requested that).
		if plan.WaitForNetwork.ValueBool() {
			diag := waitInstanceNetwork(ctx, server, instanceName)
			if diag != nil {
				resp.Diagnostics.Append(diag)
				return
			}
		}
	} else if !plan.Running.ValueBool() && !isInstanceStopped(*instanceState) {
		instanceRunning = false

		// Stop the instance gracefully.
		_, diag := stopInstance(ctx, server, instanceName, false)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}
	}

	// Get instance and its etag.
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
		key := "limits." + k
		config[key] = v
	}

	newInstance := api.InstancePut{
		Description:  plan.Description.ValueString(),
		Ephemeral:    plan.Ephemeral.ValueBool(),
		Architecture: instance.Architecture,
		Stateful:     instance.Stateful,
		Config:       config,
		Profiles:     profiles,
		Devices:      devices,
	}

	// Update the instance.
	opUpdate, err := server.UpdateInstance(instanceName, newInstance, etag)
	if err == nil {
		// Wait for the instance to be updated.
		err = opUpdate.Wait()
	}

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

		targetPath := f.TargetPath.ValueString()
		err := common.InstanceFileDelete(server, instanceName, targetPath)
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

	// Execute commands.
	oldExecs, diags := common.ToExecMap(ctx, state.Execs)
	resp.Diagnostics.Append(diags...)

	newExecs, diags := common.ToExecMap(ctx, plan.Execs)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Execute commands.
	for _, k := range utils.SortMapKeys(newExecs) {
		newExec := newExecs[k]
		oldExec := oldExecs[k]

		// Copy run count from state (if exists).
		if oldExec != nil {
			newExec.RunCount = oldExec.RunCount
		}

		if instanceRunning && newExec.IsTriggered(instanceStarted) {
			diags := newExec.Execute(ctx, server, instance.Name)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	plan.Execs, diags = common.ToExecMapType(ctx, newExecs)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
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

	// Set deletion timeout.
	timeout, diags := state.Timeouts.Delete(ctx, r.provider.DefaultTimeout())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := state.Name.ValueString()

	// Force stop the instance, because we are deleting it anyway.
	isFound, diag := stopInstance(ctx, server, instanceName, true)
	if diag != nil {
		// Ephemeral instances will be removed when stopped.
		if !isFound {
			return
		}

		resp.Diagnostics.Append(diag)
		return
	}

	// Delete the instance.
	opDelete, err := server.DeleteInstance(instanceName)
	if err == nil {
		// Wait for the instance to be deleted.
		err = opDelete.WaitContext(ctx)
	}

	if err != nil && !errors.IsNotFoundError(err) {
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

	// Reset IPv4, IPv6, and MAC addresses. In case instance has lost
	// network connectivity, we should reflect that in state.
	m.IPv4 = types.StringNull()
	m.IPv6 = types.StringNull()
	m.MAC = types.StringNull()

	accIface, ok := instance.ExpandedConfig["user.access_interface"]
	if ok {
		// If there is an user.access_interface set, extract IPv4, IPv6 and
		// MAC addresses from that network interface.
		net, ok := instanceState.Network[accIface]
		if ok {
			m.MAC = types.StringValue(net.Hwaddr)
			ipv4, ipv6 := findGlobalIPAddresses(net)

			if ipv4 != "" {
				m.IPv4 = types.StringValue(ipv4)
			}

			if ipv6 != "" {
				m.IPv6 = types.StringValue(ipv6)
			}
		}
	} else {
		// Search for the first interface (alphabetically sorted) that has
		// global IPv4 or IPv6 address.
		for _, iface := range utils.SortMapKeys(instanceState.Network) {
			if iface == "lo" {
				continue
			}

			net := instanceState.Network[iface]
			ipv4, ipv6 := findGlobalIPAddresses(net)
			if ipv4 != "" || ipv6 != "" {
				m.MAC = types.StringValue(net.Hwaddr)

				if ipv4 != "" {
					m.IPv4 = types.StringValue(ipv4)
				}

				if ipv6 != "" {
					m.IPv6 = types.StringValue(ipv6)
				}

				break
			}
		}
	}

	// Extract user defined config and merge it with current resource config.
	stateConfig := common.StripConfig(instance.Config, m.Config, m.ComputedKeys())

	// Extract enteries with "limits." prefix.
	instanceLimits := make(map[string]string)
	for k, v := range stateConfig {
		key, ok := strings.CutPrefix(k, "limits.")
		if ok {
			instanceLimits[key] = *v
			delete(stateConfig, k)
		}
	}

	// Convert config, limits, profiles, and devices into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	respDiags.Append(diags...)

	limits, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(instanceLimits), m.Config)
	respDiags.Append(diags...)

	profiles, diags := ToProfileListType(ctx, instance.Profiles)
	respDiags.Append(diags...)

	devices, diags := common.ToDeviceSetType(ctx, instance.Devices)
	respDiags.Append(diags...)

	interfaces, diags := common.ToInterfaceMapType(ctx, instanceState.Network, instance.Config)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return respDiags
	}

	m.Name = types.StringValue(instance.Name)
	m.Type = types.StringValue(instance.Type)
	m.Description = types.StringValue(instance.Description)
	m.Ephemeral = types.BoolValue(instance.Ephemeral)
	m.Status = types.StringValue(instance.Status)
	m.Profiles = profiles
	m.Limits = limits
	m.Devices = devices
	m.Interfaces = interfaces
	m.Config = config

	// Update "running" attribute based on the instance's current status.
	// This way, terraform will detect the change if the current status
	// does not match the expected one.
	m.Running = types.BoolValue(instanceState.Status == api.Running.String())

	m.Location = types.StringValue("")
	if server.IsClustered() || instance.Location != "none" {
		m.Location = types.StringValue(instance.Location)

		// Check if the instance is located on the configured cluster
		// member or within the configured cluster member group.
		ok, err := checkInstanceLocation(server, instance.Location, m.Target.ValueString())
		if err != nil {
			respDiags.AddError("Failed to check if instance is located on correct cluster member", err.Error())
			return respDiags
		}

		if !ok {
			// Trigger plan mismatch by setting target to an
			// actual instance location.
			m.Target = m.Location
		}
	}

	// Ensure default values are set for provider specific attributes
	// to prevent plan diff on import.
	if m.WaitForNetwork.IsNull() {
		m.WaitForNetwork = types.BoolValue(true)
	}

	if m.WaitForOperational.IsNull() {
		m.WaitForOperational = types.BoolValue(true)
	}

	if m.AllowRestart.IsNull() {
		m.AllowRestart = types.BoolValue(false)
	}

	return tfState.Set(ctx, &m)
}

// ComputedKeys returns list of computed config keys.
func (m InstanceModel) ComputedKeys() []string {
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

// startInstance starts an instance with the given name. It also waits
// for it to become fully operational.
func startInstance(ctx context.Context, server lxd.InstanceServer, instanceName string, checkOperational bool) diag.Diagnostic {
	st, etag, err := server.GetInstanceState(instanceName)
	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
	}

	// Return if the instance is already fully operational.
	if checkOperational && isInstanceOperational(*st) {
		return nil
	}

	startReq := api.InstanceStatePut{
		Action:  "start",
		Force:   false,
		Timeout: utils.ContextTimeout(ctx, 3*time.Minute),
	}

	// Start the instance.
	op, err := server.UpdateInstanceState(instanceName, startReq, etag)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to start instance %q", instanceName), err.Error())
	}

	instanceStartedCheck := func() (any, string, error) {
		st, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return st, "Error", err
		}

		// If instance is running, but not yet fully operationl, it
		// means that the instance is still initializing.
		if isInstanceRunning(*st) && checkOperational && !isInstanceOperational(*st) {
			return st, "Running (initializing)", nil
		}

		return st, st.Status, nil
	}

	// Even though op.Wait has completed, wait until we can see
	// the instance is fully started via a new API call.
	_, err = waitForState(ctx, instanceStartedCheck, api.Running.String())
	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to wait for instance %q to start", instanceName), err.Error())
	}

	return nil
}

// stopInstance stops an instance with the given name. It waits for its
// status to become Stopped or the instance to be removed (not found) in
// case of an ephemeral instance. In the latter case, false is returned
// along an error.
func stopInstance(ctx context.Context, server lxd.InstanceServer, instanceName string, force bool) (bool, diag.Diagnostic) {
	st, etag, err := server.GetInstanceState(instanceName)
	if err != nil {
		return true, diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
	}

	// Return if the instance is already stopped.
	if isInstanceStopped(*st) {
		return true, nil
	}

	stopReq := api.InstanceStatePut{
		Action:  "stop",
		Force:   force,
		Timeout: utils.ContextTimeout(ctx, 3*time.Minute),
	}

	// Stop the instance.
	op, err := server.UpdateInstanceState(instanceName, stopReq, etag)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		return true, diag.NewErrorDiagnostic(fmt.Sprintf("Failed to stop instance %q", instanceName), err.Error())
	}

	instanceStoppedCheck := func() (any, string, error) {
		st, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return st, "Error", err
		}

		return st, st.Status, nil
	}

	// Even though op.Wait has completed, wait until we can see
	// the instance is stopped via a new API call.
	_, err = waitForState(ctx, instanceStoppedCheck, api.Stopped.String())
	if err != nil {
		found := !errors.IsNotFoundError(err)
		return found, diag.NewErrorDiagnostic(fmt.Sprintf("Failed to wait for instance %q to stop", instanceName), err.Error())
	}

	return true, nil
}

// renameInstance renames an instance with the given old name to a new name.
// Instance has to be stopped beforehand, otherwise the operation will fail.
func renameInstance(ctx context.Context, server lxd.InstanceServer, oldName string, newName string) error {
	// Unset target to prevent LXD from assuming we are attempting migration
	// in case both instance name and target were changed.
	op, err := server.UseTarget("").RenameInstance(oldName, api.InstancePost{Name: newName})
	if err != nil {
		return err
	}

	return op.WaitContext(ctx)
}

// migrateInstance moves an instance to a different cluster member.
func migrateInstance(ctx context.Context, server lxd.InstanceServer, instanceName string, target string) error {
	// Migrate the instance to the desired location.
	req := api.InstancePost{
		Name:      instanceName,
		Migration: true,
		Live:      false, // We do not support live migration (yet).
	}

	op, err := server.UseTarget(target).MigrateInstance(instanceName, req)
	if err != nil {
		return err
	}

	return op.WaitContext(ctx)
}

// waitInstanceNetwork waits for an instance with the given name to receive
// an IPv4 address on any interface (excluding loopback). This should be
// called only if the instance is running.
func waitInstanceNetwork(ctx context.Context, server lxd.InstanceServer, instanceName string) diag.Diagnostic {
	// instanceNetworkCheck function checks whether instance has
	// received an IP address.
	instanceNetworkCheck := func() (any, string, error) {
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

		return st, "Waiting for network", nil
	}

	_, err := waitForState(ctx, instanceNetworkCheck, "OK")
	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to wait for instance %q to get an IP address", instanceName), err.Error())
	}

	return nil
}

// waitForState waits until the provided function reports one of the target
// states. It returns either the resulting state or an error.
func waitForState(ctx context.Context, refreshFunc retry.StateRefreshFunc, targets ...string) (any, error) {
	stateRefreshConf := &retry.StateChangeConf{
		Refresh:    refreshFunc,
		Target:     targets,
		Timeout:    3 * time.Minute,
		MinTimeout: 2 * time.Second, // Timeout increases: 2, 4, 8, 10, 10, ...
		Delay:      2 * time.Second, // Delay before the first check/refresh.
	}

	return stateRefreshConf.WaitForStateContext(ctx)
}

// isInstanceOperational determines if an instance is fully operational based
// on its state. It returns true if the instance is running and the reported
// process count is positive. Checking for a positive process count is essential
// for virtual machines, which can report this metric only if the LXD agent has
// started and has established a connection to the LXD server.
func isInstanceOperational(s api.InstanceState) bool {
	return isInstanceRunning(s) && s.Processes > 0
}

// isInstanceRunning returns true if its status is either "Running" or "Ready".
func isInstanceRunning(s api.InstanceState) bool {
	return s.StatusCode == api.Running || s.StatusCode == api.Ready
}

// isInstanceStopped returns true if instance's status "Stopped".
func isInstanceStopped(s api.InstanceState) bool {
	return s.StatusCode == api.Stopped
}

// findGlobalIPAddresses returns first global IPv4 and IPv6 addresses of the
// provided network interface. If an IP address is not found, an empty string
// is returned.
func findGlobalIPAddresses(network api.InstanceStateNetwork) (ipv4 string, ipv6 string) {
	for _, ip := range network.Addresses {
		if ip.Scope != "global" {
			continue
		}

		if ipv4 == "" && ip.Family == "inet" {
			ipv4 = ip.Address
		}

		if ipv6 == "" && ip.Family == "inet6" {
			ipv6 = ip.Address
		}
	}

	return ipv4, ipv6
}

// checkInstanceLocation checks whether the instance is located on the
// desired cluster member or within the desired cluster member group.
func checkInstanceLocation(server lxd.InstanceServer, location string, target string) (bool, error) {
	// If server is not clustered, there is only one option where
	// instance can be located. If the target matches the location,
	// the instance is already present on the desired cluster member.
	// Finally, if the server is clustered and target is empty, we do
	// not really care where the instance is located.
	if !server.IsClustered() || target == location || target == "" {
		return true, nil
	}

	// If target has prefix "@", we are dealing with the cluster
	// member group.
	targetGroup, ok := strings.CutPrefix(target, "@")
	if ok {
		group, _, err := server.GetClusterGroup(targetGroup)
		if err != nil {
			return false, err
		}

		// If the current cluster member (location) is part of the target
		// cluster group, then the instance is located on the correct
		// cluster member.
		if slices.Contains(group.Members, location) {
			return true, nil
		}
	}

	// The instance is not located on the desired cluster member/group.
	return false, nil
}
