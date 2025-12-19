package restapi

import (
	"context"
	"fmt"
	"strconv"

	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RestAPIObjectResource{}
var _ resource.ResourceWithImportState = &RestAPIObjectResource{}
var _ resource.ResourceWithModifyPlan = &RestAPIObjectResource{}

type RestAPIObjectResource struct {
	providerData *ProviderData
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
				/* TODO - see:
					https://developer.hashicorp.com/terraform/plugin/framework/handling-data/types/custom
					https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework-jsontypes
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != "" {
						data := make(map[string]interface{})
						err := json.Unmarshal([]byte(v), &data)
						if err != nil {
							errs = append(errs, fmt.Errorf("data attribute is invalid JSON: %v", err))
						}
					}
					return warns, errs
				},
				*/
			},
			"debug": schema.BoolAttribute{
				Description: "Whether to emit the HTTP request and response to STDOUT while working with the API object on the server.",
				Optional:    true,
			},
			"read_search": schema.StringAttribute{
				Description: "Custom search for `read_path`. This map will take `search_data`, `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation)",
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
				/* TODO
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != "" {
						data := make(map[string]interface{})
						err := json.Unmarshal([]byte(v), &data)
						if err != nil {
							errs = append(errs, fmt.Errorf("read_data attribute is invalid JSON: %v", err))
						}
					}
					return warns, errs
				},
				*/
			},
			"update_data": schema.StringAttribute{
				Optional:    true,
				Description: "Valid JSON object to pass during to update requests.",
				Sensitive:   isDataSensitive,
				/* TODO
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != "" {
						data := make(map[string]interface{})
						err := json.Unmarshal([]byte(v), &data)
						if err != nil {
							errs = append(errs, fmt.Errorf("update_data attribute is invalid JSON: %v", err))
						}
					}
					return warns, errs
				},
				*/
			},
			"destroy_data": schema.StringAttribute{
				Optional:    true,
				Description: "Valid JSON object to pass during to destroy requests.",
				Sensitive:   isDataSensitive,
				/* TODO
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != "" {
						data := make(map[string]interface{})
						err := json.Unmarshal([]byte(v), &data)
						if err != nil {
							errs = append(errs, fmt.Errorf("destroy_data attribute is invalid JSON: %v", err))
						}
					}
					return warns, errs
				},
				*/
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
	// Handle force_new attributes
}

func (r *RestAPIObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	/*
		obj, err := makeAPIObject(d, meta)
		if err != nil {
			return err
		}

		tflog.Debug(ctx, "Create routine called", map[string]interface{}{"object": obj.String()})

		err = obj.CreateObject(context.TODO())
		if err == nil {
			// Setting terraform ID tells terraform the object was created or it exists
			d.SetId(obj.ID)
			apiclient.SetResourceState(obj, d)
			// Only set during create for APIs that don't return sensitive data on subsequent retrieval
			d.Set("create_response", obj.APIResponse)
		}
		return err
	*/
}

func (r *RestAPIObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	/*
		obj, err := makeAPIObject(d, meta)
		if err != nil {
			if strings.Contains(err.Error(), "error parsing data provided") {
				tflog.Warn(ctx, "The data passed from Terraform's state is invalid!", map[string]interface{}{"error": err})
				tflog.Warn(ctx, "Continuing with partially constructed object...", nil)
			} else {
				return err
			}
		}

		tflog.Debug(ctx, "Read routine called", map[string]interface{}{"object": obj.String()})

		err = obj.ReadObject(ctx)
		if err == nil {
			// Setting terraform ID tells terraform the object was created or it exists
			tflog.Debug(ctx, "Read resource. Returned id is '%s'", map[string]interface{}{"id": obj.ID})
			d.SetId(obj.ID)

			apiclient.SetResourceState(obj, d)

			// Check whether the remote resource has changed.
			if !(d.Get("ignore_all_server_changes")).(bool) {
				ignoreList := []string{}
				v, ok := d.GetOk("ignore_changes_to")
				if ok {
					for _, s := range v.([]interface{}) {
						ignoreList = append(ignoreList, s.(string))
					}
				}

				// This checks if there were any changes to the remote resource that will need to be corrected
				// by comparing the current state with the response returned by the api.
				modifiedResource, hasDifferences := obj.GetDelta(ignoreList)

				if hasDifferences {
					tflog.Info(ctx, "Found differences in remote resource", nil)
					encoded, err := json.Marshal(modifiedResource)
					if err != nil {
						return err
					}
					jsonString := string(encoded)
					d.Set("data", jsonString)
				}
			}

		}
		return err
	*/
}

func (r *RestAPIObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	/*
		obj, err := makeAPIObject(d, meta)
		if err != nil {
			d.Partial(true)
			return err
		}

		// If copy_keys is not empty, we have to grab the latest
		// data so we can copy anything needed before the update
		client := meta.(*apiclient.APIClient)
		if client.CopyKeysEnabled() {
			err = obj.ReadObject(ctx)
			if err != nil {
				return err
			}
		}

		tflog.Debug(ctx, "Update routine called", map[string]interface{}{"object": obj.String()})

		err = obj.UpdateObject(ctx)
		if err == nil {
			apiclient.SetResourceState(obj, d)
		} else {
			d.Partial(true)
		}
		return err
	*/
}

func (r *RestAPIObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	/*
		ctx := context.TODO()
		obj, err := makeAPIObject(d, meta)
		if err != nil {
			return err
		}
		tflog.Debug(ctx, "Delete routine called. Object built", map[string]interface{}{"object": obj.String()})

		err = obj.DeleteObject(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "404") {
				// 404 means it doesn't exist. Call that good enough
				// TODO: Coax this to a specific error type we can check for instead of magic strings
				err = nil
			}
		}
		return err
	*/
}

// resourceRestAPIImport imports an existing API object into Terraform state.
// Since there is nothing in the ResourceData structure other
// than the "id" passed on the command line, we have to use an opinionated
// view of the API paths to figure out how to read that object
// from the API
func (r *RestAPIObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	/*
		input := d.Id()

		hasTrailingSlash := strings.HasSuffix(input, "/")
		var n int
		if hasTrailingSlash {
			n = strings.LastIndex(input[0:len(input)-1], "/")
		} else {
			n = strings.LastIndex(input, "/")
		}

		if n == -1 {
			return imported, fmt.Errorf("invalid path to import api_object '%s' - must be /<full path from server root>/<object id>", input)
		}

		path := input[0:n]
		d.Set("path", path)

		var id string
		if hasTrailingSlash {
			id = input[n+1 : len(input)-1]
		} else {
			id = input[n+1:]
		}

		d.Set("data", fmt.Sprintf(`{ "id": "%s" }`, id))
		d.SetId(id)

		// Troubleshooting is hard enough. Emit log messages so TF_LOG
		// has useful information in case an import isn't working
		d.Set("debug", true)

		obj, err := makeAPIObject(d, meta)
		if err != nil {
			return imported, err
		}
		tflog.Debug(ctx, "Import routine called.", map[string]interface{}{"object": obj.String()})

		err = obj.ReadObject(context.TODO())
		if err == nil {
			apiclient.SetResourceState(obj, d)
			// Data that we set in the state above must be passed along
			// as an item in the stack of imported data
			imported = append(imported, d)
		}

		return imported, err
	*/
}

/*
func buildAPIObjectOpts(d *schema.ResourceData) (*apiclient.APIObjectOpts, error) {
	ctx := context.TODO()
	opts := &apiclient.APIObjectOpts{
		Path: d.Get("path").(string),
	}

	// Allow user to override provider-level id_attribute
	if v, ok := d.GetOk("id_attribute"); ok {
		opts.IDAttribute = v.(string)
	}

	// Allow user to specify the ID manually
	if v, ok := d.GetOk("object_id"); ok {
		opts.ID = v.(string)
	} else {
		// If not specified, see if terraform has an ID
		opts.ID = d.Id()
	}

	tflog.Debug(ctx, "buildAPIObjectOpts routine called for id", map[string]interface{}{"id": opts.ID})

	if v, ok := d.GetOk("create_path"); ok {
		opts.PostPath = v.(string)
	}
	if v, ok := d.GetOk("read_path"); ok {
		opts.GetPath = v.(string)
	}
	if v, ok := d.GetOk("update_path"); ok {
		opts.PutPath = v.(string)
	}
	if v, ok := d.GetOk("create_method"); ok {
		opts.CreateMethod = v.(string)
	}
	if v, ok := d.GetOk("read_method"); ok {
		opts.ReadMethod = v.(string)
	}
	if v, ok := d.GetOk("read_data"); ok {
		opts.ReadData = v.(string)
	}
	if v, ok := d.GetOk("update_method"); ok {
		opts.UpdateMethod = v.(string)
	}
	if v, ok := d.GetOk("update_data"); ok {
		opts.UpdateData = v.(string)
	}
	if v, ok := d.GetOk("destroy_method"); ok {
		opts.DestroyMethod = v.(string)
	}
	if v, ok := d.GetOk("destroy_data"); ok {
		opts.DestroyData = v.(string)
	}
	if v, ok := d.GetOk("destroy_path"); ok {
		opts.DestroyPath = v.(string)
	}
	if v, ok := d.GetOk("query_string"); ok {
		opts.QueryString = v.(string)
	}

	readSearch := expandReadSearch(d.Get("read_search").(map[string]interface{}))
	opts.ReadSearch = readSearch

	opts.Data = d.Get("data").(string)
	opts.Debug = d.Get("debug").(bool)

	return opts, nil
}

func expandReadSearch(v map[string]interface{}) (readSearch map[string]string) {
	readSearch = make(map[string]string)
	for key, val := range v {
		readSearch[key] = val.(string)
	}

	return
}
*/
