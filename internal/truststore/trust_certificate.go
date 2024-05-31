package truststore

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	lxd "github.com/canonical/lxd/client"
	lxd_shared "github.com/canonical/lxd/shared"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type TrustCertificateModel struct {
	Name     types.String `tfsdk:"name"`
	Path     types.String `tfsdk:"path"`
	Content  types.String `tfsdk:"content"`
	Projects types.List   `tfsdk:"projects"`
	Remote   types.String `tfsdk:"remote"`

	// Computed.
	Fingerprint types.String `tfsdk:"fingerprint"`
}

// TrustCertificateResource represent LXD trust certificate resource.
type TrustCertificateResource struct {
	provider *provider_config.LxdProviderConfig
}

// NewTrustCertificateResource returns a new trust certificate resource.
func NewTrustCertificateResource() resource.Resource {
	return &TrustCertificateResource{}
}

func (r TrustCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_trust_certificate", req.ProviderTypeName)
}

func (r TrustCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the certificate.",
			},

			"path": schema.StringAttribute{
				Optional:    true,
				Description: "Path to the client certificate.",
			},

			"content": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Content of the client certificate.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("content"),
						path.MatchRoot("path"),
					),
				},
			},

			"projects": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				Description: "List of projects to restrict the certificate to. By default, no restriction applies.",
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				ElementType: types.StringType,
			},

			"remote": schema.StringAttribute{
				Optional:    true,
				Description: "The remote in which the certificate is created. If not provided, the provider's default remote is used.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// Computed.
			"fingerprint": schema.StringAttribute{
				Computed:    true,
				Description: "Fingerprint of the certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *TrustCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TrustCertificateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var err error
	var plan *TrustCertificateModel

	// Ignore plan modification if plan is null (on destroy).
	if req.Plan.Raw.IsNull() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// We need to parse the certificate ahead of time, and evaluate it's fingerprint.
	// If fingerprint has changed, it will force recreation of the certificate.
	certName := plan.Name.ValueString()
	certPath := plan.Path.ValueString()
	certContent := []byte(plan.Content.ValueString())
	if certPath != "" {
		certContent, err = os.ReadFile(certPath)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to read certificate %q", certName),
				fmt.Sprintf("Read certificate on path %q: %v", certPath, err),
			)
			return
		}
	}

	// Parse the certificate.
	x509Cert, err := ParseCertX509(certContent)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to parse certificate %q", certName), err.Error())
		return
	}

	// Calculate certificate fingerprint.
	resp.Plan.SetAttribute(ctx, path.Root("fingerprint"), lxd_shared.CertFingerprint(x509Cert))
}

func (r TrustCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TrustCertificateModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Get list of project to restrict the certificate to.
	certProjects, diags := ToProjectList(ctx, plan.Projects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get certificate content.
	certName := plan.Name.ValueString()
	certPath := plan.Path.ValueString()
	certContent := []byte(plan.Content.ValueString())
	if certPath != "" {
		certContent, err = os.ReadFile(certPath)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Failed to read certificate %q", certName),
				fmt.Sprintf("Read certificate on path %q: %v", certPath, err),
			)
			return
		}
	}

	// Parse the certificate.
	x509Cert, err := ParseCertX509(certContent)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to parse certificate %q", certName), err.Error())
		return
	}

	// Calculate certificate fingerprint.
	plan.Fingerprint = types.StringValue(lxd_shared.CertFingerprint(x509Cert))

	// Create new certificate.
	cert := api.CertificatesPost{
		Type:        "client",
		Name:        certName,
		Projects:    certProjects,
		Restricted:  len(certProjects) > 0,
		Certificate: base64.StdEncoding.EncodeToString(x509Cert.Raw),
	}

	err = server.CreateCertificate(cert)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to add certificate %q to the trust store", certName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r TrustCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TrustCertificateModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r TrustCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TrustCertificateModel
	var state TrustCertificateModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certFingerprint := state.Fingerprint.ValueString()

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	certName := plan.Name.ValueString()
	cert, etag, err := server.GetCertificate(certFingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve certificate %q", certName), err.Error())
		return
	}

	certProjects, diags := ToProjectList(ctx, plan.Projects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update existing certificate.
	newCert := cert.Writable()
	newCert.Name = certName
	newCert.Projects = certProjects
	newCert.Restricted = len(certProjects) > 0

	err = server.UpdateCertificate(cert.Fingerprint, newCert, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update certificate %q", cert.Name), err.Error())
		return
	}

	plan.Fingerprint = state.Fingerprint

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r TrustCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TrustCertificateModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	fingerprint := state.Fingerprint.ValueString()
	err = server.DeleteCertificate(fingerprint)
	if err != nil {
		if errors.IsNotFoundError(err) {
			return
		}

		certName := state.Name.ValueString()
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove certificate %q", certName), err.Error())
	}
}

// SyncState fetches the server's current state for a certificate and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r TrustCertificateResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m TrustCertificateModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	certName := m.Name.ValueString()
	certFingerprint := m.Fingerprint.ValueString()
	cert, _, err := server.GetCertificate(certFingerprint)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve certificate %q", certName), err.Error())
		return respDiags
	}

	projects, diags := ToProjectListType(ctx, cert.Projects)
	if diags.HasError() {
		return diags
	}

	// Sync all fields except Certificate itself - we do not want to keep it
	// in Terraform state. If fingerprint changes, we will update the state.
	m.Name = types.StringValue(cert.Name)
	m.Fingerprint = types.StringValue(cert.Fingerprint)
	m.Projects = projects

	return tfState.Set(ctx, &m)
}

// ToProjectList converts projects from type types.List into []string.
func ToProjectList(ctx context.Context, projectList types.List) ([]string, diag.Diagnostics) {
	projects := make([]string, 0, len(projectList.Elements()))
	diags := projectList.ElementsAs(ctx, &projects, false)
	return projects, diags
}

// ToProjectListType converts projects from type []string into types.List.
func ToProjectListType[T string](ctx context.Context, projects []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, projects)
}

// ParseCertX509 decodes bytes into x509.Certificate.
func ParseCertX509(bytes []byte) (*x509.Certificate, error) {
	certBlock, _ := pem.Decode(bytes)
	if certBlock == nil {
		return nil, fmt.Errorf("Invalid certificate file")
	}

	return x509.ParseCertificate(certBlock.Bytes)
}
