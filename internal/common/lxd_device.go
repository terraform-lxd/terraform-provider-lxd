package common

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type DeviceModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Properties types.Map    `tfsdk:"properties"`
}

// ToDeviceMap converts deviecs from types.Set into map[string]map[string]string.
func ToDeviceMap(ctx context.Context, deviceSet types.Set) (map[string]map[string]string, diag.Diagnostics) {
	if deviceSet.IsNull() || deviceSet.IsUnknown() {
		return make(map[string]map[string]string), nil
	}

	deviceList := make([]DeviceModel, 0, len(deviceSet.Elements()))
	diags := deviceSet.ElementsAs(ctx, &deviceList, false)
	if diags.HasError() {
		return nil, diags
	}

	devices := make(map[string]map[string]string, len(deviceList))
	for _, d := range deviceList {
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

// ToDeviceSetType converts deviecs from map[string]map[string]string
// into types.Set.
func ToDeviceSetType(ctx context.Context, devices map[string]map[string]string) (types.Set, diag.Diagnostics) {
	deviceType := map[string]attr.Type{
		"name":       types.StringType,
		"type":       types.StringType,
		"properties": types.MapType{ElemType: types.StringType},
	}

	nilSet := types.SetNull(types.ObjectType{AttrTypes: deviceType})

	if len(devices) == 0 {
		return nilSet, nil
	}

	deviceList := make([]DeviceModel, 0, len(devices))
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

		dev := DeviceModel{
			Name:       devName,
			Type:       devType,
			Properties: devProps,
		}

		deviceList = append(deviceList, dev)
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: deviceType}, deviceList)
}

// ToDeviceMapType converts devices from map[string]map[string]string
// into types.Map.
func ToDeviceMapType(ctx context.Context, devices map[string]map[string]string) (types.Map, diag.Diagnostics) {
	deviceType := map[string]attr.Type{
		"type":       types.StringType,
		"properties": types.MapType{ElemType: types.StringType},
	}

	nilMap := types.MapNull(types.ObjectType{AttrTypes: deviceType})

	if len(devices) == 0 {
		return nilMap, nil
	}

	deviceMap := make(map[string]attr.Value, len(devices))
	for key, props := range devices {
		devType := types.StringValue(props["type"])

		delete(props, "type")
		devProps, diags := types.MapValueFrom(ctx, types.StringType, props)
		if diags.HasError() {
			return nilMap, diags
		}

		dev := types.ObjectValueMust(deviceType, map[string]attr.Value{
			"type":       devType,
			"properties": devProps,
		})

		deviceMap[key] = dev
	}

	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: deviceType}, deviceMap)
}
