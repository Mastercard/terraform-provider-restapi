package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	restapi "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RestAPISettingResource{}
var _ resource.ResourceWithModifyPlan = &RestAPISettingResource{}

type RestAPISettingResource struct {
	object       *restapi.APIObject
	providerData *ProviderData
}

type RestAPISettingResourceModel struct {
	ReadPath              types.String         `tfsdk:"read_path"`
	UpdatePath            types.String         `tfsdk:"update_path"`
	ReadMethod            types.String         `tfsdk:"read_method"`
	UpdateMethod          types.String         `tfsdk:"update_method"`
	Data                  jsontypes.Normalized `tfsdk:"data"`
	Debug                 types.Bool           `tfsdk:"debug"`
	QueryString           types.String         `tfsdk:"query_string"`
	ForceNew              types.List           `tfsdk:"force_new"`
	ReadData              jsontypes.Normalized `tfsdk:"read_data"`
	UpdateData            jsontypes.Normalized `tfsdk:"update_data"`
	IgnoreChangesTo       types.List           `tfsdk:"ignore_changes_to"`
	IgnoreServerAdditions types.Bool           `tfsdk:"ignore_server_additions"`
	Headers               types.Map            `tfsdk:"headers"`

	APIData         types.Map    `tfsdk:"api_data"`
	APIResponse     types.String `tfsdk:"api_response"`
	InitialResponse types.String `tfsdk:"initial_response"`
}

func NewRestAPISettingResource() resource.Resource {
	return &RestAPISettingResource{}
}

func (r *RestAPISettingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting"
}

func (r *RestAPISettingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Consider data sensitive if env variables is set to true.
	isDataSensitive := strings.ToLower(os.Getenv("API_DATA_IS_SENSITIVE")) == "true"

	resp.Schema = schema.Schema{
		Description:         "Acting as a restful API client, this setting supports POST, GET, PUT and DELETE on the specified url",
		MarkdownDescription: "Acting as a restful API client, this setting supports POST, GET, PUT and DELETE on the specified url",
		Attributes: map[string]schema.Attribute{
			"read_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Required:    true,
			},
			"update_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Required:    true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation)",
				Optional:    true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation)",
				Optional:    true,
			},
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
				// TODO ValidateFunc not supported for lists, but should probably validate that the ignore paths are valid
			},
			"ignore_server_additions": schema.BoolAttribute{
				Description: "When set to 'true', fields added by the server (but not present in your configuration) will be ignored for drift detection. This prevents resource recreation when the API returns additional fields like defaults, timestamps, or metadata. Unlike 'ignore_all_server_changes', this still detects when the server modifies fields you explicitly configured. Default: false",
				Optional:    true,
			},
			"initial_response": schema.StringAttribute{
				Description: "The raw body of the HTTP response returned when creating the object.",
				Computed:    true,
				Sensitive:   isDataSensitive,
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
		},
	}
}

func (r *RestAPISettingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. This should be impossible!", req.ProviderData),
		)
		return
	}

	r.providerData = providerData
}

