package config

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	localtls "github.com/lxc/incus/v6/shared/tls"

	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type CertificateModel struct {
	Certificate types.String `tfsdk:"certificate"`
	Description types.String `tfsdk:"description"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	Name        types.String `tfsdk:"name"`
	Projects    types.Set    `tfsdk:"projects"`
	Remote      types.String `tfsdk:"remote"`
	Restricted  types.Bool   `tfsdk:"restricted"`
	Type        types.String `tfsdk:"type"`
}

type CertificateResource struct {
	provider *provider_config.IncusProviderConfig
}

func NewCertificateResource() resource.Resource {
	return &CertificateResource{}
}

func (r *CertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_certificate", req.ProviderTypeName)
}

func (r *CertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"certificate": schema.StringAttribute{
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

			"projects": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					// Prevent empty values.
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"restricted": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("client"),
				Validators: []validator.String{
					stringvalidator.OneOf("client", "metrics"),
				},
			},

			// Computed attributes

			"fingerprint": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *CertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.IncusProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	r.provider = provider
}

func (r *CertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CertificateModel

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

	projects, diags := toProjectList(ctx, plan.Projects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	certificateStr := plan.Certificate.ValueString()
	x509Cert, err := parseCertificate(certificateStr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse certificate", err.Error())
		return
	}

	certificate := api.CertificatesPost{
		CertificatePut: api.CertificatePut{
			Name:        plan.Name.ValueString(),
			Type:        plan.Type.ValueString(),
			Restricted:  plan.Restricted.ValueBool(),
			Projects:    projects,
			Certificate: base64.StdEncoding.EncodeToString(x509Cert.Raw),
			Description: plan.Description.ValueString(),
		},
		Token: false,
	}

	err = server.CreateCertificate(certificate)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create certificate %q", certificate.Name), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *CertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CertificateModel

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

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *CertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CertificateModel
	var state CertificateModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
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

	fingerprint := state.Fingerprint.ValueString()
	certificate, etag, err := server.GetCertificate(fingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve certificate %q with fingerprint %q", plan.Name.ValueString(), fingerprint), err.Error())
		return
	}

	projects, diags := toProjectList(ctx, plan.Projects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedCertificate := api.CertificatePut{
		Name:        plan.Name.ValueString(),
		Type:        plan.Type.ValueString(),
		Restricted:  plan.Restricted.ValueBool(),
		Projects:    projects,
		Certificate: certificate.Certificate,
		Description: plan.Description.ValueString(),
	}

	err = server.UpdateCertificate(fingerprint, updatedCertificate, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update certificate %q with fingerprint %q", certificate.Name, fingerprint), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *CertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CertificateModel

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
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove certificate %q with fingerprint %q", state.Name.ValueString(), fingerprint), err.Error())
	}
}

// SyncState fetches the server's current state for a certificate and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r *CertificateResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m CertificateModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	fingerprint, diags := getFingerprint(m)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	certificate, _, err := server.GetCertificate(fingerprint)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve certificate %q with fingerprint %q", m.Name.ValueString(), fingerprint), err.Error())
		return respDiags
	}

	projects, diags := toProjectType(ctx, certificate.Projects)
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return respDiags
	}

	m.Name = types.StringValue(certificate.Name)
	m.Fingerprint = types.StringValue(certificate.Fingerprint)
	m.Type = types.StringValue(certificate.Type)
	m.Restricted = types.BoolValue(certificate.Restricted)
	m.Projects = projects
	m.Certificate = types.StringValue(certificate.Certificate)
	m.Description = types.StringValue(certificate.Description)

	return tfState.Set(ctx, &m)
}

func toProjectList(ctx context.Context, projectSet types.Set) ([]string, diag.Diagnostics) {
	if projectSet.IsNull() || projectSet.IsUnknown() {
		return []string{}, nil
	}

	projects := make([]string, 0, len(projectSet.Elements()))
	diags := projectSet.ElementsAs(ctx, &projects, false)
	return projects, diags
}

func toProjectType(ctx context.Context, projects []string) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, types.StringType, projects)
}

func parseCertificate(certificate string) (*x509.Certificate, error) {
	certBlock, _ := pem.Decode([]byte(certificate))
	if certBlock == nil {
		return nil, fmt.Errorf("Invalid certificate file")
	}

	return x509.ParseCertificate(certBlock.Bytes)
}

func getFingerprint(m CertificateModel) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if !m.Fingerprint.IsNull() && !m.Fingerprint.IsUnknown() {
		return m.Fingerprint.ValueString(), diags
	}

	certificateStr := m.Certificate.ValueString()
	x509Cert, err := parseCertificate(certificateStr)
	if err != nil {
		diags.AddError("Failed to parse certificate", err.Error())
		return "", diags
	}

	return localtls.CertFingerprint(x509Cert), diags
}
