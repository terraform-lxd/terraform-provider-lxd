/*
 * This is a noop resource that is included only when running tests and
 * should be used exclusively for testing the LXD "provider" block.
 *
 * The resource is used to force loading of the provider's remote configuration,
 * as it is lazy-loaded.
 */
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type noopModel struct {
	Project        types.String `tfsdk:"project"`
	Remote         types.String `tfsdk:"remote"`
	AuthUserMethod types.String `tfsdk:"auth_user_method"`
	ServerVersion  types.String `tfsdk:"server_version"`
}

// noopResource represents noop resource used for testing.
type noopResource struct {
	provider *provider_config.LxdProviderConfig
}

// newNoopResource returns a new noop resource.
func newNoopResource() resource.Resource {
	return &noopResource{}
}

// Metadata for noop resource.
func (r noopResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_noop"
}

// Schema for noop resource.
func (r noopResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(provider_config.DefaultProject),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					// Force replace because resource does not implement "Update".
					stringplanmodifier.RequiresReplace(),
				},
			},

			"server_version": schema.StringAttribute{
				Computed: true,
			},

			"auth_user_method": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *noopResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r noopResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan noopModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.SyncState(ctx, &resp.State, plan)...)
}

func (r noopResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state noopModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.SyncState(ctx, &resp.State, state)...)
}

func (r noopResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Nothing to do. All fields trigger a replace.
}

func (r noopResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do. Just remove the resource from the state.
}

func (r noopResource) SyncState(ctx context.Context, tfState *tfsdk.State, m noopModel) diag.Diagnostics {
	remote := m.Remote.ValueString()
	project := m.Project.ValueString()

	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		return diag.Diagnostics{errors.NewInstanceServerError(err)}
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve the API server for remote %q", remote), err.Error())}
	}

	m.Project = types.StringValue(apiServer.Environment.Project)
	m.ServerVersion = types.StringValue(apiServer.Environment.ServerVersion)
	m.AuthUserMethod = types.StringValue(apiServer.AuthUserMethod)

	return tfState.Set(ctx, &m)
}