func (r *RestAPISettingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RestAPISettingResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Create routine called", map[string]interface{}{"object": plan})

	client, err := r.providerData.GetClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			fmt.Sprintf("Failed to get API client: %s", err.Error()),
		)
		return
	}

	obj, err := makeAPISetting(ctx, client, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Setting",
			fmt.Sprintf("Could not create API setting: %s", err.Error()),
		)
		return
	}

	err = obj.CreateSetting(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Setting",
			fmt.Sprintf("Could not create API setting: %s", err.Error()),
		)
		return
	}

	setResourceModelDataSetting(ctx, obj, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RestAPISettingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RestAPISettingResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Read routine called", map[string]interface{}{"object": state})

	client, err := r.providerData.GetClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			fmt.Sprintf("Failed to get API client: %s", err.Error()),
		)
		return
	}

	obj, err := makeAPISetting(ctx, client, &state)
	if err != nil {
		if strings.Contains(err.Error(), "error parsing data provided") {
			tflog.Warn(ctx, "The data passed from Terraform's state is invalid!", map[string]interface{}{"error": err})
			tflog.Warn(ctx, "Continuing with partially constructed object...", nil)
		} else {
			resp.Diagnostics.AddError(
				"Error Creating API Object",
				fmt.Sprintf("Could not create API object: %s", err.Error()),
			)
			return
		}
	}

	err = obj.ReadSetting(ctx)
	if err != nil {
		tflog.Error(ctx, "Error reading API object", map[string]interface{}{"error": err})
		resp.Diagnostics.AddError(
			"Error Reading API Object",
			fmt.Sprintf("Could not read API object: %s", err.Error()),
		)
		return
	}
	objString := obj.GetApiResponse()
	tflog.Debug(ctx, "Read resource", map[string]interface{}{"id": obj.ID})

	// ignore_changes_to
	if !state.IgnoreChangesTo.IsNull() && !state.IgnoreChangesTo.IsUnknown() {
		var ignoreFields []string
		resp.Diagnostics.Append(state.IgnoreChangesTo.ElementsAs(ctx, &ignoreFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		tflog.Debug(ctx, "Read: processing ignore_changes_to", map[string]interface{}{
			"ignoreFields": ignoreFields,
		})

		if len(ignoreFields) > 0 {
			// Skip ignore_changes_to processing if state.Data is null or unknown
			// This can happen when the data attribute contains unknown interpolations
			if state.Data.IsNull() || state.Data.IsUnknown() {
				tflog.Debug(ctx, "Read: skipping ignore_changes_to processing due to null/unknown state data",
					map[string]interface{}{
						"state_data_null":    state.Data.IsNull(),
						"state_data_unknown": state.Data.IsUnknown(),
					})
			} else {
				planData, stateData := getPlanAndStateData(obj.GetApiResponse(), state.Data.ValueString(), &resp.Diagnostics)
				if resp.Diagnostics.HasError() {
					return
				}

				tflog.Debug(ctx, "Read: before ignoring", map[string]interface{}{
					"planData":  planData,
					"stateData": stateData,
				})

				ignoreServerAdditions := !state.IgnoreServerAdditions.IsNull() && state.IgnoreServerAdditions.ValueBool()
				mergedData, hasDelta := getDelta(stateData, planData, ignoreFields, ignoreServerAdditions)
				tflog.Debug(ctx, "Read: after ignoring", map[string]interface{}{
					"mergedData": mergedData,
					"hasDelta":   hasDelta,
				})

				jsonData, err := json.Marshal(mergedData)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Marshaling Merged Data",
						fmt.Sprintf("Could not marshal merged data: %s", err.Error()),
					)
					return
				}
				objString = string(jsonData)
			}
		}
	}

	// For Read we want to write to state only what was observed from the server - this may later be negated during ModifyPlan
	// However, when ignore_server_additions is true, we should NOT overwrite state.Data with the full API response
	// because the config only contains the fields the user configured, not the server-added fields
	if state.IgnoreServerAdditions.IsNull() || !state.IgnoreServerAdditions.ValueBool() {
		state.Data = jsontypes.NewNormalizedValue(objString)
	} else {
		tflog.Debug(ctx, "Read: keeping state.Data unchanged due to ignore_server_additions")
	}
	setResourceModelDataSetting(ctx, obj, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RestAPISettingResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "ModifyPlan routine called")

	// Don't modify plan during resource creation or destruction
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan RestAPISettingResourceModel
	var state RestAPISettingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Skip plan modification if data is unknown/null (e.g., contains computed values)
	if plan.Data.IsUnknown() || plan.Data.IsNull() || state.Data.IsUnknown() || state.Data.IsNull() {
		tflog.Debug(ctx, "ModifyPlan: skipping due to unknown/null data")
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		return
	}

	// Parse JSON data once for use below
	planData, stateData := getPlanAndStateData(plan.Data.ValueString(), state.Data.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.IgnoreServerAdditions.IsNull() && plan.IgnoreServerAdditions.ValueBool() {
		// ignore_server_additions: Read preserves user config in state.Data,
		// so we just preserve api_data/api_response if user didn't change anything
		if plan.Data.Equal(state.Data) {
			plan.APIData = state.APIData
			plan.APIResponse = state.APIResponse
		}
	} else {
		// Normal flow: normalize null fields that server omits
		if normalizeNullFields(planData, stateData) {
			normalizedJSON, err := json.Marshal(planData)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Normalizing Null Fields",
					fmt.Sprintf("Could not marshal normalized data: %s", err.Error()),
				)
				return
			}
			plan.Data = jsontypes.NewNormalizedValue(string(normalizedJSON))
			plan.APIData = state.APIData
			plan.APIResponse = state.APIResponse
		}
	}

	// force_new: check if any fields require resource replacement
	if !plan.ForceNew.IsNull() && !plan.ForceNew.IsUnknown() {
		var newFields []string
		resp.Diagnostics.Append(plan.ForceNew.ElementsAs(ctx, &newFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, field := range newFields {
			if stateValue, err := getNestedValue(stateData, field); err == nil {
				planValue, _ := getNestedValue(planData, field)
				if fmt.Sprintf("%v", planValue) != fmt.Sprintf("%v", stateValue) {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("api_data").AtMapKey(field))
				}
			}
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *RestAPISettingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RestAPISettingResourceModel
	var state RestAPISettingResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read prior state to preserve computed fields like create_response
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Computed fields may be unknown in plan during Update; carry forward from state.
	if plan.InitialResponse.IsNull() || plan.InitialResponse.IsUnknown() {
		plan.InitialResponse = state.InitialResponse
	}

	tflog.Debug(ctx, "Update routine called", map[string]interface{}{"object": plan})

	client, err := r.providerData.GetClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			fmt.Sprintf("Failed to get API client: %s", err.Error()),
		)
		return
	}

	obj, err := makeAPISetting(ctx, client, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	err = obj.UpdateSetting(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating API Object",
			fmt.Sprintf("Could not update API object: %s", err.Error()),
		)

		// Read the current state from the server to get actual values after failed update
		readErr := obj.ReadSetting(ctx)
		if readErr != nil {
			tflog.Error(ctx, "Failed to read object after failed update", map[string]interface{}{"error": readErr})
			// Continue with what we have - better to save something than nothing
		}

		// Update plan.Data to reflect the actual server state, not the desired state
		if apiResponse := obj.GetApiResponse(); len(apiResponse) > 0 {
			plan.Data = jsontypes.NewNormalizedValue(apiResponse)
		}
	}

	setResourceModelDataSetting(ctx, obj, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RestAPISettingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RestAPISettingResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Delete routine called", map[string]interface{}{"state": state})

	client, err := r.providerData.GetClient()
	if err != nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			fmt.Sprintf("Failed to get API client: %s", err.Error()),
		)
		return
	}

	obj, err := makeAPISetting(ctx, client, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	err = obj.DeleteSetting(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting API Object",
			fmt.Sprintf("Could not delete API object: %s", err.Error()),
		)
		return
	}

}

