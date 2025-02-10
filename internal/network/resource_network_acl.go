package network

import (
	"context"
	"fmt"

	lxd "github.com/canonical/lxd/client"
	"github.com/canonical/lxd/shared/api"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/common"
	"github.com/terraform-lxd/terraform-provider-lxd/internal/errors"
	provider_config "github.com/terraform-lxd/terraform-provider-lxd/internal/provider-config"
)

// NetworkAclModel resource data model that matches the schema.
type NetworkAclModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
	Egress      types.Set    `tfsdk:"egress"`
	Ingress     types.Set    `tfsdk:"ingress"`
}

// NetworkAclRuleModel resource data model that matches the schema.
type NetworkAclRuleModel struct {
	Action          types.String `tfsdk:"action"`
	Destination     types.String `tfsdk:"destination"`
	DestinationPort types.String `tfsdk:"destination_port"`
	Protocol        types.String `tfsdk:"protocol"`
	Description     types.String `tfsdk:"description"`
	State           types.String `tfsdk:"state"`
	Source          types.String `tfsdk:"source"`
	ICMPType        types.String `tfsdk:"icmp_type"`
	ICMPCode        types.String `tfsdk:"icmp_code"`
}

// NetworkAclResource represent LXD network ACL resource.
type NetworkAclResource struct {
	provider *provider_config.LxdProviderConfig
}

func NewNetworkAclResource() resource.Resource {
	return &NetworkAclResource{}
}

func (r *NetworkAclResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_acl"
}

func (r *NetworkAclResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	aclRuleObjectType := aclRuleObjectType()

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"project": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},

			"egress": schema.SetNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  setdefault.StaticValue(types.SetNull(aclRuleObjectType)),
				NestedObject: schema.NestedAttributeObject{
					Attributes: aclRuleAttributes(),
				},
			},

			"ingress": schema.SetNestedAttribute{
				Optional: true,
				Computed: true,
				Default:  setdefault.StaticValue(types.SetNull(aclRuleObjectType)),
				NestedObject: schema.NestedAttributeObject{
					Attributes: aclRuleAttributes(),
				},
			},
		},
	}
}

func aclRuleObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"action":           types.StringType,
			"destination":      types.StringType,
			"destination_port": types.StringType,
			"protocol":         types.StringType,
			"description":      types.StringType,
			"state":            types.StringType,
			"source":           types.StringType,
			"icmp_type":        types.StringType,
			"icmp_code":        types.StringType,
		},
	}
}

func aclRuleAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"action": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.OneOf("allow", "allow-stateless", "drop", "reject"),
			},
		},

		"destination": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},

		"destination_port": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},

		"protocol": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},

		"description": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},

		"state": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.OneOf("enabled", "disabled", "logged"),
			},
		},

		"source": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},

		"icmp_type": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},

		"icmp_code": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},
	}
}

