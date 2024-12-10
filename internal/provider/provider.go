package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/canonical/lxd/shared"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/image"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/instance"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/network"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/profile"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/project"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/storage"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/truststore"
)

// LxdProviderRemoteModel represents provider's schema remote.
type LxdProviderRemoteModel struct {
	Name     types.String `tfsdk:"name"`
	Address  types.String `tfsdk:"address"`
	Port     types.String `tfsdk:"port"`
	Protocol types.String `tfsdk:"protocol"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
	Scheme   types.String `tfsdk:"scheme"`
	Default  types.Bool   `tfsdk:"default"`
}

// LxdProviderModel represents provider's schema.
type LxdProviderModel struct {
	Remotes                    []LxdProviderRemoteModel `tfsdk:"remote"`
	ConfigDir                  types.String             `tfsdk:"config_dir"`
	AcceptRemoteCertificate    types.Bool               `tfsdk:"accept_remote_certificate"`
	GenerateClientCertificates types.Bool               `tfsdk:"generate_client_certificates"`
}

// LxdProvider ...
type LxdProvider struct {
	version string
}

// New returns LXD provider with the given version set.
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
			"config_dir": schema.StringAttribute{
				Optional:    true,
				Description: "The directory to look for existing LXD configuration. (default = $HOME/snap/lxd/common/config:$HOME/.config/lxc)",
			},

			"generate_client_certificates": schema.BoolAttribute{
				Optional:    true,
				Description: "Automatically generate the LXD client certificates if they don't exist.",
			},

			"accept_remote_certificate": schema.BoolAttribute{
				Optional:    true,
				Description: "Accept the server certificate.",
			},
		},

		Blocks: map[string]schema.Block{
			"remote": schema.ListNestedBlock{
				Description: "LXD Remote",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the LXD remote. Required when lxd_scheme set to https, to enable locating server certificate.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},

						"address": schema.StringAttribute{
							Optional:    true,
							Description: "The FQDN or IP where the LXD daemon can be contacted. (default = \"\")",
						},

						// Deprecated, leave the attribute in so that we error out if it's used.
						// DeprecationMessage would just print the warning, but we want to error
						// out with a custom message.
						"port": schema.StringAttribute{
							Optional:    true,
							Description: "Port LXD Daemon API is listening on. (default = 8443)",
						},

						// Deprecated, leave the attribute in so that we error out if it's used.
						// DeprecationMessage would just print the warning, but we want to error
						// out with a custom message.
						"scheme": schema.StringAttribute{
							Optional:    true,
							Description: "Unix (unix) or HTTPs (https). (default = unix)",
						},

						"password": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "The trust password used for initial authentication with the LXD remote.",
						},

						"token": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "The trust token used for initial authentication with the LXD remote.",
							Validators: []validator.String{
								// Mutually exclusive with "password".
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("password")),
							},
						},

						"protocol": schema.StringAttribute{
							Optional:    true,
							Description: "Remote protocol",
							Validators: []validator.String{
								stringvalidator.OneOf("lxd", "simplestreams"),
							},
						},

						"default": schema.BoolAttribute{
							Optional:    true,
							Description: "Set this remote as default.",
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

	// Determine if the LXD server's SSL certificates should be accepted.
	// If this is set to false and if the remote's certificates haven't
	// already been accepted, the user will need to accept the certificates
	// out of band of Terraform.
	acceptServerCertificate := data.AcceptRemoteCertificate.ValueBool()
	if data.AcceptRemoteCertificate.IsNull() || data.AcceptRemoteCertificate.IsUnknown() {
		v, ok := os.LookupEnv("LXD_ACCEPT_SERVER_CERTIFICATE")
		if ok {
			acceptServerCertificate = shared.IsTrue(v)
		}
	}

	// Determine if the missing client certificates should be generated.
	// This has no effect if the certificates already exist.
	generateClientCertificates := data.GenerateClientCertificates.ValueBool()
	if data.GenerateClientCertificates.IsNull() || data.GenerateClientCertificates.IsUnknown() {
		v, ok := os.LookupEnv("LXD_GENERATE_CLIENT_CERTS")
		if ok {
			generateClientCertificates = shared.IsTrue(v)
		}
	}

	remotes := make(map[string]provider_config.LxdRemote)

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

		// Error out if deprecated port and scheme attributes are used.
		if remote.Port.ValueString() != "" {
			resp.Diagnostics.AddError(
				fmt.Sprintf(`Remote %q contains deprecated attribute "port"`, name),
				fmt.Sprintf(`Please remove the attribute "port" and set "address" to the fully qualified address instead. For example, "address=%s".`, address),
			)
			return
		}

		if remote.Scheme.ValueString() != "" {
			resp.Diagnostics.AddError(
				fmt.Sprintf(`Remote %q contains deprecated attribute "scheme"`, name),
				fmt.Sprintf(`Please remove the attribute "port" and set "address" to the fully qualified address instead. For example, "address=%s".`, address),
			)
			return
		}

		remotes[name] = provider_config.LxdRemote{
			Address:   address,
			Protocol:  protocol,
			Password:  remote.Password.ValueString(),
			Token:     remote.Token.ValueString(),
			IsDefault: remote.Default.ValueBool(),
		}
	}

	options := provider_config.Options{
		ConfigDir:                  data.ConfigDir.ValueString(),
		AcceptServerCertificate:    acceptServerCertificate,
		GenerateClientCertificates: generateClientCertificates,
	}

	// Initialize LXD provider configuration.
	lxdProvider, err := provider_config.NewLxdProviderConfig(p.version, remotes, options)
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
		image.NewCachedImageResource,
		image.NewPublishImageResource,
		instance.NewInstanceResource,
		instance.NewInstanceFileResource,
		instance.NewInstanceSnapshotResource,
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
	return nil
}
