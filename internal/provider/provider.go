package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

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

// LxdProviderRemoteModel represents provider's schema remote.
type LxdProviderRemoteModel struct {
	Name                         types.String `tfsdk:"name"`
	Address                      types.String `tfsdk:"address"`
	Protocol                     types.String `tfsdk:"protocol"`
	TrustToken                   types.String `tfsdk:"trust_token"`
	BearerToken                  types.String `tfsdk:"bearer_token"`
	BearerTokenFile              types.String `tfsdk:"bearer_token_file"`
	ClientKey                    types.String `tfsdk:"client_key"`
	ClientKeyFile                types.String `tfsdk:"client_key_file"`
	ClientCertificate            types.String `tfsdk:"client_certificate"`
	ClientCertificateFile        types.String `tfsdk:"client_certificate_file"`
	ServerCertificateFingerprint types.String `tfsdk:"server_certificate_fingerprint"`
}

// LxdProviderModel represents provider's schema.
type LxdProviderModel struct {
	Remotes       []LxdProviderRemoteModel `tfsdk:"remote"`
	DefaultRemote types.String             `tfsdk:"default_remote"`
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
			"default_remote": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the default LXD remote to use when no remote is specified in the resource. If two or more remotes are defined, one must be set as the default.",
			},
		},

		Blocks: map[string]schema.Block{
			"remote": schema.ListNestedBlock{
				Description: "LXD Remote",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the LXD remote.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},

						"address": schema.StringAttribute{
							Required:    true,
							Description: "Address of the LXD or SimpleStreams remote.",
						},

						"protocol": schema.StringAttribute{
							Optional:    true,
							Description: "Remote protocol",
							Validators: []validator.String{
								stringvalidator.OneOf("lxd", "simplestreams"),
							},
						},

						"trust_token": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "The trust token used for initial authentication with the LXD remote.",
							Validators: []validator.String{
								stringvalidator.ConflictsWith(
									path.MatchRelative().AtParent().AtName("bearer_token"),
									path.MatchRelative().AtParent().AtName("bearer_token_file"),
								),
							},
						},

						"bearer_token": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "Bearer token for authentication.",
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
				},
			},
		},
	}
}

func (p *LxdProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data LxdProviderModel

	// Read provider schema into model.
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	remotes := make(map[string]provider_config.LxdRemote)
	defRemote := data.DefaultRemote.ValueString()

	// Read remotes from Terraform schema.
	for _, remote := range data.Remotes {
		name := remote.Name.ValueString()

		protocol := remote.Protocol.ValueString()
		if protocol == "" {
			protocol = "lxd"
		}

		address, err := provider_config.DetermineLXDAddress(protocol, remote.Address.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Invalid remote %q", name), err.Error())
			return
		}

		// Parse bearer token.
		bearerToken := remote.BearerToken.ValueString()
		if bearerToken == "" {
			bearerTokenFile := remote.BearerTokenFile.ValueString()

			if remote.BearerTokenFile.ValueString() != "" {
				content, err := os.ReadFile(bearerTokenFile)
				if err != nil {
					resp.Diagnostics.AddError("Failed to read bearer token file", err.Error())
					return
				}

				bearerToken = strings.TrimSpace(string(content))
			}
		}

		// Parse client certificate.
		clientCertificate := remote.ClientCertificate.ValueString()
		if clientCertificate == "" {
			clientCertificateFile := remote.ClientCertificateFile.ValueString()

			if clientCertificateFile != "" {
				content, err := os.ReadFile(clientCertificateFile)
				if err != nil {
					resp.Diagnostics.AddError("Failed to read client certificate file", err.Error())
					return
				}

				clientCertificate = string(content)
			}
		}

		// Parse client key.
		clientKey := remote.ClientKey.ValueString()
		if clientKey == "" {
			clientKeyFile := remote.ClientKeyFile.ValueString()

			if clientKeyFile != "" {
				content, err := os.ReadFile(clientKeyFile)
				if err != nil {
					resp.Diagnostics.AddError("Failed to read client key file", err.Error())
					return
				}

				clientKey = string(content)
			}
		}

		if (clientCertificate != "" || clientKey != "") && (clientCertificate == "" || clientKey == "") {
			resp.Diagnostics.AddError(fmt.Sprintf("Client certificate and key must be provided for remote %q", name), "Both client certificate and client key must be provided for TLS authentication.")
			return
		}

		remotes[name] = provider_config.LxdRemote{
			Address:                      address,
			Protocol:                     protocol,
			TrustToken:                   remote.TrustToken.ValueString(),
			BearerToken:                  bearerToken,
			ClientKey:                    clientKey,
			ClientCertificate:            clientCertificate,
			ServerCertificateFingerprint: remote.ServerCertificateFingerprint.ValueString(),
		}
	}

	// Initialize LXD provider configuration.
	lxdProvider, err := provider_config.NewLxdProviderConfig(p.version, remotes, defRemote)
	if err != nil {
		resp.Diagnostics.AddError("Failed initialize LXD provider", err.Error())
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
