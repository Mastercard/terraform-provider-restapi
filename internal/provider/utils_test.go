package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetNestedValue(t *testing.T) {
	tests := map[string]struct {
		data        map[string]interface{}
		path        string
		expected    interface{}
		expectError bool
	}{
		"simple_field": {
			data:        map[string]interface{}{"name": "test"},
			path:        "name",
			expected:    "test",
			expectError: false,
		},
		"nested_field": {
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
				},
			},
			path:        "metadata.timestamp",
			expected:    "2024-01-01",
			expectError: false,
		},
		"deeply_nested": {
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "value",
					},
				},
			},
			path:        "level1.level2.level3",
			expected:    "value",
			expectError: false,
		},
		"field_not_found": {
			data:        map[string]interface{}{"name": "test"},
			path:        "missing",
			expected:    nil,
			expectError: true,
		},
		"nested_field_not_found": {
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
				},
			},
			path:        "metadata.missing",
			expected:    nil,
			expectError: true,
		},
		"path_through_non_map": {
			data: map[string]interface{}{
				"metadata": "string_not_map",
			},
			path:        "metadata.timestamp",
			expected:    nil,
			expectError: true,
		},
		"empty_path": {
			data:        map[string]interface{}{"name": "test"},
			path:        "",
			expected:    nil,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := getNestedValue(tc.data, tc.path)
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tc.expected {
					t.Errorf("expected %v, got %v", tc.expected, result)
				}
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	tests := map[string]struct {
		initialData map[string]interface{}
		path        string
		value       interface{}
		expected    map[string]interface{}
	}{
		"simple_field": {
			initialData: map[string]interface{}{},
			path:        "name",
			value:       "test",
			expected:    map[string]interface{}{"name": "test"},
		},
		"nested_field_new": {
			initialData: map[string]interface{}{},
			path:        "metadata.timestamp",
			value:       "2024-01-01",
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
				},
			},
		},
		"nested_field_existing": {
			initialData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"version": "1.0",
				},
			},
			path:  "metadata.timestamp",
			value: "2024-01-01",
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"version":   "1.0",
					"timestamp": "2024-01-01",
				},
			},
		},
		"deeply_nested": {
			initialData: map[string]interface{}{},
			path:        "level1.level2.level3",
			value:       "value",
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "value",
					},
				},
			},
		},
		"overwrite_existing": {
			initialData: map[string]interface{}{"name": "old"},
			path:        "name",
			value:       "new",
			expected:    map[string]interface{}{"name": "new"},
		},
		"overwrite_non_map_with_nested": {
			initialData: map[string]interface{}{"metadata": "string"},
			path:        "metadata.timestamp",
			value:       "2024-01-01",
			expected: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			data := tc.initialData
			setNestedValue(data, tc.path, tc.value)

			// Verify the result
			result, err := getNestedValue(data, tc.path)
			if err != nil {
				t.Errorf("failed to get nested value after setting: %v", err)
			}
			if result != tc.value {
				t.Errorf("expected %v, got %v", tc.value, result)
			}
		})
	}
}

