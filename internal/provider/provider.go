package provider

import (
	"context"
	"log"
	"os"
	"path/filepath"

	lxd_config "github.com/canonical/lxd/lxc/config"
	lxd_shared "github.com/canonical/lxd/shared"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
						},

						"address": schema.StringAttribute{
							Optional:    true,
							Description: "The FQDN or IP where the LXD daemon can be contacted. (default = \"\" (read from lxc config))",
						},

						"port": schema.StringAttribute{
							Optional:    true,
							Description: "Port LXD Daemon API is listening on. (default = 8443)",
						},

						"scheme": schema.StringAttribute{
							Optional:    true,
							Description: "Unix (unix) or HTTPs (https). (default = unix)",
							Validators: []validator.String{
								stringvalidator.OneOf("unix", "https"),
							},
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

	// Determine LXD configuration directory. First check for the presence
	// of the /var/snap/lxd directory. If the directory exists, return
	// snap's config path. Otherwise return the fallback path.
	configDir := data.ConfigDir.ValueString()
	if configDir == "" {
		_, err := os.Stat("/var/snap/lxd")
		if err == nil || os.IsExist(err) {
			configDir = "$HOME/snap/lxd/common/config"
		} else {
			configDir = "$HOME/.config/lxc"
		}
	}

	configDir = os.ExpandEnv(configDir)

	// Try to load config.yml from determined configDir. If there's
	// an error loading config.yml, default config will be used.
	configPath := filepath.Join(configDir, "config.yml")
	config, err := lxd_config.LoadConfig(configPath)
	if err != nil {
		config = lxd_config.DefaultConfig()
		config.ConfigDir = configDir
	}

	log.Printf("[DEBUG] LXD Config: %#v", config)

	// Determine if the LXD server's SSL certificates should be
	// accepted. If this is set to false and if the remote's
	// certificates haven't already been accepted, the user will
	// need to accept the certificates out of band of Terraform.
	acceptServerCertificate := data.AcceptRemoteCertificate.ValueBool()
	if data.AcceptRemoteCertificate.IsNull() || data.AcceptRemoteCertificate.IsUnknown() {
		v, ok := os.LookupEnv("LXD_ACCEPT_SERVER_CERTIFICATE")
		if ok {
			acceptServerCertificate = lxd_shared.IsTrue(v)
		}
	}

	// Determine if the client LXD (ie: the workstation running Terraform)
	// should generate client certificates if they don't already exist.
	generateClientCertificates := data.GenerateClientCertificates.ValueBool()
	if data.AcceptRemoteCertificate.IsNull() || data.GenerateClientCertificates.IsUnknown() {
		v, ok := os.LookupEnv("LXD_GENERATE_CLIENT_CERTS")
		if ok {
			generateClientCertificates = lxd_shared.IsTrue(v)
		}
	}

	if generateClientCertificates {
		err := config.GenerateClientCertificate()
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate client certificate", err.Error())
			return
		}
	}

	// Initialize global LxdProvider struct.
	// This struct is used to store information about this Terraform
	// provider's configuration for reference throughout the lifecycle.
	lxdProvider := provider_config.NewLxdProvider(config, acceptServerCertificate)

	// Create LXD remote from environment variables (if defined).
	// This emulates the Terraform provider "remote" config:
	//
	// remote {
	//   name    = LXD_REMOTE
	//   address = LXD_ADDR
	//   ...
	// }
	envName := os.Getenv("LXD_REMOTE")
	if envName != "" {
		envRemote := provider_config.LxdProviderRemoteConfig{
			Name:     envName,
			Address:  os.Getenv("LXD_ADDR"),
			Port:     os.Getenv("LXD_PORT"),
			Password: os.Getenv("LXD_PASSWORD"),
			Token:    os.Getenv("LXD_TOKEN"),
			Scheme:   os.Getenv("LXD_SCHEME"),
			Protocol: "lxd",
		}

		// This will be the default remote unless overridden by an
		// explicitly defined remote in the Terraform configuration.
		lxdProvider.SetRemote(envRemote, true)
	}

	// Loop over LXD Remotes defined in the schema and create
	// an lxdRemoteConfig for each one.
	//
	// This does not yet connect to any of the defined remotes,
	// it only stores the configuration information until it is
	// necessary to connect to the remote.
	//
	// This lazy loading allows this LXD provider to be used
	// in Terraform configurations where the LXD remote might not
	// exist yet.
	for _, remote := range data.Remotes {
		protocol := remote.Protocol.ValueString()
		if protocol == "" {
			protocol = "lxd"
		}

		port := remote.Port.ValueString()
		if port == "" {
			port = "8443"
			if protocol == "simplestreams" {
				port = "443"
			}
		}

		scheme := remote.Scheme.ValueString()
		if scheme == "" {
			scheme = "unix"
			if protocol == "simplestreams" {
				scheme = "https"
			}
		}

		lxdRemote := provider_config.LxdProviderRemoteConfig{
			Name:     remote.Name.ValueString(),
			Password: remote.Password.ValueString(),
			Token:    remote.Token.ValueString(),
			Address:  remote.Address.ValueString(),
			Protocol: protocol,
			Port:     port,
			Scheme:   scheme,
		}

		isDefault := remote.Default.ValueBool()
		if protocol == "simplestreams" {
			// Simplestreams cannot be default.
			isDefault = false
		}

		lxdProvider.SetRemote(lxdRemote, isDefault)
	}

	log.Printf("[DEBUG] LXD Provider: %#v", &lxdProvider)

	resp.ResourceData = lxdProvider
	resp.DataSourceData = lxdProvider
}

func (p *LxdProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		image.NewCachedImageResource,
		image.NewPublishImageResource,
		instance.NewInstanceResource,
		instance.NewInstanceFileResource,
		instance.NewInstanceSnapshotResource,
		network.NewNetworkResource,
		network.NewNetworkAclResource,
		network.NewNetworkForwardResource,
		network.NewNetworkLBResource,
		network.NewNetworkZoneResource,
		network.NewNetworkZoneRecordResource,
		profile.NewProfileResource,
		project.NewProjectResource,
		storage.NewStoragePoolResource,
		storage.NewStorageVolumeResource,
		storage.NewStorageVolumeCopyResource,
		truststore.NewTrustCertificateResource,
	}
}

func (p *LxdProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
