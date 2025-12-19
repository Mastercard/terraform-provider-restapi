package restapi

import (
	"context"
	"strconv"

	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RestAPIObjectDataSource struct {
	providerData *ProviderData
}

func NewRestAPIObjectDataSource() datasource.DataSource {
	return &RestAPIObjectDataSource{}
}

func (r *RestAPIObjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *RestAPIObjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
		}, // End schema
	}
}

func (r *RestAPIObjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	/*

		path := d.Get("path").(string)
		searchPath := d.Get("search_path").(string)
		queryString := d.Get("query_string").(string)
		debug := d.Get("debug").(bool)
		client := meta.(*apiclient.APIClient)

		tflog.Debug(ctx, "Data routine called.", map[string]interface{}{})

		readQueryString := d.Get("read_query_string").(string)
		if readQueryString == "not-set" {
			readQueryString = queryString
		}

		searchKey := d.Get("search_key").(string)
		searchValue := d.Get("search_value").(string)
		searchData := d.Get("search_data").(string)
		resultsKey := d.Get("results_key").(string)
		idAttribute := d.Get("id_attribute").(string)

		send := ""
		if len(searchData) > 0 {
			tmpData, _ := json.Marshal(searchData)
			send = string(tmpData)
			tflog.Debug(ctx, "Using search data", map[string]interface{}{"data": send})
		}

		tflog.Debug(ctx, "Data parameters", map[string]interface{}{
			"path":         path,
			"search_path":  searchPath,
			"query_string": queryString,
			"search_key":   searchKey,
			"search_value": searchValue,
			"results_key":  resultsKey,
			"id_attribute": idAttribute,
		})

		opts := &apiclient.APIObjectOpts{
			Path:        path,
			SearchPath:  searchPath,
			Debug:       debug,
			QueryString: readQueryString,
			IDAttribute: idAttribute,
		}

		obj, err := apiclient.NewAPIObject(client, opts)
		if err != nil {
			return err
		}

		if _, err := obj.FindObject(context.TODO(), queryString, searchKey, searchValue, resultsKey, send); err != nil {
			return err
		}

		// Back to terraform-specific stuff. Create an api_object with the ID and refresh it object
		tflog.Debug(ctx, "Attempting to construct api_object to refresh data", map[string]interface{}{})

		d.SetId(obj.ID)

		err = obj.ReadObject(context.TODO())
		if err == nil {
			// Setting terraform ID tells terraform the object was created or it exists
			tflog.Debug(ctx, "Data resource. Returned id is '%s'", map[string]interface{}{"id": obj.ID})
			d.SetId(obj.ID)
			apiclient.SetResourceState(obj, d)
		}
		return err
	*/
}