func TestExistingOrProviderOrDefaultString(t *testing.T) {
	tests := map[string]struct {
		curVal   types.String
		provVal  string
		def      string
		expected string
	}{
		"use_current_value": {
			curVal:   types.StringValue("current"),
			provVal:  "provider",
			def:      "default",
			expected: "current",
		},
		"use_provider_value": {
			curVal:   types.StringNull(),
			provVal:  "provider",
			def:      "default",
			expected: "provider",
		},
		"use_default_value": {
			curVal:   types.StringNull(),
			provVal:  "",
			def:      "default",
			expected: "default",
		},
		"empty_string_is_valid": {
			curVal:   types.StringValue(""),
			provVal:  "provider",
			def:      "default",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := existingOrProviderOrDefaultString(tc.curVal, tc.provVal, tc.def)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestExistingOrEnvOrDefaultString(t *testing.T) {
	tests := map[string]struct {
		curVal      types.String
		envKey      string
		envValue    string
		def         string
		required    bool
		expected    string
		expectError bool
	}{
		"use_current_value": {
			curVal:      types.StringValue("current"),
			envKey:      "TEST_VAR",
			envValue:    "env",
			def:         "default",
			required:    false,
			expected:    "current",
			expectError: false,
		},
		"use_env_value": {
			curVal:      types.StringNull(),
			envKey:      "TEST_VAR",
			envValue:    "env",
			def:         "default",
			required:    false,
			expected:    "env",
			expectError: false,
		},
		"use_default_value": {
			curVal:      types.StringNull(),
			envKey:      "TEST_VAR_UNSET",
			envValue:    "",
			def:         "default",
			required:    false,
			expected:    "default",
			expectError: false,
		},
		"required_missing": {
			curVal:      types.StringNull(),
			envKey:      "TEST_VAR_UNSET",
			envValue:    "",
			def:         "default",
			required:    true,
			expected:    "default",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
				defer os.Unsetenv(tc.envKey)
			}

			d := diag.Diagnostics{}
			result := existingOrEnvOrDefaultString(&d, "test_key", tc.curVal, tc.envKey, tc.def, tc.required)

			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}

			if tc.expectError && !d.HasError() {
				t.Error("expected error but got none")
			}
			if !tc.expectError && d.HasError() {
				t.Errorf("unexpected error: %v", d.Errors())
			}
		})
	}
}

func TestExistingOrEnvOrDefaultInt(t *testing.T) {
	tests := map[string]struct {
		curVal      types.Int64
		envKey      string
		envValue    string
		def         int64
		required    bool
		expected    int64
		expectError bool
	}{
		"use_current_value": {
			curVal:      types.Int64Value(42),
			envKey:      "TEST_INT",
			envValue:    "100",
			def:         0,
			required:    false,
			expected:    42,
			expectError: false,
		},
		"use_env_value": {
			curVal:      types.Int64Null(),
			envKey:      "TEST_INT",
			envValue:    "100",
			def:         0,
			required:    false,
			expected:    100,
			expectError: false,
		},
		"use_default_value": {
			curVal:      types.Int64Null(),
			envKey:      "TEST_INT_UNSET",
			envValue:    "",
			def:         99,
			required:    false,
			expected:    99,
			expectError: false,
		},
		"invalid_env_value": {
			curVal:      types.Int64Null(),
			envKey:      "TEST_INT",
			envValue:    "not_a_number",
			def:         0,
			required:    false,
			expected:    0,
			expectError: true,
		},
		"required_missing": {
			curVal:      types.Int64Null(),
			envKey:      "TEST_INT_UNSET",
			envValue:    "",
			def:         0,
			required:    true,
			expected:    0,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
				defer os.Unsetenv(tc.envKey)
			}

			d := diag.Diagnostics{}
			result := existingOrEnvOrDefaultInt(&d, "test_key", tc.curVal, tc.envKey, tc.def, tc.required)

			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}

			if tc.expectError && !d.HasError() {
				t.Error("expected error but got none")
			}
			if !tc.expectError && d.HasError() {
				t.Errorf("unexpected error: %v", d.Errors())
			}
		})
	}
}

