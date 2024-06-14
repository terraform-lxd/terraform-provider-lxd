package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

// NetworkZoneRecordModel resource data model that
// matches the schema.
type NetworkZoneRecordModel struct {
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	Enteries    types.Set    `tfsdk:"entry"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
}

// NetworkZoneRecordResource represent Incus network zone record resource.
type NetworkZoneRecordResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewNetworkZoneRecordResource returns a new network zone record resource.
func NewNetworkZoneRecordResource() resource.Resource {
	return &NetworkZoneRecordResource{}
}

func (r NetworkZoneRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_zone_record", req.ProviderTypeName)
}

func (r NetworkZoneRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"zone": schema.StringAttribute{
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
			},
		},

		Blocks: map[string]schema.Block{
			"entry": schema.SetNestedBlock{
				Description: "Network zone record entry",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Record entry type",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"A", "AAAA", "CNAME", "TXT",
								),
							},
						},

						"value": schema.StringAttribute{
							Required:    true,
							Description: "Record entry value",
						},

						"ttl": schema.Int64Attribute{
							Optional:    true,
							Description: "Record entry TTL",
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
					},
				},
			},
		},
	}
}

func (r *NetworkZoneRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r NetworkZoneRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkZoneRecordModel

	// Fetch resource model from Terraform plan.
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

	// Convert network zone record config and entries.
	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	entries, diags := ToZoneRecordEntryList(ctx, plan.Enteries)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := plan.Zone.ValueString()
	recordName := plan.Name.ValueString()
	recordReq := api.NetworkZoneRecordsPost{
		Name: recordName,
		NetworkZoneRecordPut: api.NetworkZoneRecordPut{
			Description: plan.Description.ValueString(),
			Config:      config,
			Entries:     entries,
		},
	}

	// Create network zone record.
	err = server.CreateNetworkZoneRecord(zoneName, recordReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network zone record %q", recordName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkZoneRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkZoneRecordModel

	// Fetch resource model from Terraform state.
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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkZoneRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkZoneRecordModel

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

	// Get existing network zone record.
	zoneName := plan.Zone.ValueString()
	recordName := plan.Name.ValueString()
	_, etag, err := server.GetNetworkZoneRecord(zoneName, recordName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network zone record %q", recordName), err.Error())
		return
	}

	// Convert network zone record config and entries.
	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	entries, diags := ToZoneRecordEntryList(ctx, plan.Enteries)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update network zone record.
	recordReq := api.NetworkZoneRecordPut{
		Description: plan.Description.ValueString(),
		Entries:     entries,
		Config:      config,
	}

	err = server.UpdateNetworkZoneRecord(zoneName, recordName, recordReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network zone record %q", zoneName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r NetworkZoneRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkZoneRecordModel

	// Fetch resource model from Terraform state.
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

	zoneName := state.Zone.ValueString()
	recordName := state.Name.ValueString()
	err = server.DeleteNetworkZoneRecord(zoneName, recordName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network zone record %q", recordName), err.Error())
	}
}

func (r NetworkZoneRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "network_zone_record",
		RequiredFields: []string{"zone", "name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// SyncState fetches the server's current state for a network zone record and
// updates the provided model. It then applies this updated model as the new
// state in Terraform.
func (r NetworkZoneRecordResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m NetworkZoneRecordModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	zoneName := m.Zone.ValueString()
	recordName := m.Name.ValueString()
	record, _, err := server.GetNetworkZoneRecord(zoneName, recordName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve network zone record %q", recordName), err.Error())
		return respDiags
	}

	entries, diags := ToZoneRecordEntrySetType(ctx, record.Entries)
	respDiags.Append(diags...)

	config, diags := common.ToConfigMapType(ctx, record.Config)
	respDiags.Append(diags...)

	m.Zone = types.StringValue(zoneName)
	m.Name = types.StringValue(record.Name)
	m.Description = types.StringValue(record.Description)
	m.Enteries = entries
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

type NetworkZoneRecordEntryModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
	TTL   types.Int64  `tfsdk:"ttl"`
}

// ToZoneRecordMap converts network zone record of type types.Map
// into []IncusNetworkZoneEntryModel.
func ToZoneRecordEntryList(ctx context.Context, entrySet types.Set) ([]api.NetworkZoneRecordEntry, diag.Diagnostics) {
	if entrySet.IsNull() || entrySet.IsUnknown() {
		return []api.NetworkZoneRecordEntry{}, nil
	}

	// Convert into intermediary struct (that has struct tags).
	entryList := make([]NetworkZoneRecordEntryModel, 0, len(entrySet.Elements()))
	diags := entrySet.ElementsAs(ctx, &entryList, false)
	if diags.HasError() {
		return nil, diags
	}

	// Convert into API network zone entries.
	entries := make([]api.NetworkZoneRecordEntry, 0, len(entryList))
	for _, e := range entryList {
		entry := api.NetworkZoneRecordEntry{
			Type:  e.Type.ValueString(),
			Value: e.Value.ValueString(),
			TTL:   uint64(e.TTL.ValueInt64()),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// ToZoneRecordEntrySetType converts list of network zone records into
// set of type types.Set.
func ToZoneRecordEntrySetType(ctx context.Context, entries []api.NetworkZoneRecordEntry) (types.Set, diag.Diagnostics) {
	entryList := make([]NetworkZoneRecordEntryModel, 0, len(entries))
	for _, e := range entries {
		entry := NetworkZoneRecordEntryModel{
			Type:  types.StringValue(e.Type),
			Value: types.StringValue(e.Value),
			TTL:   types.Int64Value(int64(e.TTL)),
		}
		entryList = append(entryList, entry)
	}

	entryType := map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
		"ttl":   types.Int64Type,
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: entryType}, entryList)
}
