package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type RestAPIObjectDataSource struct {
	providerData *ProviderData
}

type RestAPIObjectDataSourceModel struct {
	Path            types.String         `tfsdk:"path"`
	SearchPath      types.String         `tfsdk:"search_path"`
	QueryString     types.String         `tfsdk:"query_string"`
	ReadQueryString types.String         `tfsdk:"read_query_string"`
	SearchData      jsontypes.Normalized `tfsdk:"search_data"`
	SearchKey       types.String         `tfsdk:"search_key"`
	SearchValue     types.String         `tfsdk:"search_value"`
	ResultsKey      types.String         `tfsdk:"results_key"`
	IDAttribute     types.String         `tfsdk:"id_attribute"`
	Debug           types.Bool           `tfsdk:"debug"`
	ID              types.String         `tfsdk:"id"`
	APIData         types.Map            `tfsdk:"api_data"`
	APIResponse     types.String         `tfsdk:"api_response"`
}

func NewRestAPIObjectDataSource() datasource.DataSource {
	return &RestAPIObjectDataSource{}
}

func (r *RestAPIObjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *RestAPIObjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"search_path": schema.StringAttribute{
				Description: "The API path on top of the base URL set in the provider that represents the location to search for objects of this type on the API server. If not set, defaults to the value of path.",
				Optional:    true,
			},
			"query_string": schema.StringAttribute{
				Description: "An optional query string to send when performing the search.",
				Optional:    true,
			},
			"read_query_string": schema.StringAttribute{
				Description: "Defaults to `query_string` set on data source. This key allows setting a different or empty query string for reading the object.",
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
			"id_attribute": schema.StringAttribute{
				Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
				Optional:    true,
			},
			"debug": schema.BoolAttribute{
				Description: "Whether to emit verbose debug output while working with the API object on the server.",
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
			"id": schema.StringAttribute{
				Description: "The ID of the object.",
				Computed:    true,
			},
		}, // End schema
	}
}

func (r *RestAPIObjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *RestAPIObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state RestAPIObjectDataSourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	queryString := existingOrDefaultString(state.QueryString, "")
	searchKey := existingOrDefaultString(state.SearchKey, "")
	searchValue := existingOrDefaultString(state.SearchValue, "")
	resultsKey := existingOrDefaultString(state.ResultsKey, "")

	send := ""
	client := r.providerData.client

	tflog.Debug(ctx, "Read routine called", map[string]interface{}{"object": state})

	opts := &apiclient.APIObjectOpts{
		Path:        state.Path.ValueString(),
		SearchPath:  state.SearchPath.ValueString(),
		Debug:       state.Debug.ValueBool(),
		QueryString: queryString,
		IDAttribute: existingOrProviderOrDefaultString(state.IDAttribute, client.Opts.IDAttribute, ""),
	}

	// If we have a read_query_string, we will use that in the API Object since the
	// query_string parameter in the datasource is used for searching
	if !state.ReadQueryString.IsNull() && !state.ReadQueryString.IsUnknown() {
		opts.QueryString = state.ReadQueryString.ValueString()
	}

	if !state.SearchData.IsNull() && !state.SearchData.IsUnknown() {
		send = state.SearchData.ValueString()
	}

	obj, err := apiclient.NewAPIObject(client, opts)
	if err != nil {
		tflog.Error(ctx, "Error creating API object", map[string]interface{}{"error": err})
		resp.Diagnostics.AddError(
			"Error Creating API Object",
			fmt.Sprintf("Could not create API object: %s", err.Error()),
		)
		return
	}

	_, err = obj.FindObject(ctx, queryString, searchKey, searchValue, resultsKey, send)
	if err != nil {
		tflog.Error(ctx, "Error finding API object", map[string]interface{}{"error": err})
		resp.Diagnostics.AddError(
			"Error Finding API Object",
			fmt.Sprintf("Could not find API object: %s", err.Error()),
		)
		return
	}

	// Found - read it to populate all data
	err = obj.ReadObject(ctx)
	if err != nil {
		tflog.Error(ctx, "Error reading API object", map[string]interface{}{"error": err})
		resp.Diagnostics.AddError(
			"Error Reading API Object",
			fmt.Sprintf("Could not read API object: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Found object", map[string]interface{}{
		"id":           obj.ID,
		"api_data":     obj.GetApiData(),
		"api_response": obj.GetApiResponse(),
	})

	state.ID = types.StringValue(obj.ID)
	state.APIResponse = types.StringValue(obj.GetApiResponse())
	v, d := types.MapValueFrom(ctx, types.StringType, obj.GetApiData())
	state.APIData = v
	resp.Diagnostics.Append(d...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