func TestExistingOrEnvOrDefaultFloat(t *testing.T) {
	tests := map[string]struct {
		curVal      types.Float64
		envKey      string
		envValue    string
		def         float64
		required    bool
		expected    float64
		expectError bool
	}{
		"use_current_value": {
			curVal:      types.Float64Value(3.14),
			envKey:      "TEST_FLOAT",
			envValue:    "2.71",
			def:         0.0,
			required:    false,
			expected:    3.14,
			expectError: false,
		},
		"use_env_value": {
			curVal:      types.Float64Null(),
			envKey:      "TEST_FLOAT",
			envValue:    "2.71",
			def:         0.0,
			required:    false,
			expected:    2.71,
			expectError: false,
		},
		"use_default_value": {
			curVal:      types.Float64Null(),
			envKey:      "TEST_FLOAT_UNSET",
			envValue:    "",
			def:         1.5,
			required:    false,
			expected:    1.5,
			expectError: false,
		},
		"invalid_env_value": {
			curVal:      types.Float64Null(),
			envKey:      "TEST_FLOAT",
			envValue:    "not_a_float",
			def:         0.0,
			required:    false,
			expected:    0.0,
			expectError: true,
		},
		"required_missing": {
			curVal:      types.Float64Null(),
			envKey:      "TEST_FLOAT_UNSET",
			envValue:    "",
			def:         0.0,
			required:    true,
			expected:    0.0,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
				defer os.Unsetenv(tc.envKey)
			}

			d := diag.Diagnostics{}
			result := existingOrEnvOrDefaultFloat(&d, "test_key", tc.curVal, tc.envKey, tc.def, tc.required)

			if result != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, result)
			}

			if tc.expectError && !d.HasError() {
				t.Error("expected error but got none")
			}
			if !tc.expectError && d.HasError() {
				t.Errorf("unexpected error: %v", d.Errors())
			}
		})
	}
}

func TestExistingOrEnvOrDefaultBool(t *testing.T) {
	tests := map[string]struct {
		curVal      types.Bool
		envKey      string
		envValue    string
		def         bool
		required    bool
		expected    bool
		expectError bool
	}{
		"use_current_value_true": {
			curVal:      types.BoolValue(true),
			envKey:      "TEST_BOOL",
			envValue:    "false",
			def:         false,
			required:    false,
			expected:    true,
			expectError: false,
		},
		"use_current_value_false": {
			curVal:      types.BoolValue(false),
			envKey:      "TEST_BOOL",
			envValue:    "true",
			def:         true,
			required:    false,
			expected:    false,
			expectError: false,
		},
		"use_env_value_true": {
			curVal:      types.BoolNull(),
			envKey:      "TEST_BOOL",
			envValue:    "true",
			def:         false,
			required:    false,
			expected:    true,
			expectError: false,
		},
		"use_env_value_1": {
			curVal:      types.BoolNull(),
			envKey:      "TEST_BOOL",
			envValue:    "1",
			def:         false,
			required:    false,
			expected:    true,
			expectError: false,
		},
		"use_env_value_false": {
			curVal:      types.BoolNull(),
			envKey:      "TEST_BOOL",
			envValue:    "false",
			def:         true,
			required:    false,
			expected:    false,
			expectError: false,
		},
		"use_default_value": {
			curVal:      types.BoolNull(),
			envKey:      "TEST_BOOL_UNSET",
			envValue:    "",
			def:         true,
			required:    false,
			expected:    true,
			expectError: false,
		},
		"invalid_env_value": {
			curVal:      types.BoolNull(),
			envKey:      "TEST_BOOL",
			envValue:    "not_a_bool",
			def:         false,
			required:    false,
			expected:    false,
			expectError: true,
		},
		"required_missing": {
			curVal:      types.BoolNull(),
			envKey:      "TEST_BOOL_UNSET",
			envValue:    "",
			def:         false,
			required:    true,
			expected:    false,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv(tc.envKey, tc.envValue)
				defer os.Unsetenv(tc.envKey)
			}

			d := diag.Diagnostics{}
			result := existingOrEnvOrDefaultBool(&d, "test_key", tc.curVal, tc.envKey, tc.def, tc.required)

			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}

			if tc.expectError && !d.HasError() {
				t.Error("expected error but got none")
			}
			if !tc.expectError && d.HasError() {
				t.Errorf("unexpected error: %v", d.Errors())
			}
		})
	}
}

