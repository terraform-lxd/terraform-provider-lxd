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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type InstanceSnapshotModel struct {
	Name     types.String `tfsdk:"name"`
	Instance types.String `tfsdk:"instance"`
	Stateful types.Bool   `tfsdk:"stateful"`
	Project  types.String `tfsdk:"project"`
	Remote   types.String `tfsdk:"remote"`

	// Computed.
	CreatedAt types.Int64 `tfsdk:"created_at"`
}

// InstanceSnapshotResource represent LXD instance snapshot resource.
type InstanceSnapshotResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewInstanceSnapshotResource returns a new instance snapshot resource.
func NewInstanceSnapshotResource() resource.Resource {
	return &InstanceSnapshotResource{}
}

func (r InstanceSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_snapshot", req.ProviderTypeName)
}

func (r InstanceSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *InstanceSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r InstanceSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceSnapshotModel

	// Fetch resource model from Terraform plan.
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

	instanceName := plan.Instance.ValueString()
	snapshotName := plan.Name.ValueString()

	snapshotReq := api.InstanceSnapshotsPost{
		Name:     snapshotName,
		Stateful: plan.Stateful.ValueBool(),
	}

	var serr error
	for i := 0; i < 5; i++ {
		op, err := server.CreateInstanceSnapshot(instanceName, snapshotReq)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create snapshot %q for instance %q", snapshotName, instanceName), serr.Error())
			return
		}

		// Wait for snapshot operation to complete.
		serr = op.Wait()
		if serr == nil {
			break
		}

		if snapshotReq.Stateful && strings.Contains(serr.Error(), "Dumping FAILED") {
			log.Printf("[DEBUG] Error creating stateful snapshot [retry %d]: %v", i, serr)
			time.Sleep(3 * time.Second)
		} else if strings.Contains(serr.Error(), "file has vanished") {
			// Ignore, try again.
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	if serr != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create snapshot %q for instance %q", snapshotName, instanceName), serr.Error())
		return
	}

	// Partially update state to make Terraform aware of the created resource.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), snapshotName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance"), instanceName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("remote"), remote)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceSnapshotModel

	// Fetch resource model from Terraform state.
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

func (r InstanceSnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r InstanceSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceSnapshotModel

	// Fetch resource model from Terraform state.
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

	instanceName := state.Instance.ValueString()
	snapshotName := state.Name.ValueString()
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

// SyncState fetches the server's current state for an instance snapshot and
// updates the provided model. It then applies this updated model as the new
// state in Terraform.
func (r InstanceSnapshotResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m InstanceSnapshotModel) diag.Diagnostics {
	instanceName := m.Instance.ValueString()
	snapshotName := m.Name.ValueString()
	snapshot, _, err := server.GetInstanceSnapshot(instanceName, snapshotName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		return diag.Diagnostics{diag.NewErrorDiagnostic(
			fmt.Sprintf("Failed to retrieve snapshot %q for instance %q", snapshotName, instanceName),
			err.Error(),
		)}
	}

	m.Stateful = types.BoolValue(snapshot.Stateful)
	m.CreatedAt = types.Int64Value(snapshot.CreatedAt.Unix())

	return tfState.Set(ctx, &m)
}
