package instance

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// InstanceDeviceModel represents a single device attached to an LXD instance.
type InstanceDeviceModel struct {
	Name         types.String `tfsdk:"name"`
	InstanceName types.String `tfsdk:"instance_name"`
	Project      types.String `tfsdk:"project"`
	Remote       types.String `tfsdk:"remote"`
	Target       types.String `tfsdk:"target"`
	Type         types.String `tfsdk:"type"`
	Properties   types.Map    `tfsdk:"properties"`
}

// InstanceDeviceResource represents a device attachable to LXD instance.
type InstanceDeviceResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewInstanceDeviceResource returns a new instance device resource.
func NewInstanceDeviceResource() resource.Resource {
	return &InstanceDeviceResource{}
}

// Metadata for the device resource.
func (r InstanceDeviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_device"
}

// Schema for the device resource.
func (r InstanceDeviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Device name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"instance_name": schema.StringAttribute{
				Required:    true,
				Description: "Instance name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"project": schema.StringAttribute{
				Optional:    true,
				Description: "Project",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional:    true,
				Description: "Remote",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"target": schema.StringAttribute{
				Optional:    true,
				Description: "Target",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
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
					mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
		},
	}
}

func (r *InstanceDeviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.LxdProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
	}

	r.provider = provider
}

func (r *InstanceDeviceResource) ModifyPlan(_ context.Context, _ resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	resp.Diagnostics.AddWarning(
		"lxd_instance_device is experimental",
		"lxd_instance_device resource is an experimental feature of Terraform LXD Provider and it may change in the future.",
	)
}

func (r InstanceDeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceDeviceModel

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

	props, diags := common.ToConfigMap(ctx, plan.Properties)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, etag, err := server.GetInstance(plan.InstanceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", plan.InstanceName.ValueString()), err.Error())
		return
	}

	deviceName := plan.Name.ValueString()
	deviceType := plan.Type.ValueString()
	props["type"] = deviceType

	// Mark the device as managed by terraform to differentiate between
	// devices added by terraform and devices added manually.
	props[common.UserManagedBy] = common.DeviceManagedByTerraform

	if instance.Devices == nil {
		instance.Devices = make(map[string]map[string]string)
	}

	_, deviceExists := instance.Devices[deviceName]
	if deviceExists {
		msg := fmt.Sprintf("Device %q on instance %q already exists", deviceName, instance.Name)
		resp.Diagnostics.AddError("Device already exists", msg)
		return
	}

	// Modify devices map to add the provided device.
	instance.Devices[deviceName] = props

	updatedInstance := instance.Writable()
	op, err := server.UpdateInstance(instance.Name, updatedInstance, etag)
	if err == nil {
		// Wait for the instance to be updated.
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update instance %q", instance.Name), err.Error())
		return
	}

	// Sync state after successfully attaching the device.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceDeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceDeviceModel

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

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceDeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceDeviceModel

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

	props, diags := common.ToConfigMap(ctx, plan.Properties)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, etag, err := server.GetInstance(plan.InstanceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", plan.InstanceName.ValueString()), err.Error())
		return
	}

	deviceName := plan.Name.ValueString()
	deviceType := plan.Type.ValueString()
	props["type"] = deviceType

	// Mark the device as managed by terraform to differentiate between
	// devices added by terraform and devices added manually.
	props[common.UserManagedBy] = common.DeviceManagedByTerraform

	if instance.Devices == nil {
		msg := fmt.Sprintf("Instance %q has no devices", instance.Name)
		resp.Diagnostics.AddError("Instance has no devices", msg)
		return
	}

	oldDevice, deviceExists := instance.Devices[deviceName]
	if !deviceExists {
		msg := fmt.Sprintf("Device %q on instance %q not found", deviceName, instance.Name)
		resp.Diagnostics.AddError("Device not found", msg)
		return
	}

	if oldDevice[common.UserManagedBy] != common.DeviceManagedByTerraform {
		msg := fmt.Sprintf("Device %q on instance %q is not managed by Terraform", deviceName, instance.Name)
		resp.Diagnostics.AddError("Cannot update non-managed device", msg)
		return
	}

	// Modify devices map to add the provided device.
	instance.Devices[deviceName] = props

	updatedInstance := instance.Writable()
	op, err := server.UpdateInstance(instance.Name, updatedInstance, etag)
	if err == nil {
		// Wait for the instance to be updated.
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update instance %q", instance.Name), err.Error())
		return
	}

	// Sync state after successfully updating the device properties.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceDeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceDeviceModel

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

	instanceName := state.InstanceName.ValueString()
	deviceName := state.Name.ValueString()

	instance, etag, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", state.InstanceName.ValueString()), err.Error())
		return
	}

	// Skip the update if the device not found.
	oldDevice, deviceExists := instance.Devices[deviceName]
	if !deviceExists {
		return
	}

	if oldDevice[common.UserManagedBy] != common.DeviceManagedByTerraform {
		msg := fmt.Sprintf("Device %q on instance %q is not managed by Terraform", deviceName, instance.Name)
		resp.Diagnostics.AddError("Cannot delete non-managed device", msg)
		return
	}

	// Modify devices map to remove specified device.
	delete(instance.Devices, deviceName)

	updatedInstance := instance.Writable()
	op, err := server.UpdateInstance(instance.Name, updatedInstance, etag)
	if err == nil {
		// Wait for the instance to be updated.
		err = op.WaitContext(ctx)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update instance %q", instance.Name), err.Error())
		return
	}

	// Sync state after successfully removing the device.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

// SyncState fetches the server's current state for the device and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r InstanceDeviceResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m InstanceDeviceModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	instanceName := m.InstanceName.ValueString()
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve instance %q", instanceName), err.Error())
		return respDiags
	}

	deviceName := m.Name.ValueString()
	deviceProps, ok := instance.Devices[deviceName]
	if !ok || deviceProps == nil {
		tfState.RemoveResource(ctx)
		return nil
	}

	deviceType, ok := deviceProps["type"]
	if !ok || deviceType == "" {
		respDiags.AddError(
			"Device is missing type",
			fmt.Sprintf("Device %q for instance %q is missing type field", deviceName, instanceName))
		return respDiags
	}

	// Delete to avoid duplication, "type" value is stored separately in DeviceModel.
	delete(deviceProps, "type")

	// Delete "user.managed-by" key from internal state to avoid config mismatch.
	delete(deviceProps, common.UserManagedBy)

	properties, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(deviceProps), m.Properties)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return respDiags
	}

	m.Type = types.StringValue(deviceType)
	m.Properties = properties

	return tfState.Set(ctx, &m)
}
