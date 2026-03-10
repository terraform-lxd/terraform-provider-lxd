package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/auth"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/image"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/instance"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/network"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/profile"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/project"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/server"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/storage"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/truststore"
)

// LxdProviderModel represents the provider schema.
type LxdProviderModel struct {
	Address                      types.String `tfsdk:"address"`
	Protocol                     types.String `tfsdk:"protocol"`
	BearerToken                  types.String `tfsdk:"bearer_token"`
	BearerTokenFile              types.String `tfsdk:"bearer_token_file"`
	ClientKey                    types.String `tfsdk:"client_key"`
	ClientKeyFile                types.String `tfsdk:"client_key_file"`
	ClientCertificate            types.String `tfsdk:"client_certificate"`
	ClientCertificateFile        types.String `tfsdk:"client_certificate_file"`
	ServerCertificateFingerprint types.String `tfsdk:"server_certificate_fingerprint"`
}

// LxdProvider ...
type LxdProvider struct {
	version string
}

// NewLxdProvider returns LXD provider with the given version set.
func NewLxdProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LxdProvider{
			version: version,
		}
	}
}

func (p *LxdProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lxd"
	resp.Version = p.version
}

func (p *LxdProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				Optional:    true,
				Description: `Address of the LXD server. Supports "https://<host>:<port>" and "unix://<path>" schemes. Defaults to the local unix socket ("unix://").`,
			},

			"protocol": schema.StringAttribute{
				Optional:    true,
				Description: `Remote protocol. One of "lxd" (default) or "simplestreams".`,
				Validators: []validator.String{
					stringvalidator.OneOf("lxd", "simplestreams"),
				},
			},

			"bearer_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token for authentication. Mutually exclusive with client_certificate and load_from_lxc_config.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("bearer_token_file"),
						path.MatchRelative().AtParent().AtName("client_certificate"),
						path.MatchRelative().AtParent().AtName("client_certificate_file"),
						path.MatchRelative().AtParent().AtName("client_key"),
						path.MatchRelative().AtParent().AtName("client_key_file"),
					),
				},
			},

			"bearer_token_file": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Path to the file containing the bearer token for authentication.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("bearer_token"),
						path.MatchRelative().AtParent().AtName("client_certificate"),
						path.MatchRelative().AtParent().AtName("client_certificate_file"),
						path.MatchRelative().AtParent().AtName("client_key"),
						path.MatchRelative().AtParent().AtName("client_key_file"),
					),
				},
			},

			"client_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PEM-encoded private key for mTLS authentication.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("client_key_file"),
						path.MatchRelative().AtParent().AtName("bearer_token"),
						path.MatchRelative().AtParent().AtName("bearer_token_file"),
					),
				},
			},

			"client_key_file": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Path to the PEM-encoded private key for mTLS authentication.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("client_key"),
						path.MatchRelative().AtParent().AtName("bearer_token"),
						path.MatchRelative().AtParent().AtName("bearer_token_file"),
					),
				},
			},

			"client_certificate": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "PEM-encoded client certificate for mTLS authentication.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("client_certificate_file"),
						path.MatchRelative().AtParent().AtName("bearer_token"),
						path.MatchRelative().AtParent().AtName("bearer_token_file"),
					),
				},
			},

			"client_certificate_file": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Path to the PEM-encoded client certificate for mTLS authentication.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("client_certificate"),
						path.MatchRelative().AtParent().AtName("bearer_token"),
						path.MatchRelative().AtParent().AtName("bearer_token_file"),
					),
				},
			},

			"server_certificate_fingerprint": schema.StringAttribute{
				Optional:    true,
				Description: "SHA-256 fingerprint of the remote server's TLS certificate. Used to pin and verify the server certificate.",
			},
		},
	}
}

