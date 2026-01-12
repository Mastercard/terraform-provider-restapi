package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
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
var _ resource.Resource = &RestAPIObjectResource{}
var _ resource.ResourceWithImportState = &RestAPIObjectResource{}
var _ resource.ResourceWithModifyPlan = &RestAPIObjectResource{}

type RestAPIObjectResource struct {
	object       *restapi.APIObject
	providerData *ProviderData
}

type RestAPIObjectResourceModel struct {
	Path                   types.String         `tfsdk:"path"`
	CreatePath             types.String         `tfsdk:"create_path"`
	ReadPath               types.String         `tfsdk:"read_path"`
	UpdatePath             types.String         `tfsdk:"update_path"`
	DestroyPath            types.String         `tfsdk:"destroy_path"`
	CreateMethod           types.String         `tfsdk:"create_method"`
	ReadMethod             types.String         `tfsdk:"read_method"`
	UpdateMethod           types.String         `tfsdk:"update_method"`
	DestroyMethod          types.String         `tfsdk:"destroy_method"`
	IDAttribute            types.String         `tfsdk:"id_attribute"`
	ObjectID               types.String         `tfsdk:"object_id"`
	Data                   jsontypes.Normalized `tfsdk:"data"`
	Debug                  types.Bool           `tfsdk:"debug"`
	ReadSearch             *ReadSearchModel     `tfsdk:"read_search"`
	QueryString            types.String         `tfsdk:"query_string"`
	ForceNew               types.List           `tfsdk:"force_new"`
	ReadData               jsontypes.Normalized `tfsdk:"read_data"`
	UpdateData             jsontypes.Normalized `tfsdk:"update_data"`
	DestroyData            jsontypes.Normalized `tfsdk:"destroy_data"`
	IgnoreChangesTo        types.List           `tfsdk:"ignore_changes_to"`
	IgnoreAllServerChanges types.Bool           `tfsdk:"ignore_all_server_changes"`
	IgnoreServerAdditions  types.Bool           `tfsdk:"ignore_server_additions"`

	ID             types.String `tfsdk:"id"`
	APIData        types.Map    `tfsdk:"api_data"`
	APIResponse    types.String `tfsdk:"api_response"`
	CreateResponse types.String `tfsdk:"create_response"`
}

type ReadSearchModel struct {
	SearchData   jsontypes.Normalized `tfsdk:"search_data"`
	SearchKey    types.String         `tfsdk:"search_key"`
	SearchValue  types.String         `tfsdk:"search_value"`
	ResultsKey   types.String         `tfsdk:"results_key"`
	QueryString  types.String         `tfsdk:"query_string"`
	SearchPatch  jsontypes.Normalized `tfsdk:"search_patch"`
}

func NewRestAPIObjectResource() resource.Resource {
	return &RestAPIObjectResource{}
}

