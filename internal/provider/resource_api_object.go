package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	restapi "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	ReadSearch             ReadSearchModel      `tfsdk:"read_search"`
	QueryString            types.String         `tfsdk:"query_string"`
	ForceNew               types.List           `tfsdk:"force_new"`
	ReadData               jsontypes.Normalized `tfsdk:"read_data"`
	UpdateData             jsontypes.Normalized `tfsdk:"update_data"`
	DestroyData            jsontypes.Normalized `tfsdk:"destroy_data"`
	IgnoreChangesTo        types.List           `tfsdk:"ignore_changes_to"`
	IgnoreAllServerChanges types.Bool           `tfsdk:"ignore_all_server_changes"`

	ID             types.String `tfsdk:"id"`
	APIData        types.Map    `tfsdk:"api_data"`
	APIResponse    types.String `tfsdk:"api_response"`
	CreateResponse types.String `tfsdk:"create_response"`
}

type ReadSearchModel struct {
	SearchData  jsontypes.Normalized `tfsdk:"search_data"`
	SearchKey   types.String         `tfsdk:"search_key"`
	SearchValue types.String         `tfsdk:"search_value"`
	ResultsKey  types.String         `tfsdk:"results_key"`
	QueryString types.String         `tfsdk:"query_string"`
}

func NewRestAPIObjectResource() resource.Resource {
	return &RestAPIObjectResource{}
}

func (r *RestAPIObjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *RestAPIObjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Consider data sensitive if env variables is set to true.
	isDataSensitive, _ := strconv.ParseBool(apiclient.GetEnvOrDefault("API_DATA_IS_SENSITIVE", "false"))

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
			"create_response": schema.StringAttribute{
				Description: "The raw body of the HTTP response returned when creating the object.",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"force_new": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				// TODO: Add plan modifier
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
		Blocks: map[string]schema.Block{
			"read_search": schema.SingleNestedBlock{
				Description: "Custom search for `read_path`. This map will take `search_data`, `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation)",
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
						Description: "The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.",
						Required:    true,
					},
					"results_key": schema.StringAttribute{
						Description: "When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are already an array and are to be used exactly as-is.",
						Optional:    true,
					},
				},
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

func (r *RestAPIObjectResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// TODO: Handle force_new attributes
	// TODO: Handle ignore_changes_to attributes
	/*
		// If copy_keys is not empty, we have to grab the latest
		// data so we can copy anything needed before the update
		client := meta.(*apiclient.APIClient)
		if client.CopyKeysEnabled() {
			err = obj.ReadObject(ctx)
			if err != nil {
				return err
			}
		}
	*/
}

func (r *RestAPIObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestAPIObjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Create routine called", map[string]interface{}{"object": data})
	obj, err := makeAPIObject(ctx, r.providerData.client, "", &data)
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

	data.ID = types.StringValue(obj.ID)
	data.CreateResponse = types.StringValue(obj.GetApiResponse())
	setResourceModelData(ctx, obj, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestAPIObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestAPIObjectResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Read routine called", map[string]interface{}{"object": data})
	obj, err := makeAPIObject(ctx, r.providerData.client, "", &data)
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

	// Setting terraform ID tells terraform the object was created or it exists
	tflog.Debug(ctx, "Read resource. Returned id is '%s'", map[string]interface{}{"id": obj.ID})
	data.ID = types.StringValue(obj.ID)
	setResourceModelData(ctx, obj, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestAPIObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestAPIObjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Update routine called", map[string]interface{}{"object": data})
	obj, err := makeAPIObject(ctx, r.providerData.client, "", &data)
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
		// NOTE: no return here - we want to set the partial state below
	}

	setResourceModelData(ctx, obj, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RestAPIObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestAPIObjectResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Delete routine called", map[string]interface{}{"object": data})
	obj, err := makeAPIObject(ctx, r.providerData.client, "", &data)
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
		ID: types.StringValue(input[n+1:]),

		// Add leading slash back to path
		Path: types.StringValue(fmt.Sprintf("/%s", input[0:n])),

		// Troubleshooting is hard enough. Emit log messages so TF_LOG
		// has useful information in case an import isn't working
		Debug: types.BoolValue(true),
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
			"Error UpdReading API Object",
			fmt.Sprintf("Could not read API object: %s", err.Error()),
		)
		return
	}

	setResourceModelData(ctx, obj, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// makeAPIObject creates APIObjectOpts from the resource model
func makeAPIObject(ctx context.Context, client *apiclient.APIClient, id string, model *RestAPIObjectResourceModel) (*apiclient.APIObject, error) {
	tflog.Debug(ctx, "buildAPIObjectOpts routine called", map[string]interface{}{"id": id, "path": id})

	opts := &apiclient.APIObjectOpts{
		Path:  model.Path.ValueString(),
		Data:  model.Data.ValueString(),
		Debug: model.Debug.ValueBool(),

		// Allow override of provider-level attributes
		IDAttribute: existingOrProviderOrDefaultString("id_attribute", model.IDAttribute, client.Opts.IDAttribute, ""),

		CreatePath:   existingOrDefaultString("create_path", model.CreatePath, ""),
		CreateMethod: existingOrProviderOrDefaultString("create_method", model.CreateMethod, client.Opts.CreateMethod, "POST"),

		ReadPath:   existingOrDefaultString("read_path", model.ReadPath, "{id}"),
		ReadMethod: existingOrProviderOrDefaultString("read_method", model.ReadMethod, client.Opts.ReadMethod, "GET"),
		ReadData:   model.ReadData.ValueString(),

		UpdatePath:   existingOrDefaultString("update_path", model.UpdatePath, "{id}"),
		UpdateMethod: existingOrProviderOrDefaultString("update_method", model.UpdateMethod, client.Opts.UpdateMethod, "PUT"),
		UpdateData:   model.UpdateData.ValueString(),

		DestroyPath:   existingOrDefaultString("destroy_path", model.DestroyPath, "{id}"),
		DestroyMethod: existingOrProviderOrDefaultString("destroy_method", model.DestroyMethod, client.Opts.DestroyMethod, "DELETE"),
		DestroyData:   model.DestroyData.ValueString(),

		//TODO: Update readsearch implementation
		//ReadSearch:             model.ReadSearch.ValueString(),
		QueryString: existingOrDefaultString("query_string", model.QueryString, ""),
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
	data.APIResponse = types.StringValue(obj.GetApiResponse())
	v, d := types.MapValueFrom(ctx, types.StringType, obj.GetApiData())
	data.APIData = v
	diag.Append(d...)
}
