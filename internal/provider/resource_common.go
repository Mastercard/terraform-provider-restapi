package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CommonResourceModel holds the Terraform schema attributes shared between
// restapi_object and restapi_setting. Embed this struct in both resource models.
type CommonResourceModel struct {
	Data                  jsontypes.Normalized `tfsdk:"data"`
	Debug                 types.Bool           `tfsdk:"debug"`
	QueryString           types.String         `tfsdk:"query_string"`
	ForceNew              types.List           `tfsdk:"force_new"`
	ReadData              jsontypes.Normalized `tfsdk:"read_data"`
	UpdateData            jsontypes.Normalized `tfsdk:"update_data"`
	IgnoreChangesTo       types.List           `tfsdk:"ignore_changes_to"`
	IgnoreServerAdditions types.Bool           `tfsdk:"ignore_server_additions"`
	Headers               types.Map            `tfsdk:"headers"`

	APIData     types.Map    `tfsdk:"api_data"`
	APIResponse types.String `tfsdk:"api_response"`
}

// commonSchemaAttributes returns the schema attribute definitions shared between
// restapi_object and restapi_setting. Merge the returned map into each resource's
// full attribute map.
func commonSchemaAttributes(isDataSensitive bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"data": schema.StringAttribute{
			Description: "Valid JSON object that this provider will manage with the API server.",
			Required:    true,
			Sensitive:   isDataSensitive,
			CustomType:  jsontypes.NormalizedType{},
		},
		"debug": schema.BoolAttribute{
			Description: "Whether to emit the HTTP request and response to STDERR while working with the API object on the server.",
			Optional:    true,
		},
		"headers": schema.MapAttribute{
			ElementType: types.StringType,
			Description: "A map of header names and values to set on all outbound requests. This is useful if you want to modify header values which are set by the provider configuration",
			Optional:    true,
		},
		"query_string": schema.StringAttribute{
			Description: "Query string to be included in the path",
			Optional:    true,
		},
		"force_new": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Description: "Any changes to these values will result in recreating the resource instead of updating.",
		},
		"read_data": schema.StringAttribute{
			Optional:    true,
			Description: "Valid JSON object to pass during read requests.",
			Sensitive:   isDataSensitive,
			CustomType:  jsontypes.NormalizedType{},
		},
		"update_data": schema.StringAttribute{
			Optional:    true,
			Description: "Valid JSON object to pass during to update requests.",
			Sensitive:   isDataSensitive,
			CustomType:  jsontypes.NormalizedType{},
		},
		"ignore_changes_to": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Description: "A list of fields to which remote changes will be ignored. For example, an API might add or remove metadata, such as a 'last_modified' field, which Terraform should not attempt to correct. To ignore changes to nested fields, use the dot syntax: 'metadata.timestamp'",
			Sensitive:   isDataSensitive,
		},
		"ignore_server_additions": schema.BoolAttribute{
			Description: "When set to 'true', fields added by the server (but not present in your configuration) will be ignored for drift detection. This prevents resource recreation when the API returns additional fields like defaults, timestamps, or metadata. Unlike 'ignore_all_server_changes', this still detects when the server modifies fields you explicitly configured. Default: false",
			Optional:    true,
		},
		"api_data": schema.MapAttribute{
			ElementType: types.StringType,
			Description: "After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).",
			Computed:    true,
			Sensitive:   isDataSensitive,
		},
		"api_response": schema.StringAttribute{
			Description: "The raw body of the HTTP response from the last read of the object.",
			Computed:    true,
			Sensitive:   isDataSensitive,
		},
	}
}

// configureProviderData is the shared implementation of resource.Resource.Configure.
func configureProviderData(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *ProviderData {
	if req.ProviderData == nil {
		return nil
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. This should be impossible!", req.ProviderData),
		)
		return nil
	}

	return providerData
}

// modifyPlanDataResult contains the outputs of applyModifyPlanDataLogic.
type modifyPlanDataResult struct {
	// NewData is set (non-zero) when planData was normalized (null field removal).
	NewData         jsontypes.Normalized
	DataWasModified bool
	// PreserveAPIComputed is true when api_data/api_response should be copied from state.
	PreserveAPIComputed bool
	// ForceNewPaths are attribute paths that require resource replacement.
	ForceNewPaths []path.Path
}

