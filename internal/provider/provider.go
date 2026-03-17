package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
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
	Name       types.String `tfsdk:"name"`
	Address    types.String `tfsdk:"address"`
	Protocol   types.String `tfsdk:"protocol"`
	TrustToken types.String `tfsdk:"trust_token"`
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
							Description: "Name of the LXD remote. Required when lxd_scheme set to https, to enable locating server certificate.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},

						"address": schema.StringAttribute{
							Optional:    true,
							Description: "The FQDN or IP where the LXD daemon can be contacted. (default = \"\")",
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

		remotes[name] = provider_config.LxdRemote{
			Address:    address,
			Protocol:   protocol,
			TrustToken: remote.TrustToken.ValueString(),
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
