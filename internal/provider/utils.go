package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

func getPlanAndStateData(planDataString, stateDataString string, diag *diag.Diagnostics) (map[string]interface{}, map[string]interface{}) {
	planData := make(map[string]interface{})
	stateData := make(map[string]interface{})
	if err := json.Unmarshal([]byte(planDataString), &planData); err != nil {
		diag.AddError(
			"Error Parsing Plan Data",
			fmt.Sprintf("Could not parse plan data JSON: %s", err.Error()),
		)
		return nil, nil
	}
	if err := json.Unmarshal([]byte(stateDataString), &stateData); err != nil {
		diag.AddError(
			"Error Parsing Server Data",
			fmt.Sprintf("Could not parse server data JSON: %s", err.Error()),
		)
		return nil, nil
	}
	return planData, stateData
}

// Helper function to get nested value using dot notation (e.g., "metadata.timestamp")
func getNestedValue(data map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			val, exists := current[part]
			if !exists {
				return nil, fmt.Errorf("field %s not found", part)
			}
			return val, nil
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path %s not found", path)
		}
		current = next
	}
	return nil, fmt.Errorf("empty path")
}

// Helper function to set nested value using dot notation
func setNestedValue(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[part] = next
		}
		current = next
	}
}

// normalizeNullFields removes fields from planData that are null and missing from stateData.
// This handles the common REST API pattern where null fields are omitted from responses.
// Returns true if planData was modified, false otherwise.
func normalizeNullFields(planData, stateData map[string]interface{}) bool {
	modified := false

	for key, planValue := range planData {
		if planValue == nil {
			if _, existsInState := stateData[key]; !existsInState {
				delete(planData, key)
				modified = true
				continue
			}
		}

		if planMap, isPlanMap := planValue.(map[string]interface{}); isPlanMap {
			if stateValue, existsInState := stateData[key]; existsInState {
				if stateMap, isStateMap := stateValue.(map[string]interface{}); isStateMap {
					if normalizeNullFields(planMap, stateMap) {
						modified = true
					}
				}
			}
		}
	}

	return modified
}
