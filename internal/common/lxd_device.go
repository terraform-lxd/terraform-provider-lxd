package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type LxdDeviceModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Properties types.Map    `tfsdk:"properties"`
}

// ToDeviceMap converts deviecs from types.Set into map[string]map[string]string.
func ToDeviceMap(ctx context.Context, dataDevices types.Set) (map[string]map[string]string, diag.Diagnostics) {
	if dataDevices.IsNull() || dataDevices.IsUnknown() {
		return make(map[string]map[string]string), nil
	}

	// Convert types.Set into set of device models.
	modelDevices := make([]LxdDeviceModel, 0, len(dataDevices.Elements()))
	diags := dataDevices.ElementsAs(ctx, &modelDevices, false)
	if diags.HasError() {
		return nil, diags
	}

	devices := make(map[string]map[string]string, len(modelDevices))
	for _, d := range modelDevices {
		devName := d.Name.ValueString()
		devType := d.Type.ValueString()

		// Convert properties into map[string]string.
		device := make(map[string]string, len(d.Properties.Elements()))
		if !d.Properties.IsNull() && !d.Properties.IsUnknown() {
			diags := d.Properties.ElementsAs(ctx, &device, false)
			if diags.HasError() {
				return nil, diags
			}
		}

		device["type"] = devType
		devices[devName] = device
	}

	return devices, nil
}

// ToDeviceSetType converts deviecs from map[string]map[string]string into types.Set.
func ToDeviceSetType(ctx context.Context, devices map[string]map[string]string) (types.Set, diag.Diagnostics) {
	devModelTypes := map[string]attr.Type{
		"name":       types.StringType,
		"type":       types.StringType,
		"properties": types.MapType{ElemType: types.StringType},
	}

	nilSet := types.SetNull(types.ObjectType{AttrTypes: devModelTypes})

	if len(devices) == 0 {
		return nilSet, nil
	}

	modelDevices := make([]LxdDeviceModel, 0, len(devices))
	for key := range devices {
		props := devices[key]

		devName := types.StringValue(key)
		devType := types.StringValue(props["type"])

		// Remove type from properties, as we manage it
		// outside properties.
		delete(props, "type")

		devProps, diags := types.MapValueFrom(ctx, types.StringType, props)
		if diags.HasError() {
			return nilSet, diags
		}

		dev := LxdDeviceModel{
			Name:       devName,
			Type:       devType,
			Properties: devProps,
		}

		modelDevices = append(modelDevices, dev)
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: devModelTypes}, modelDevices)
}