func (p *LxdProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data LxdProviderModel

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	protocol := data.Protocol.ValueString()

	// Client certificate and client key must both be provided.
	if (data.ClientCertificate.ValueString() != "" || data.ClientKey.ValueString() != "") && (data.ClientCertificate.ValueString() == "" || data.ClientKey.ValueString() == "") {
		resp.Diagnostics.AddError(
			"Invalid provider configuration",
			`Both "client_certificate" and "client_key" must be provided.`,
		)
		return
	}

	// Resolve address.
	address, err := provider_config.DetermineLXDAddress(protocol, data.Address.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider address", err.Error())
		return
	}

	// Parse bearer token.
	bearerToken := data.BearerToken.ValueString()

	if bearerToken == "" {
		bearerTokenFile := data.BearerTokenFile.ValueString()

		if data.BearerTokenFile.ValueString() != "" {
			content, err := os.ReadFile(bearerTokenFile)
			if err != nil {
				resp.Diagnostics.AddError("Failed to read bearer token file", err.Error())
				return
			}

			bearerToken = string(content)
		}
	}

	// Parse client certificate.
	clientCertificate := data.ClientCertificate.ValueString()
	if clientCertificate == "" {
		clientCertificateFile := data.ClientCertificateFile.ValueString()

		if clientCertificateFile != "" {
			content, err := os.ReadFile(clientCertificateFile)
			if err != nil {
				resp.Diagnostics.AddError("Failed to read client certificate file", err.Error())
				return
			}

			clientCertificate = string(content)
		}
	}

	// Parse client certificate.
	clientKey := data.ClientKey.ValueString()
	if clientKey == "" {
		clientKeyFile := data.ClientKeyFile.ValueString()

		if clientKeyFile != "" {
			content, err := os.ReadFile(clientKeyFile)
			if err != nil {
				resp.Diagnostics.AddError("Failed to read client key file", err.Error())
				return
			}

			clientKey = string(content)
		}
	}

	remote := provider_config.LxdRemote{
		Address:                      address,
		Protocol:                     protocol,
		BearerToken:                  bearerToken,
		ClientKey:                    clientKey,
		ClientCertificate:            clientCertificate,
		ServerCertificateFingerprint: data.ServerCertificateFingerprint.ValueString(),
	}

	lxdProvider, err := provider_config.NewLxdProviderConfig(p.version, remote)
	if err != nil {
		resp.Diagnostics.AddError("Failed to initialize LXD provider", err.Error())
		return
	}

	tflog.Debug(ctx, "LXD Provider configured", map[string]any{"provider": lxdProvider})

	resp.ResourceData = lxdProvider
	resp.DataSourceData = lxdProvider
}

func (p *LxdProvider) Resources(_ context.Context) []func() resource.Resource {
	resources := []func() resource.Resource{
		auth.NewAuthGroupResource,
		auth.NewAuthIdentityResource,
		image.NewCachedImageResource,
		image.NewPublishImageResource,
		instance.NewInstanceResource,
		instance.NewInstanceFileResource,
		instance.NewInstanceSnapshotResource,
		instance.NewInstanceDeviceResource,
		network.NewNetworkResource,
		network.NewNetworkAclResource,
		network.NewNetworkForwardResource,
		network.NewNetworkLBResource,
		network.NewNetworkPeerResource,
		network.NewNetworkZoneResource,
		network.NewNetworkZoneRecordResource,
		profile.NewProfileResource,
		project.NewProjectResource,
		storage.NewStorageBucketResource,
		storage.NewStorageBucketKeyResource,
		storage.NewStoragePoolResource,
		storage.NewStorageVolumeResource,
		storage.NewStorageVolumeCopyResource,
		truststore.NewTrustCertificateResource,
		truststore.NewTrustTokenResource,
	}

	// Resources for testing.
	if p.version == "test" {
		resources = append(resources, newNoopResource)
	}

	return resources
}

func (p *LxdProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		auth.NewAuthGroupDataSource,
		auth.NewAuthIdentityDataSource,
		image.NewImageDataSource,
		instance.NewInstanceDataSource,
		network.NewNetworkDataSource,
		profile.NewProfileDataSource,
		project.NewProjectDataSource,
		server.NewInfoDataSource,
		storage.NewStoragePoolDataSource,
	}
}
