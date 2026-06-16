package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	restapi "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	CommonResourceModel
	ReadPath     types.String `tfsdk:"read_path"`
	UpdatePath   types.String `tfsdk:"update_path"`
	ReadMethod   types.String `tfsdk:"read_method"`
	UpdateMethod types.String `tfsdk:"update_method"`

	InitialResponse types.String `tfsdk:"initial_response"`
}

func NewRestAPISettingResource() resource.Resource {
	return &RestAPISettingResource{}
}

func (r *RestAPISettingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting"
}

func (r *RestAPISettingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	isDataSensitive := isDataSensitiveFromEnv()

	attrs := commonSchemaAttributes(isDataSensitive)

	// Setting-specific attributes (required paths override the optional ones in commonSchemaAttributes)
	attrs["read_path"] = schema.StringAttribute{
		Description: "Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
		Required:    true,
	}
	attrs["update_path"] = schema.StringAttribute{
		Description: "Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
		Required:    true,
	}
	attrs["read_method"] = schema.StringAttribute{
		Description: "Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation)",
		Optional:    true,
	}
	attrs["update_method"] = schema.StringAttribute{
		Description: "Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation)",
		Optional:    true,
	}
	attrs["initial_response"] = schema.StringAttribute{
		Description: "The raw body of the HTTP response returned when creating the object.",
		Computed:    true,
		Sensitive:   isDataSensitive,
	}

	resp.Schema = schema.Schema{
		Description:         "Acting as a restful API client, this setting supports POST, GET, PUT and DELETE on the specified url",
		MarkdownDescription: "Acting as a restful API client, this setting supports POST, GET, PUT and DELETE on the specified url",
		Attributes:          attrs,
	}
}

func (r *RestAPISettingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = configureProviderData(req, resp)
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

	// For Read we want to write to state only what was observed from the server.
	// When ignore_server_additions is true, filter out server-added fields while still
	// detecting drift in fields the user explicitly configured.
	if !state.IgnoreServerAdditions.IsNull() && state.IgnoreServerAdditions.ValueBool() && !state.Data.IsNull() && !state.Data.IsUnknown() {
		var ignoreFields []string
		if !state.IgnoreChangesTo.IsNull() && !state.IgnoreChangesTo.IsUnknown() {
			resp.Diagnostics.Append(state.IgnoreChangesTo.ElementsAs(ctx, &ignoreFields, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		serverData, configData := getPlanAndStateData(obj.GetApiResponse(), state.Data.ValueString(), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		mergedData, _ := getDelta(configData, serverData, ignoreFields, true)
		jsonData, err := json.Marshal(mergedData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Marshaling Merged Data",
				fmt.Sprintf("Could not marshal merged data: %s", err.Error()),
			)
			return
		}
		state.Data = jsontypes.NewNormalizedValue(string(jsonData))
	} else {
		state.Data = jsontypes.NewNormalizedValue(objString)
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

	result := applyModifyPlanDataLogic(ctx, planData, stateData, plan.Data, state.Data,
		!plan.IgnoreServerAdditions.IsNull() && plan.IgnoreServerAdditions.ValueBool(),
		plan.ForceNew, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if result.DataWasModified {
		plan.Data = result.NewData
	}
	if result.PreserveAPIComputed {
		plan.APIData = state.APIData
		plan.APIResponse = state.APIResponse
	}
	resp.RequiresReplace = append(resp.RequiresReplace, result.ForceNewPaths...)

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

func setResourceModelDataSetting(ctx context.Context, obj *apiclient.APISetting, data *RestAPISettingResourceModel, diags *diag.Diagnostics) {
	ignoreServerAdditions := !data.IgnoreServerAdditions.IsNull() && data.IgnoreServerAdditions.ValueBool()
	setCommonModelData(ctx, obj.GetApiResponse(), obj.GetApiData(), data.Data.ValueString(), ignoreServerAdditions, &data.CommonResourceModel, diags)
	if initialResponse := obj.GetInitialStateResponse(); initialResponse != "" {
		if ignoreServerAdditions {
			data.InitialResponse = types.StringValue(filterAPIOutputToConfiguredKeys(initialResponse, data.Data.ValueString()))
		} else {
			data.InitialResponse = types.StringValue(initialResponse)
		}
	}
}
