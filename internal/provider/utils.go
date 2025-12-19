package restapi

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func existingOrEnvOrDefaultString(d diag.Diagnostics, key string, curVal basetypes.StringValue, envKey string, def string, required bool) string {
	if !curVal.IsNull() {
		return curVal.ValueString()
	}

	val, ok := os.LookupEnv(envKey)
	if ok {
		return val
	}

	if required {
		d.AddError(
			"Missing Required Configuration",
			fmt.Sprintf("The %s configuration value is required. You can set this value in the provider configuration or in the %s environment variable.", key, envKey),
		)
	}

	return def
}

func existingOrEnvOrDefaultInt(d diag.Diagnostics, key string, curVal basetypes.Int64Value, envKey string, def int64, required bool) int64 {
	if !curVal.IsNull() {
		return curVal.ValueInt64()
	}

	val, ok := os.LookupEnv(envKey)
	if ok {
		tmp, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			d.AddError(
				"Invalid Configuration",
				fmt.Sprintf("The %s configuration value must be a valid integer. The value '%s' from the %s environment variable is not valid.", key, val, envKey),
			)
		}
		return tmp
	}

	if required {
		d.AddError(
			"Missing Required Configuration",
			fmt.Sprintf("The %s configuration value is required. You can set this value in the provider configuration or in the %s environment variable.", key, envKey),
		)
	}

	return def
}

func existingOrEnvOrDefaultFloat(d diag.Diagnostics, key string, curVal basetypes.Float64Value, envKey string, def float64, required bool) float64 {
	if !curVal.IsNull() {
		return curVal.ValueFloat64()
	}

	val, ok := os.LookupEnv(envKey)
	if ok {
		tmp, err := strconv.ParseFloat(val, 64)
		if err != nil {
			d.AddError(
				"Invalid Configuration",
				fmt.Sprintf("The %s configuration value must be a valid float. The value '%s' from the %s environment variable is not valid.", key, val, envKey),
			)
		}
		return tmp
	}

	if required {
		d.AddError(
			"Missing Required Configuration",
			fmt.Sprintf("The %s configuration value is required. You can set this value in the provider configuration or in the %s environment variable.", key, envKey),
		)
	}

	return def
}

func existingOrEnvOrDefaultBool(d diag.Diagnostics, key string, curVal basetypes.BoolValue, envKey string, def bool, required bool) bool {
	if !curVal.IsNull() {
		return curVal.ValueBool()
	}

	val, ok := os.LookupEnv(envKey)
	if ok {
		tmp, err := strconv.ParseBool(val)
		if err != nil {
			d.AddError(
				"Invalid Configuration",
				fmt.Sprintf("The %s configuration value must be a valid boolean. The value '%s' from the %s environment variable is not valid.", key, val, envKey),
			)
		}
		return tmp
	}

	if required {
		d.AddError(
			"Missing Required Configuration",
			fmt.Sprintf("The %s configuration value is required. You can set this value in the provider configuration or in the %s environment variable.", key, envKey),
		)
	}

	return def
}

func existingOrDefaultString(key string, curVal basetypes.StringValue, def string) string {
	if !curVal.IsNull() {
		return curVal.ValueString()
	}

	return def
}

func existingOrProviderOrDefaultString(key string, curVal basetypes.StringValue, provVal string, def string) string {
	if !curVal.IsNull() {
		return curVal.ValueString()
	}

	if provVal != "" {
		return provVal
	}

	return def
}

func convertListTypeToStringSlice(list types.List) []string {
	result := make([]string, 0)
	if list.IsNull() || list.IsUnknown() {
		return result
	}

	var listVals []basetypes.StringValue
	err := list.ElementsAs(context.Background(), &listVals, false)
	if err != nil {
		return result
	}

	for _, v := range listVals {
		if !v.IsNull() && !v.IsUnknown() {
			result = append(result, v.ValueString())
		}
	}

	return result
}
