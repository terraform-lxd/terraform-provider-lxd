package image

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type ImageDataSourceModel struct {
	Aliases      types.Set    `tfsdk:"aliases"`
	Architecture types.String `tfsdk:"architecture"`
	CreatedAt    types.Int64  `tfsdk:"created_at"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
	Name         types.String `tfsdk:"name"`
	Project      types.String `tfsdk:"project"`
	Remote       types.String `tfsdk:"remote"`
	Type         types.String `tfsdk:"type"`
}

type ImageDataSource struct {
	provider *provider_config.IncusProviderConfig
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
			"name": schema.StringAttribute{
				Optional: true,
			},

			"fingerprint": schema.StringAttribute{
				Optional: true,
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

			"aliases": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"project": schema.StringAttribute{
				Optional: true,
			},

			"remote": schema.StringAttribute{
				Optional: true,
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

	provider, ok := data.(*provider_config.IncusProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	d.provider = provider
}

func (d *ImageDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var state ImageDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Name.IsNull() && state.Fingerprint.IsNull() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Either name or fingerprint must be set.",
		)
		return
	}

	if !state.Name.IsNull() && !state.Fingerprint.IsNull() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Only name or fingerprint can be set.",
		)
		return
	}
}

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ImageDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := d.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewImageServerError(err))
		return
	}

	var fingerprint string
	if state.Fingerprint.IsNull() {
		imageName := state.Name.ValueString()
		architecture := state.Architecture.ValueString()

		if architecture != "" {
			imageType := state.Type.ValueString()
			availableArchitectures, err := server.GetImageAliasArchitectures(imageType, imageName)
			if err != nil {
				resp.Diagnostics.AddError("Failed to get image alias architectures", err.Error())
				return
			}

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

				resp.Diagnostics.AddError(fmt.Sprintf("No image alias found for architecture: %s. Available architectures: %s ", architecture, keyList), "")
				return
			}
		} else {
			var imageAlias *api.ImageAliasesEntry
			if state.Type.IsNull() {
				imageAlias, _, err = server.GetImageAlias(imageName)
				if err != nil {
					resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve image by alias %q", imageName), err.Error())
					return
				}
			} else {
				imageType := state.Type.ValueString()
				imageAlias, _, err = server.GetImageAliasType(imageType, imageName)
				if err != nil {
					resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve image by alias %q and type %q", imageName, imageType), err.Error())
					return
				}
			}

			fingerprint = imageAlias.Target
		}
	} else {
		fingerprint = state.Fingerprint.ValueString()
	}

	image, _, err := server.GetImage(fingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve image by fingerprint %q", fingerprint), err.Error())
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