// makeAPIObject creates APIObjectOpts from the resource model
func makeAPISetting(ctx context.Context, client *apiclient.APIClient, model *RestAPISettingResourceModel) (*apiclient.APISetting, error) {
	tflog.Debug(ctx, "makeAPISetting routine called", map[string]interface{}{"read_path": model.ReadPath.ValueString()})

	opts := &apiclient.APISettingOpts{
		Data:  model.Data.ValueString(),
		Debug: model.Debug.ValueBool(),

		ReadPath:   existingOrDefaultString(model.ReadPath, ""),
		ReadMethod: existingOrProviderOrDefaultString(model.ReadMethod, client.Opts.ReadMethod, "GET"),
		ReadData:   model.ReadData.ValueString(),

		UpdatePath:   existingOrDefaultString(model.UpdatePath, ""),
		UpdateMethod: existingOrProviderOrDefaultString(model.UpdateMethod, client.Opts.UpdateMethod, "PUT"),
		UpdateData:   model.UpdateData.ValueString(),
		InitialState: model.InitialResponse.ValueString(),

		QueryString: existingOrDefaultString(model.QueryString, ""),
	}

	if !model.Headers.IsNull() {
		headers := make(map[string]string, len(model.Headers.Elements()))
		diags := model.Headers.ElementsAs(ctx, &headers, false)
		if diags.HasError() {
			return nil, errors.New("Could not convert resource headers to map")
		}
		opts.Headers = headers
	}

	return apiclient.NewAPISetting(client, opts)
}

func setResourceModelDataSetting(ctx context.Context, obj *apiclient.APISetting, data *RestAPISettingResourceModel, diag *diag.Diagnostics) {
	data.APIResponse = types.StringValue(obj.GetApiResponse())
	if initialResponse := obj.GetInitialStateResponse(); initialResponse != "" {
		data.InitialResponse = types.StringValue(initialResponse)
	}
	v, d := types.MapValueFrom(ctx, types.StringType, obj.GetApiData())
	data.APIData = v
	diag.Append(d...)
}
