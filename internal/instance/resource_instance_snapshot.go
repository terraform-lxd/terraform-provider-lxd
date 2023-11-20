package instance

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type LxdInstanceSnapshotResourceModel struct {
	Name     types.String `tfsdk:"name"`
	Instance types.String `tfsdk:"instance"`
	Stateful types.Bool   `tfsdk:"stateful"`
	Project  types.String `tfsdk:"project"`
	Remote   types.String `tfsdk:"remote"`

	// Computed.
	CreatedAt types.Int64 `tfsdk:"created_at"`
}

// LxdInstanceSnapshotResource represent LXD instance snapshot resource.
type LxdInstanceSnapshotResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewLxdInstanceSnapshotResource returns a new instance snapshot resource.
func NewLxdInstanceSnapshotResource() resource.Resource {
	return &LxdInstanceSnapshotResource{}
}

func (r LxdInstanceSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_snapshot", req.ProviderTypeName)
}

func (r LxdInstanceSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"instance": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"stateful": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
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

			// Computed.

			"created_at": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *LxdInstanceSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r LxdInstanceSnapshotResource) Setup(_ context.Context, data LxdInstanceSnapshotResourceModel) (lxd.InstanceServer, diag.Diagnostic) {
	server, err := r.provider.InstanceServer(data.Remote.ValueString())
	if err != nil {
		return nil, errors.NewInstanceServerError(err)
	}

	project := data.Project.ValueString()
	if project != "" {
		server = server.UseProject(project)
	}

	return server, nil
}

func (r LxdInstanceSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LxdInstanceSnapshotResourceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, data)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	instanceName := data.Instance.ValueString()
	snapshotName := data.Instance.ValueString()

	snapshotReq := api.InstanceSnapshotsPost{
		Name:     snapshotName,
		Stateful: data.Stateful.ValueBool(),
	}

	var i int
	var serr error
	for i = 0; i < 5; i++ {
		op, err := server.CreateInstanceSnapshot(instanceName, snapshotReq)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create snapshot %q for instance %q", snapshotName, instanceName), serr.Error())
			return
		}

		// Wait for snapshot operation to complete
		serr = op.Wait()
		if serr != nil {
			if snapshotReq.Stateful && strings.Contains(err.Error(), "Dumping FAILED") {
				log.Printf("[DEBUG] error creating stateful snapshot [%d]: %v", i, err)
				time.Sleep(3 * time.Second)
			} else if strings.Contains(err.Error(), "file has vanished") {
				// Ignore, try again.
				time.Sleep(3 * time.Second)
			} else {
				break
			}
		} else {
			break
		}
	}

	if serr != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create snapshot %q for instance %q", snapshotName, instanceName), serr.Error())
		return
	}

	_, diags = data.SyncState(ctx, server)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Update Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r LxdInstanceSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LxdInstanceSnapshotResourceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, data)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	found, diags := data.SyncState(ctx, server)
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

func (r LxdInstanceSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r LxdInstanceSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LxdInstanceSnapshotResourceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	server, diag := r.Setup(ctx, data)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	instanceName := data.Instance.ValueString()
	snapshotName := data.Name.ValueString()
	op, err := server.DeleteInstanceSnapshot(instanceName, snapshotName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove snapshot %q for instance %q", snapshotName, instanceName), err.Error())
		return
	}

	err = op.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove snapshot %q for instance %q", snapshotName, instanceName), err.Error())
	}
}

// SyncState pulls instance snapshot data from the server and updates the model
// in-place. It returns a boolean indicating whether resource is found and
// diagnostics that contain potential errors.
// This should be called before updating Terraform state.
func (m *LxdInstanceSnapshotResourceModel) SyncState(ctx context.Context, server lxd.InstanceServer) (bool, diag.Diagnostics) {
	instanceName := m.Instance.ValueString()
	snapshotName := m.Name.ValueString()

	snap, _, err := server.GetInstanceSnapshot(instanceName, snapshotName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return false, nil
		}

		return true, diag.Diagnostics{diag.NewErrorDiagnostic(
			fmt.Sprintf("Failed to retrieve snapshot %q for instance %q", snapshotName, instanceName),
			err.Error(),
		)}
	}

	m.Stateful = types.BoolValue(snap.Stateful)
	m.CreatedAt = types.Int64Value(snap.CreatedAt.Unix())

	return true, nil
}
