package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkAclDataSourceModel resource data model that matches the schema.
type NetworkAclDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Description types.String `tfsdk:"description"`
	Config      types.Map    `tfsdk:"config"`
	Egress      types.Set    `tfsdk:"egress"`
	Ingress     types.Set    `tfsdk:"ingress"`
}

type NetworkAclDataSource struct {
	provider *provider_config.LxdProviderConfig
}

func NewNetworkAclDataSource() datasource.DataSource {
	return &NetworkAclDataSource{}
}

func (d *NetworkAclDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_acl", req.ProviderTypeName)
}

func (d *NetworkAclDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"project": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			"description": schema.StringAttribute{
				Computed: true,
			},

			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"egress": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: aclRuleAttributesComputed(),
				},
			},

			"ingress": schema.SetNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: aclRuleAttributesComputed(),
				},
			},
		},
	}
}

func (d *NetworkAclDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkAclDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state NetworkAclDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := d.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	aclName := state.Name.ValueString()
	acl, _, err := server.GetNetworkACL(aclName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve network ACL %q", aclName), err.Error())
		return
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(acl.Config), state.Config)
	resp.Diagnostics.Append(diags...)

	egress, diags := ToNetworkAclRulesSetType(acl.Egress)
	resp.Diagnostics.Append(diags...)

	ingress, diags := ToNetworkAclRulesSetType(acl.Ingress)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	state.Name = types.StringValue(acl.Name)
	state.Description = types.StringValue(acl.Description)
	state.Project = types.StringValue(acl.Project)
	state.Config = config
	state.Egress = egress
	state.Ingress = ingress

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// aclRuleAttributesComputed returns ACL rule attributes with all fields Computed,
// for use in data source schemas where no validators or defaults are needed.
func aclRuleAttributesComputed() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"action":           schema.StringAttribute{Computed: true},
		"destination":      schema.StringAttribute{Computed: true},
		"destination_port": schema.StringAttribute{Computed: true},
		"protocol":         schema.StringAttribute{Computed: true},
		"description":      schema.StringAttribute{Computed: true},
		"state":            schema.StringAttribute{Computed: true},
		"source":           schema.StringAttribute{Computed: true},
		"icmp_type":        schema.StringAttribute{Computed: true},
		"icmp_code":        schema.StringAttribute{Computed: true},
	}
}