func (r *RestAPIObjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *RestAPIObjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Consider data sensitive if env variables is set to true.
	isDataSensitive := strings.ToLower(os.Getenv("API_DATA_IS_SENSITIVE")) == "true"

	resp.Schema = schema.Schema{
		Description:         "Acting as a restful API client, this object supports POST, GET, PUT and DELETE on the specified url",
		MarkdownDescription: "Acting as a restful API client, this object supports POST, GET, PUT and DELETE on the specified url",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
				Required:    true,
			},
			"create_path": schema.StringAttribute{
				Description: "Defaults to `path`. The API path that represents where to CREATE (POST) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object if the data contains the `id_attribute`.",
				Optional:    true,
			},
			"read_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"update_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"create_method": schema.StringAttribute{
				Description: "Defaults to `create_method` set on the provider. Allows per-resource override of `create_method` (see `create_method` provider config documentation)",
				Optional:    true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation)",
				Optional:    true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation)",
				Optional:    true,
			},
			"destroy_method": schema.StringAttribute{
				Description: "Defaults to `destroy_method` set on the provider. Allows per-resource override of `destroy_method` (see `destroy_method` provider config documentation)",
				Optional:    true,
			},
			"destroy_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that represents where to DESTROY (DELETE) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"id_attribute": schema.StringAttribute{
				Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
				Optional:    true,
			},
			"object_id": schema.StringAttribute{
				Description: "Defaults to the id learned by the provider during normal operations and `id_attribute`. Allows you to set the id manually. This is used in conjunction with the `*_path` attributes.",
				Optional:    true,
			},
			"data": schema.StringAttribute{
				Description: "Valid JSON object that this provider will manage with the API server.",
				Required:    true,
				Sensitive:   isDataSensitive,
				CustomType:  jsontypes.NormalizedType{},
			},
			"debug": schema.BoolAttribute{
				Description: "Whether to emit the HTTP request and response to STDOUT while working with the API object on the server.",
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
			"destroy_data": schema.StringAttribute{
				Optional:    true,
				Description: "Valid JSON object to pass during to destroy requests.",
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
			"ignore_all_server_changes": schema.BoolAttribute{
				Description: "By default Terraform will attempt to revert changes to remote resources. Set this to 'true' to ignore any remote changes. Default: false",
				Optional:    true,
			},
			"ignore_server_additions": schema.BoolAttribute{
				Description: "When set to 'true', fields added by the server (but not present in your configuration) will be ignored for drift detection. This prevents resource recreation when the API returns additional fields like defaults, timestamps, or metadata. Unlike 'ignore_all_server_changes', this still detects when the server modifies fields you explicitly configured. Default: false",
				Optional:    true,
			},
			"read_search": schema.SingleNestedAttribute{
				Description: "Custom search for `read_path`. This map will take `search_data`, `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation)",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"query_string": schema.StringAttribute{
						Description: "An optional query string to send when performing the search.",
						Optional:    true,
					},
					"search_data": schema.StringAttribute{
						Description: "Valid JSON object to pass to search request as body",
						Optional:    true,
						CustomType:  jsontypes.NormalizedType{},
					},
					"search_key": schema.StringAttribute{
						Description: "When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'. Similar to results_key, the value may be in the format of 'field/field/field' to search for data deeper in the returned object.",
						Required:    true,
					},
					"search_value": schema.StringAttribute{
						Description: "The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used. Supports interpolation of {id} placeholder with the object's ID.",
						Required:    true,
					},
					"results_key": schema.StringAttribute{
						Description: "When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are already an array and are to be used exactly as-is.",
						Optional:    true,
					},
					"search_patch": schema.StringAttribute{
						Description: "A JSON Patch (RFC 6902) to apply to the search result before storing in state. This allows transformation of the API response to match the expected data structure. Example: [{\"op\":\"move\",\"from\":\"/old\",\"path\":\"/new\"}]",
						Optional:    true,
						CustomType:  jsontypes.NormalizedType{},
					},
				},
			},

			"create_response": schema.StringAttribute{
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
			"id": schema.StringAttribute{
				Description: "The ID of the object.",
				Computed:    true,
			},
		},
	}
}

