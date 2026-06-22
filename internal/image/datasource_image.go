package image

import (
	"context"
	"fmt"
	"strings"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

type ImageDataSourceModel struct {
	Aliases      types.Set    `tfsdk:"aliases"`
	Architecture types.String `tfsdk:"architecture"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
	Image        types.String `tfsdk:"image"`
	Project      types.String `tfsdk:"project"`
	Type         types.String `tfsdk:"type"`
}

type ImageDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func NewImageDataSource() datasource.DataSource {
	return &ImageDataSource{}
}

func (d *ImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_image", req.ProviderTypeName)
}

func (d *ImageDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"image": schema.StringAttribute{
				Required:    true,
				Description: "Name or fingerprint of the image in the format `[<remote>:]<image>`. If the remote is omitted, the provider's default remote is used.",
			},

			"fingerprint": schema.StringAttribute{
				Computed: true,
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("container", "virtual-machine"),
				},
			},

			"architecture": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					architectureValidator{},
				},
			},

			"project": schema.StringAttribute{
				Optional: true,
			},

			"aliases": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"created_at": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *ImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.LxdProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	d.provider = provider
}

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ImageDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	imageRemote := ""
	identifier := state.Image.ValueString()

	imageParts := strings.SplitN(identifier, ":", 2)
	if len(imageParts) == 2 {
		imageRemote = imageParts[0]
		identifier = imageParts[1]
	}

	imageType := state.Type.ValueString()
	if imageType == "" {
		imageType = "container"
	}

	server, err := d.provider.ImageServer(imageRemote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewImageServerError(err))
		return
	}

	// Set project if we are dealing with instance server.
	instServer, ok := server.(lxd.InstanceServer)
	if ok {
		server = instServer.UseProject(state.Project.ValueString())
	}

	var fingerprint string

	architecture := state.Architecture.ValueString()
	if architecture != "" {
		// If an architecture is specified, look for an alias matching the identifier
		// and architecture.
		availableArchitectures, err := server.GetImageAliasArchitectures(imageType, identifier)
		if err == nil {
			// Architecture aliases found.
			// Find the alias matching the requested architecture.
			found := false
			for imageArchitecture, imageAlias := range availableArchitectures {
				if imageArchitecture == architecture {
					fingerprint = imageAlias.Target
					found = true
				}
			}

			if !found {
				keys := make([]string, 0, len(availableArchitectures))
				for key := range availableArchitectures {
					keys = append(keys, key)
				}

				keyList := strings.Join(keys, ", ")
				resp.Diagnostics.AddError(fmt.Sprintf("No image alias found for architecture %q. Available architectures: %s ", architecture, keyList), "")
				return
			}
		} else if !errors.IsNotFoundError(err) {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to get image alias architectures for image %q", identifier), err.Error())
			return
		}
	}

	if fingerprint == "" {
		// Fingerprint not determined from architecture-specific alias.
		// Look for a non-architecture-specific alias.
		var imageAlias *api.ImageAliasesEntry
		if state.Type.IsNull() {
			imageAlias, _, err = server.GetImageAlias(identifier)
		} else {
			imageType := state.Type.ValueString()
			imageAlias, _, err = server.GetImageAliasType(imageType, identifier)
		}

		if err == nil {
			fingerprint = imageAlias.Target
		} else if errors.IsNotFoundError(err) {
			// Not a known alias, treat the identifier as a fingerprint.
			fingerprint = identifier
		} else {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to get image alias %q", identifier), err.Error())
			return
		}
	}

	image, _, err := server.GetImage(fingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve image %q", identifier), err.Error())
		return
	}

	var aliases []string
	for _, a := range image.Aliases {
		aliases = append(aliases, a.Name)
	}

	aliasSet, diags := ToAliasSetType(ctx, aliases)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Aliases = aliasSet
	state.Architecture = types.StringValue(image.Architecture)
	state.CreatedAt = types.Int64Value(image.CreatedAt.Unix())
	state.Fingerprint = types.StringValue(image.Fingerprint)
	state.Type = types.StringValue(image.Type)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