// applyModifyPlanDataLogic applies the shared ModifyPlan logic for the data/force_new
// attributes. It handles ignore_server_additions, normalizeNullFields, and force_new
// field detection.
func applyModifyPlanDataLogic(
	ctx context.Context,
	planData, stateData map[string]interface{},
	planDataStr, stateDataStr jsontypes.Normalized,
	ignoreServerAdditions bool,
	forceNewList types.List,
	diags *diag.Diagnostics,
) modifyPlanDataResult {
	result := modifyPlanDataResult{}

	if ignoreServerAdditions {
		result.PreserveAPIComputed = planDataStr.Equal(stateDataStr)
	} else {
		if normalizeNullFields(planData, stateData) {
			normalizedJSON, err := json.Marshal(planData)
			if err != nil {
				diags.AddError(
					"Error Normalizing Null Fields",
					fmt.Sprintf("Could not marshal normalized data: %s", err.Error()),
				)
				return result
			}
			result.NewData = jsontypes.NewNormalizedValue(string(normalizedJSON))
			result.DataWasModified = true
			result.PreserveAPIComputed = true
		}
	}

	if !forceNewList.IsNull() && !forceNewList.IsUnknown() {
		var newFields []string
		diags.Append(forceNewList.ElementsAs(ctx, &newFields, false)...)
		if diags.HasError() {
			return result
		}
		for _, field := range newFields {
			if stateValue, err := getNestedValue(stateData, field); err == nil {
				planValue, _ := getNestedValue(planData, field)
				if fmt.Sprintf("%v", planValue) != fmt.Sprintf("%v", stateValue) {
					result.ForceNewPaths = append(result.ForceNewPaths, path.Root("api_data").AtMapKey(field))
				}
			}
		}
	}

	return result
}

// setCommonModelData populates the shared computed fields (api_response, api_data) on
// the common model. When ignoreServerAdditions is true, only keys present in configData
// are retained in the output.
func setCommonModelData(
	ctx context.Context,
	apiResponse string,
	apiData map[string]string,
	configData string,
	ignoreServerAdditions bool,
	model *CommonResourceModel,
	diags *diag.Diagnostics,
) {
	if ignoreServerAdditions && configData != "" && apiResponse != "" {
		filtered := filterAPIOutputToConfiguredKeys(apiResponse, configData)
		model.APIResponse = types.StringValue(filtered)

		filteredMap := make(map[string]string)
		var filteredJSON map[string]interface{}
		if err := json.Unmarshal([]byte(filtered), &filteredJSON); err == nil {
			for k, v := range filteredJSON {
				// Only include top-level keys from the filtered response
				filteredMap[k] = fmt.Sprintf("%v", v)
			}
		}
		v, d := types.MapValueFrom(ctx, types.StringType, filteredMap)
		model.APIData = v
		diags.Append(d...)
		return
	}

	model.APIResponse = types.StringValue(apiResponse)
	v, d := types.MapValueFrom(ctx, types.StringType, apiData)
	model.APIData = v
	diags.Append(d...)
}

// filterAPIOutputToConfiguredKeys filters a JSON API response string to only include
// top-level keys that are present in the configData JSON string. Nested maps are
// filtered recursively.
func filterAPIOutputToConfiguredKeys(apiResponse, configData string) string {
	var serverMap, configMap map[string]interface{}
	if err := json.Unmarshal([]byte(apiResponse), &serverMap); err != nil {
		return apiResponse
	}
	if err := json.Unmarshal([]byte(configData), &configMap); err != nil {
		return apiResponse
	}
	filtered := filterToConfiguredKeys(serverMap, configMap)
	b, err := json.Marshal(filtered)
	if err != nil {
		return apiResponse
	}
	return string(b)
}

// filterToConfiguredKeys returns a copy of serverData containing only the keys present
// in configData. Nested maps are filtered recursively.
func filterToConfiguredKeys(serverData, configData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key := range configData {
		val, ok := serverData[key]
		if !ok {
			continue
		}
		configVal := configData[key]
		if configMap, isConfigMap := configVal.(map[string]interface{}); isConfigMap {
			if serverMap, isServerMap := val.(map[string]interface{}); isServerMap {
				result[key] = filterToConfiguredKeys(serverMap, configMap)
				continue
			}
		}
		result[key] = val
	}
	return result
}

// isDataSensitiveFromEnv returns whether API data should be treated as sensitive
// based on the API_DATA_IS_SENSITIVE environment variable.
func isDataSensitiveFromEnv() bool {
	return strings.ToLower(os.Getenv("API_DATA_IS_SENSITIVE")) == "true"
}