func (r *RestAPIObjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RestAPIObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RestAPIObjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Create routine called", map[string]interface{}{"object": plan})
	obj, err := makeAPIObject(ctx, r.providerData.client, "", &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	err = obj.CreateObject(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	setResourceModelData(ctx, obj, &plan, &resp.Diagnostics)
	plan.CreateResponse = types.StringValue(obj.GetApiResponse())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RestAPIObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RestAPIObjectResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Read routine called", map[string]interface{}{"object": state})
	obj, err := makeAPIObject(ctx, r.providerData.client, state.ID.ValueString(), &state)
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

	err = obj.ReadObject(ctx)
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
			planData, stateData := getPlanAndStateData(obj.GetApiResponse(), state.Data.ValueString(), &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}

			tflog.Debug(ctx, "ModifyPlan: before ignoring", map[string]interface{}{
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

	// For Read we want to write to state only what was observed from the server - this may later be negated during ModifyPlan
	state.Data = jsontypes.NewNormalizedValue(objString)
	setResourceModelData(ctx, obj, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RestAPIObjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "ModifyPlan routine called")

	// Don't modify plan during resource creation or destruction
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan RestAPIObjectResourceModel
	var state RestAPIObjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ignore_all_server_changes
	if !plan.IgnoreAllServerChanges.IsNull() && plan.IgnoreAllServerChanges.ValueBool() {
		// Reset data back to state value to ignore all server-side changes
		plan.ID = state.ID
		plan.Data = state.Data
		plan.APIData = state.APIData
		plan.APIResponse = state.APIResponse
		plan.CreateResponse = state.CreateResponse

		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		return
	}

	// If plan has a null field that's missing from state, remove it from plan
	// This prevents drift detection when the server omits null fields (common REST API behavior)
	planData, stateData := getPlanAndStateData(plan.Data.ValueString(), state.Data.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if normalizeNullFields(planData, stateData) {
		normalizedJSON, err := json.Marshal(planData)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Normalizing Null Fields",
				fmt.Sprintf("Could not marshal normalized data: %s", err.Error()),
			)
			return
		}
		plan.ID = state.ID
		plan.Data = jsontypes.NewNormalizedValue(string(normalizedJSON))
		plan.APIData = state.APIData
		plan.APIResponse = state.APIResponse
		plan.CreateResponse = state.CreateResponse
	}

	// force_new
	if !plan.ForceNew.IsNull() && !plan.ForceNew.IsUnknown() {
		var newFields []string
		resp.Diagnostics.Append(plan.ForceNew.ElementsAs(ctx, &newFields, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(newFields) > 0 {
			planData, stateData := getPlanAndStateData(plan.Data.ValueString(), state.Data.ValueString(), &resp.Diagnostics)
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
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *RestAPIObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RestAPIObjectResourceModel
	var state RestAPIObjectResourceModel

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

	tflog.Debug(ctx, "Update routine called", map[string]interface{}{"object": plan})
	obj, err := makeAPIObject(ctx, r.providerData.client, plan.ID.ValueString(), &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	err = obj.UpdateObject(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating API Object",
			fmt.Sprintf("Could not update API object: %s", err.Error()),
		)

		// Read the current state from the server to get actual values after failed update
		readErr := obj.ReadObject(ctx)
		if readErr != nil {
			tflog.Error(ctx, "Failed to read object after failed update", map[string]interface{}{"error": readErr})
			// Continue with what we have - better to save something than nothing
		}

		// Update plan.Data to reflect the actual server state, not the desired state
		if apiResponse := obj.GetApiResponse(); len(apiResponse) > 0 {
			plan.Data = jsontypes.NewNormalizedValue(apiResponse)
		}
	}

	setResourceModelData(ctx, obj, &plan, &resp.Diagnostics)
	plan.CreateResponse = state.CreateResponse

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RestAPIObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RestAPIObjectResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Delete routine called", map[string]interface{}{"object": state})
	obj, err := makeAPIObject(ctx, r.providerData.client, state.ID.ValueString(), &state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	err = obj.DeleteObject(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting API Object",
			fmt.Sprintf("Could not delete API object: %s", err.Error()),
		)
		return
	}

}

// resourceRestAPIImport imports an existing API object into Terraform state.
// Since there is nothing in the ResourceData structure other
// than the "id" passed on the command line, we have to use an opinionated
// view of the API paths to figure out how to read that object
// from the API
func (r *RestAPIObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	input := req.ID

	// Remove leading and trailing slash if present
	input = strings.TrimPrefix(input, "/")
	input = strings.TrimSuffix(input, "/")

	n := strings.LastIndex(input, "/")
	if n == -1 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Invalid path to import api_object '%s' - must be /<full path from server root>/<object id>", req.ID),
		)
		return
	}

	data := RestAPIObjectResourceModel{
		ObjectID: types.StringValue(input[n+1:]),

		// Add leading slash back to path
		Path: types.StringValue(fmt.Sprintf("/%s", input[0:n])),

		// Troubleshooting is hard enough. Emit log messages so TF_LOG
		// has useful information in case an import isn't working
		Debug: types.BoolValue(true),

		ForceNew:        types.ListNull(types.StringType),
		IgnoreChangesTo: types.ListNull(types.StringType),
	}

	obj, err := makeAPIObject(ctx, r.providerData.client, "", &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Import routine called.", map[string]interface{}{"object": obj.String()})

	err = obj.ReadObject(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading API Object",
			fmt.Sprintf("Could not read API object: %s", err.Error()),
		)
		return
	}

	// For Import we want to write to state only what was observed from the server
	data.Data = jsontypes.NewNormalizedValue(obj.GetApiResponse())
	setResourceModelData(ctx, obj, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// makeAPIObject creates APIObjectOpts from the resource model
func makeAPIObject(ctx context.Context, client *apiclient.APIClient, id string, model *RestAPIObjectResourceModel) (*apiclient.APIObject, error) {
	tflog.Debug(ctx, "makeAPIObject routine called", map[string]interface{}{"id": id, "path": model.Path.ValueString()})

	opts := &apiclient.APIObjectOpts{
		Path:  model.Path.ValueString(),
		Data:  model.Data.ValueString(),
		Debug: model.Debug.ValueBool(),

		// Allow override of provider-level attributes
		IDAttribute: existingOrProviderOrDefaultString(model.IDAttribute, client.Opts.IDAttribute, ""),

		CreatePath:   existingOrDefaultString(model.CreatePath, ""),
		CreateMethod: existingOrProviderOrDefaultString(model.CreateMethod, client.Opts.CreateMethod, "POST"),

		ReadPath:   existingOrDefaultString(model.ReadPath, ""),
		ReadMethod: existingOrProviderOrDefaultString(model.ReadMethod, client.Opts.ReadMethod, "GET"),
		ReadData:   model.ReadData.ValueString(),

		UpdatePath:   existingOrDefaultString(model.UpdatePath, ""),
		UpdateMethod: existingOrProviderOrDefaultString(model.UpdateMethod, client.Opts.UpdateMethod, "PUT"),
		UpdateData:   model.UpdateData.ValueString(),

		DestroyPath:   existingOrDefaultString(model.DestroyPath, ""),
		DestroyMethod: existingOrProviderOrDefaultString(model.DestroyMethod, client.Opts.DestroyMethod, "DELETE"),
		DestroyData:   model.DestroyData.ValueString(),

		QueryString: existingOrDefaultString(model.QueryString, ""),
	}

	// Wire up read_search if configured
	if model.ReadSearch != nil {
		readSearch := make(map[string]string)
		if !model.ReadSearch.SearchKey.IsNull() && !model.ReadSearch.SearchKey.IsUnknown() {
			readSearch["search_key"] = model.ReadSearch.SearchKey.ValueString()
		}
		if !model.ReadSearch.SearchValue.IsNull() && !model.ReadSearch.SearchValue.IsUnknown() {
			readSearch["search_value"] = model.ReadSearch.SearchValue.ValueString()
		}
		if !model.ReadSearch.ResultsKey.IsNull() && !model.ReadSearch.ResultsKey.IsUnknown() {
			readSearch["results_key"] = model.ReadSearch.ResultsKey.ValueString()
		}
		if !model.ReadSearch.QueryString.IsNull() && !model.ReadSearch.QueryString.IsUnknown() {
			readSearch["query_string"] = model.ReadSearch.QueryString.ValueString()
		}
		if !model.ReadSearch.SearchData.IsNull() && !model.ReadSearch.SearchData.IsUnknown() {
			readSearch["search_data"] = model.ReadSearch.SearchData.ValueString()
		}
		if !model.ReadSearch.SearchPatch.IsNull() && !model.ReadSearch.SearchPatch.IsUnknown() {
			readSearch["search_patch"] = model.ReadSearch.SearchPatch.ValueString()
		}
		opts.ReadSearch = readSearch
	}

	// Allow user to specify the ID manually
	if !model.ObjectID.IsNull() && !model.ObjectID.IsUnknown() {
		opts.ID = model.ObjectID.ValueString()
	} else {
		// If not specified, use the terraform resource ID
		opts.ID = id
	}

	return apiclient.NewAPIObject(client, opts)
}

func setResourceModelData(ctx context.Context, obj *apiclient.APIObject, data *RestAPIObjectResourceModel, diag *diag.Diagnostics) {
	data.ID = types.StringValue(obj.ID)
	data.APIResponse = types.StringValue(obj.GetApiResponse())
	v, d := types.MapValueFrom(ctx, types.StringType, obj.GetApiData())
	data.APIData = v
	diag.Append(d...)
}
