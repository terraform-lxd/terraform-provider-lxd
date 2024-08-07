package truststore

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type TrustTokenModel struct {
	Name     types.String `tfsdk:"name"`
	Projects types.List   `tfsdk:"projects"`
	Remote   types.String `tfsdk:"remote"`
	Trigger  types.String `tfsdk:"trigger"`

	// Computed.
	Token       types.String `tfsdk:"token"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	OperationID types.String `tfsdk:"operation_id"`
}

// TrustTokenResource represent LXD trust token resource.
type TrustTokenResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewTrustTokenResource returns a new trust token resource.
func NewTrustTokenResource() resource.Resource {
	return &TrustTokenResource{}
}

func (r TrustTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_trust_token", req.ProviderTypeName)
}

func (r TrustTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"projects": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				Description: "List of projects to restrict the token to. By default, no restriction applies.",
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},

			"remote": schema.StringAttribute{
				Optional:    true,
				Description: "The remote in which the trust token is created. If not provided, the provider's default remote is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"trigger": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("once"),
				Description: "When to trigger the token generation. Possible values are once and always (if missing).",
				Validators: []validator.String{
					stringvalidator.OneOf("once", "always"),
				},
				PlanModifiers: []planmodifier.String{
					// If trigger is changed, replace the resource to
					// ensure that the update is never triggered.
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Computed.
			"token": schema.StringAttribute{
				Computed:    true,
				Description: "Generated trust token.",
			},

			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "Time when trust token will expire.",
			},

			// Operation ID is used to find the created token. We can not rely on
			// the token name, as there can be multiple tokens with the same name.
			"operation_id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *TrustTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r TrustTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TrustTokenModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "default", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	tokenName := plan.Name.ValueString()

	// Get list of project to restrict the token to.
	tokenProjects, diags := ToProjectList(ctx, plan.Projects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new token.
	tokenPost := api.CertificatesPost{
		Name:       tokenName,
		Type:       "client",
		Token:      true,
		Projects:   tokenProjects,
		Restricted: len(tokenProjects) > 0,
	}

	op, err := server.CreateCertificateToken(tokenPost)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create trust token %q", tokenName), err.Error())
		return
	}

	opAPI := op.Get()
	token, err := opAPI.ToCertificateAddToken()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to convert operation into trust token: %q", tokenName), err.Error())
		return
	}

	plan.Token = types.StringValue(token.String())
	plan.ExpiresAt = types.StringValue(token.ExpiresAt.Format("2006/01/02 15:04 MST"))
	plan.OperationID = types.StringValue(opAPI.ID)

	// Update Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r TrustTokenResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Not supported.
}

func (r TrustTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TrustTokenModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "default", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	opID := state.OperationID.ValueString()
	tokenName := state.Name.ValueString()

	_, token, err := getTrustToken(server, opID)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve trust token %q", tokenName), err.Error())
		return
	}

	// If token is found, update the token.
	//
	// Otherwise, decide based on the trigger value:
	// - always: Remove it from the state to trigger generation of a new token.
	// - once:   Leave the state as is to prevent changes in the Terraform plan.
	if token != nil {
		state.Name = types.StringValue(token.ClientName)
		state.Token = types.StringValue(token.String())
		state.ExpiresAt = types.StringValue(token.ExpiresAt.Format("2006/01/02 15:04 MST"))
	} else if state.Trigger.ValueString() == "always" {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r TrustTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TrustTokenModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "default", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	tokenName := state.Token.ValueString()
	opID := state.OperationID.ValueString()

	op, _, err := getTrustToken(server, opID)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve trust token %q", tokenName), err.Error())
		return
	}

	// Remove the operation if found. Otherwise, the token no longer exists.
	if op != nil {
		err = server.DeleteOperation(op.ID)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove trust token %q", tokenName), err.Error())
			return
		}
	}
}

// getTrustToken returns a trust token operation and parsed trust token, if found.
// If token operation is not found, no error is returned. Instead, nil operation
// and nil trust token are returned.
func getTrustToken(server lxd.InstanceServer, opID string) (*api.Operation, *api.CertificateAddToken, error) {
	// Get all operations.
	ops, err := server.GetOperations()
	if err != nil {
		return nil, nil, err
	}

	for _, op := range ops {
		if op.ID != opID {
			// Skip operations that do not match the given ID.
			continue
		}

		if op.Class != api.OperationClassToken {
			// Operation must be of type OperationClassToken.
			break
		}

		if op.StatusCode != api.Running {
			// Tokens are single use. If token is cancelled but not deleted yet
			// its not available.
			break
		}

		token, err := op.ToCertificateAddToken()
		if err != nil {
			// Operation is not a valid certificate add token operation.
			break
		}

		return &op, token, nil
	}

	return nil, nil, nil
}