func TestGetPlanAndStateData(t *testing.T) {
	tests := map[string]struct {
		planJSON    string
		stateJSON   string
		expectError bool
	}{
		"valid_json": {
			planJSON:    `{"name":"test","value":123}`,
			stateJSON:   `{"name":"test","value":123}`,
			expectError: false,
		},
		"empty_json": {
			planJSON:    `{}`,
			stateJSON:   `{}`,
			expectError: false,
		},
		"nested_json": {
			planJSON:    `{"metadata":{"timestamp":"2024-01-01"},"data":"test"}`,
			stateJSON:   `{"metadata":{"timestamp":"2024-01-01"},"data":"test"}`,
			expectError: false,
		},
		"invalid_plan_json": {
			planJSON:    `{invalid json}`,
			stateJSON:   `{"name":"test"}`,
			expectError: true,
		},
		"invalid_state_json": {
			planJSON:    `{"name":"test"}`,
			stateJSON:   `{invalid json}`,
			expectError: true,
		},
		"both_invalid": {
			planJSON:    `{invalid}`,
			stateJSON:   `{also invalid}`,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			d := diag.Diagnostics{}
			planData, stateData := getPlanAndStateData(tc.planJSON, tc.stateJSON, &d)

			if tc.expectError {
				if !d.HasError() {
					t.Error("expected error but got none")
				}
				if planData != nil || stateData != nil {
					t.Error("expected nil data on error")
				}
			} else {
				if d.HasError() {
					t.Errorf("unexpected error: %v", d.Errors())
				}
				if planData == nil || stateData == nil {
					t.Error("expected non-nil data on success")
				}
			}
		})
	}
}

func TestNormalizeNullFields(t *testing.T) {
	tests := map[string]struct {
		planData         map[string]interface{}
		stateData        map[string]interface{}
		expectModified   bool
		expectedPlanData map[string]interface{}
	}{
		"remove_null_field_missing_in_state": {
			planData:         map[string]interface{}{"name": "test", "optional": nil},
			stateData:        map[string]interface{}{"name": "test"},
			expectModified:   true,
			expectedPlanData: map[string]interface{}{"name": "test"},
		},
		"keep_null_field_present_in_state": {
			planData:         map[string]interface{}{"name": "test", "optional": nil},
			stateData:        map[string]interface{}{"name": "test", "optional": nil},
			expectModified:   false,
			expectedPlanData: map[string]interface{}{"name": "test", "optional": nil},
		},
		"no_null_fields": {
			planData:         map[string]interface{}{"name": "test", "value": 123},
			stateData:        map[string]interface{}{"name": "test", "value": 123},
			expectModified:   false,
			expectedPlanData: map[string]interface{}{"name": "test", "value": 123},
		},
		"nested_null_removal": {
			planData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
					"optional":  nil,
				},
			},
			stateData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
				},
			},
			expectModified: true,
			expectedPlanData: map[string]interface{}{
				"metadata": map[string]interface{}{
					"timestamp": "2024-01-01",
				},
			},
		},
		"multiple_null_fields": {
			planData:         map[string]interface{}{"name": "test", "field1": nil, "field2": nil},
			stateData:        map[string]interface{}{"name": "test"},
			expectModified:   true,
			expectedPlanData: map[string]interface{}{"name": "test"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			modified := normalizeNullFields(tc.planData, tc.stateData)

			if modified != tc.expectModified {
				t.Errorf("expected modified=%v, got %v", tc.expectModified, modified)
			}

			// Check that the plan data matches expected
			if len(tc.planData) != len(tc.expectedPlanData) {
				t.Errorf("expected %d fields, got %d", len(tc.expectedPlanData), len(tc.planData))
			}

			for key, expectedValue := range tc.expectedPlanData {
				actualValue, exists := tc.planData[key]
				if !exists {
					t.Errorf("expected field %s not found in plan data", key)
					continue
				}

				// For nested maps, recursively compare
				if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
					actualMap, ok := actualValue.(map[string]interface{})
					if !ok {
						t.Errorf("expected %s to be a map", key)
						continue
					}
					if len(actualMap) != len(expectedMap) {
						t.Errorf("nested map %s: expected %d fields, got %d", key, len(expectedMap), len(actualMap))
					}
				}
			}
		})
	}
}