func (r *NetworkAclResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkAclResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkAclModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	egress, diags := ToNetworkAclRules(ctx, plan.Egress)
	resp.Diagnostics.Append(diags...)

	ingress, diags := ToNetworkAclRules(ctx, plan.Ingress)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	aclName := plan.Name.ValueString()
	aclReq := api.NetworkACLsPost{
		NetworkACLPost: api.NetworkACLPost{
			Name: aclName,
		},
		NetworkACLPut: api.NetworkACLPut{
			Description: plan.Description.ValueString(),
			Config:      config,
			Egress:      egress,
			Ingress:     ingress,
		},
	}

	err = server.CreateNetworkACL(aclReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network ACL %q", aclName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkAclResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkAclModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkAclResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkAclModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	aclName := plan.Name.ValueString()
	_, etag, err := server.GetNetworkACL(aclName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network ACL %q", aclName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	egress, diags := ToNetworkAclRules(ctx, plan.Egress)
	resp.Diagnostics.Append(diags...)

	ingress, diags := ToNetworkAclRules(ctx, plan.Ingress)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	aclReq := api.NetworkACLPut{
		Description: plan.Description.ValueString(),
		Config:      config,
		Egress:      egress,
		Ingress:     ingress,
	}

	err = server.UpdateNetworkACL(aclName, aclReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network ACL %q", aclName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkAclResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkAclModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	aclName := state.Name.ValueString()
	err = server.DeleteNetworkACL(aclName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network ACL %q", aclName), err.Error())
	}
}

func (r *NetworkAclResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "network_acl",
		RequiredFields: []string{"name"},
	}

	fields, diags := meta.ParseImportID(req.ID)
	if diags != nil {
		resp.Diagnostics.Append(diags)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

func (r *NetworkAclResource) SyncState(ctx context.Context, tfState *tfsdk.State, server lxd.InstanceServer, m NetworkAclModel) diag.Diagnostics {
	aclName := m.Name.ValueString()
	acl, _, err := server.GetNetworkACL(aclName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		return diag.Diagnostics{diag.NewErrorDiagnostic(
			fmt.Sprintf("Failed to retrieve network ACL %q", aclName), err.Error(),
		)}
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(acl.Config), m.Config)
	if diags.HasError() {
		return diags
	}

	egress, diags := ToNetworkAclRulesSetType(acl.Egress)
	if diags.HasError() {
		return diags
	}

	ingress, diags := ToNetworkAclRulesSetType(acl.Ingress)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(acl.Name)
	m.Description = types.StringValue(acl.Description)
	m.Config = config
	m.Egress = egress
	m.Ingress = ingress

	return tfState.Set(ctx, &m)
}

// ToNetworkAclRules converts ACL rules from type types.Set into []api.NetworkACLRule.
func ToNetworkAclRules(ctx context.Context, aclRuleList types.Set) ([]api.NetworkACLRule, diag.Diagnostics) {
	if aclRuleList.IsNull() {
		return []api.NetworkACLRule{}, nil
	}

	aclRuleModelList := make([]NetworkAclRuleModel, 0, len(aclRuleList.Elements()))
	diags := aclRuleList.ElementsAs(ctx, &aclRuleModelList, false)
	if diags.HasError() {
		return nil, diags
	}

	aclRules := make([]api.NetworkACLRule, len(aclRuleModelList))
	for i, aclRuleModel := range aclRuleModelList {
		protocol := aclRuleModel.Protocol.ValueString()

		aclRule := api.NetworkACLRule{
			Action:          aclRuleModel.Action.ValueString(),
			Destination:     aclRuleModel.Destination.ValueString(),
			DestinationPort: aclRuleModel.DestinationPort.ValueString(),
			Protocol:        protocol,
			Description:     aclRuleModel.Description.ValueString(),
			State:           aclRuleModel.State.ValueString(),
			Source:          aclRuleModel.Source.ValueString(),
		}

		if protocol == "icmp4" || protocol == "icmp6" {
			aclRule.ICMPType = aclRuleModel.ICMPType.ValueString()
			aclRule.ICMPCode = aclRuleModel.ICMPCode.ValueString()
		}

		aclRules[i] = aclRule
	}

	return aclRules, nil
}

// ToNetworkAclRulesSetType converts []api.NetworkACLRule into acl rules of type types.Set.
func ToNetworkAclRulesSetType(networkACLRules []api.NetworkACLRule) (types.Set, diag.Diagnostics) {
	aclRuleObjectType := aclRuleObjectType()
	nilSet := types.SetNull(aclRuleObjectType)

	if len(networkACLRules) == 0 {
		return nilSet, nil
	}

	aclRuleList := make([]attr.Value, 0, len(networkACLRules))
	for _, rule := range networkACLRules {
		// Create the attribute map for each rule
		aclRuleMap := map[string]attr.Value{
			"action":           types.StringValue(rule.Action),
			"destination":      types.StringValue(rule.Destination),
			"destination_port": types.StringValue(rule.DestinationPort),
			"protocol":         types.StringValue(rule.Protocol),
			"description":      types.StringValue(rule.Description),
			"state":            types.StringValue(rule.State),
			"source":           types.StringValue(rule.Source),
			"icmp_type":        types.StringValue(rule.ICMPType),
			"icmp_code":        types.StringValue(rule.ICMPCode),
		}

		aclRuleObject, diags := types.ObjectValue(aclRuleObjectType.AttrTypes, aclRuleMap)
		if diags.HasError() {
			return nilSet, diags
		}

		aclRuleList = append(aclRuleList, aclRuleObject)
	}

	return types.SetValue(aclRuleObjectType, aclRuleList)
}
