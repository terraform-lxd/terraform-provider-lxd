package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// InstanceFileModel
//
// This model should embed common.LxdFileMode, but terraform-framework does
// not yet support unmarshaling of embedded structs.
// https://github.com/hashicorp/terraform-plugin-framework/issues/242
type InstanceFileModel struct {
	ResourceID types.String `tfsdk:"resource_id"` // Computed.
	Instance   types.String `tfsdk:"instance"`
	Project    types.String `tfsdk:"project"`
	Remote     types.String `tfsdk:"remote"`

	// common.InstanceFileModel
	Content    types.String `tfsdk:"content"`
	Source     types.String `tfsdk:"source"`
	TargetFile types.String `tfsdk:"target_file"`
	UserID     types.Int64  `tfsdk:"uid"`
	GroupID    types.Int64  `tfsdk:"gid"`
	Mode       types.String `tfsdk:"mode"`
	CreateDirs types.Bool   `tfsdk:"create_directories"`
	Append     types.Bool   `tfsdk:"append"`
}

// InstanceFileResource represent LXD instance file resource.
type InstanceFileResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewInstanceFileResource returns a new instance file resource.
func NewInstanceFileResource() resource.Resource {
	return &InstanceFileResource{}
}

func (r InstanceFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_instance_file", req.ProviderTypeName)
}

func (r InstanceFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"instance": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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

			"content": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"source": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					// Specify all attributes at one field to
					// produce only one meaningful error.
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("source"),
						path.MatchRoot("content"),
					),
				},
			},

			"target_file": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"uid": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"gid": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("0775"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"create_directories": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"append": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *InstanceFileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r InstanceFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceFileModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := plan.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Ensure instance exists.
	instanceName := plan.Instance.ValueString()
	_, _, err = server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed retireve instance %q", instanceName), err.Error())
		return
	}

	file := common.InstanceFileModel{
		Content:    plan.Content,
		Source:     plan.Source,
		TargetFile: plan.TargetFile,
		UserID:     plan.UserID,
		GroupID:    plan.GroupID,
		Mode:       plan.Mode,
		CreateDirs: plan.CreateDirs,
		Append:     plan.Append,
	}

	// Upload file.
	targetFile := plan.TargetFile.ValueString()
	err = common.InstanceFileUpload(server, instanceName, file)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create file %q on instance %q", targetFile, instanceName), err.Error())
		return
	}

	fileID := createFileResourceID(remote, instanceName, targetFile)
	plan.ResourceID = types.StringValue(fileID)

	// Update Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceFileModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, instanceName, targetFile := splitFileResourceID(state.ResourceID.ValueString())

	server, err := r.provider.InstanceServer(remote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := state.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Ensure instance exists.
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return
		}

		resp.Diagnostics.AddError(fmt.Sprintf("Failed retireve instance %q", instance.Name), err.Error())
		return
	}

	// Fetch file
	_, file, err := server.GetInstanceFile(instanceName, targetFile)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve file %q from instance %q", targetFile, instanceName), err.Error())
	}

	state.Instance = types.StringValue(instanceName)
	state.TargetFile = types.StringValue(targetFile)
	state.UserID = types.Int64Value(file.UID)
	state.GroupID = types.Int64Value(file.GID)
	state.Mode = types.StringValue(fmt.Sprintf("%04o", file.Mode))

	// Update Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r InstanceFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceFileModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, instanceName, targetFile := splitFileResourceID(state.ResourceID.ValueString())

	server, err := r.provider.InstanceServer(remote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Set project if configured.
	project := state.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	// Ensure instance exists.
	_, _, err = server.GetInstance(instanceName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return
		}

		resp.Diagnostics.AddError(fmt.Sprintf("Failed retireve instance %q", instanceName), err.Error())
		return
	}

	file := common.InstanceFileModel{
		Content:    state.Content,
		Source:     state.Source,
		TargetFile: state.TargetFile,
		UserID:     state.UserID,
		GroupID:    state.GroupID,
		Mode:       state.Mode,
		CreateDirs: state.CreateDirs,
		Append:     state.Append,
	}

	// Delete file.
	err = common.InstanceFileUpload(server, instanceName, file)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file %q from instance %q", targetFile, instanceName), err.Error())
		return
	}
}

// createFileResourceID creates new file ID by concatenating remote, instnaceName, and
// targetFile using colon.
func createFileResourceID(remote string, instanceName string, targetFile string) string {
	return fmt.Sprintf("%s:%s:%s", remote, instanceName, targetFile)
}

// splitFileResourceID splits file ID into remote, intanceName, and targetFile strings.
func splitFileResourceID(id string) (string, string, string) {
	pieces := strings.SplitN(id, ":", 3)
	return pieces[0], pieces[1], pieces[2]
}
