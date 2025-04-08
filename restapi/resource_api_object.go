package restapi

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRestAPI() *schema.Resource {
	// Consider data sensitive if env variables is set to true.
	isDataSensitive, _ := strconv.ParseBool(GetEnvOrDefault("API_DATA_IS_SENSITIVE", "false"))

	return &schema.Resource{
		Create: resourceRestAPICreate,
		Read:   resourceRestAPIRead,
		Update: resourceRestAPIUpdate,
		Delete: resourceRestAPIDelete,
		Exists: resourceRestAPIExists,

		Description: "Acting as a wrapper of cURL, this object supports POST, GET, PUT and DELETE on the specified url",

		Importer: &schema.ResourceImporter{
			State: resourceRestAPIImport,
		},

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
				Required:    true,
			},
			"create_path": {
				Type:        schema.TypeString,
				Description: "Defaults to `path`. The API path that represents where to CREATE (POST) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object if the data contains the `id_attribute`.",
				Optional:    true,
			},
			"read_path": {
				Type:        schema.TypeString,
				Description: "Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"update_path": {
				Type:        schema.TypeString,
				Description: "Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"create_method": {
				Type:        schema.TypeString,
				Description: "Defaults to `create_method` set on the provider. Allows per-resource override of `create_method` (see `create_method` provider config documentation)",
				Optional:    true,
			},
			"read_method": {
				Type:        schema.TypeString,
				Description: "Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation)",
				Optional:    true,
			},
			"update_method": {
				Type:        schema.TypeString,
				Description: "Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation). Set to `PATCH` for Midpoint integration to enable calculating changes and sending them in Midpoint's ObjectModificationType format.",
				Optional:    true,
			},
			"destroy_method": {
				Type:        schema.TypeString,
				Description: "Defaults to `destroy_method` set on the provider. Allows per-resource override of `destroy_method` (see `destroy_method` provider config documentation)",
				Optional:    true,
			},
			"destroy_path": {
				Type:        schema.TypeString,
				Description: "Defaults to `path/{id}`. The API path that represents where to DESTROY (DELETE) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"id_attribute": {
				Type:        schema.TypeString,
				Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
				Optional:    true,
			},
			"object_id": {
				Type:        schema.TypeString,
				Description: "Defaults to the id learned by the provider during normal operations and `id_attribute`. Allows you to set the id manually. This is used in conjunction with the `*_path` attributes.",
				Optional:    true,
			},
			"data": {
				Type:        schema.TypeString,
				Description: "Valid JSON object that this provider will manage with the API server.",
				Required:    true,
				Sensitive:   isDataSensitive,
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
			},
			"debug": {
				Type:        schema.TypeBool,
				Description: "Whether to emit verbose debug output while working with the API object on the server.",
				Optional:    true,
			},
			"read_search": {
				Type:        schema.TypeMap,
				Description: "Custom search for `read_path`. This map will take `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation)",
				Optional:    true,
			},
			"query_string": {
				Type:        schema.TypeString,
				Description: "Query string to be included in the path",
				Optional:    true,
			},
			"api_data": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"api_response": {
				Type:        schema.TypeString,
				Description: "The raw body of the HTTP response from the last read of the object.",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"create_response": {
				Type:        schema.TypeString,
				Description: "The raw body of the HTTP response returned when creating the object.",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"force_new": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				ForceNew:    true,
				Description: "Any changes to these values will result in recreating the resource instead of updating.",
			},
			"read_data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Valid JSON object to pass during read requests.",
				Sensitive:   isDataSensitive,
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
			},
			"update_data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Valid JSON object to pass during to update requests.",
				Sensitive:   isDataSensitive,
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
			},
			"destroy_data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Valid JSON object to pass during to destroy requests.",
				Sensitive:   isDataSensitive,
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
			},
			"filter_keys": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "A list of keys to filter out when parsing the API response. These keys will be removed from the state at any level in the JSON hierarchy.",
			},
		}, /* End schema */

	}
}

/*
Since there is nothing in the ResourceData structure other

	than the "id" passed on the command line, we have to use an opinionated
	view of the API paths to figure out how to read that object
	from the API
*/
func resourceRestAPIImport(d *schema.ResourceData, meta interface{}) (imported []*schema.ResourceData, err error) {
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

	/* Troubleshooting is hard enough. Emit log messages so TF_LOG
	   has useful information in case an import isn't working */
	d.Set("debug", true)

	obj, err := makeAPIObject(d, meta)
	if err != nil {
		return imported, err
	}
	log.Printf("resource_api_object.go: Import routine called. Object built:\n%s\n", obj.toString())

	err = obj.readObject()
	if err == nil {
		setResourceState(obj, d)
		/* Data that we set in the state above must be passed along
		   as an item in the stack of imported data */
		imported = append(imported, d)
	}

	return imported, err
}

func resourceRestAPICreate(d *schema.ResourceData, meta interface{}) error {
	obj, err := makeAPIObject(d, meta)
	if err != nil {
		return err
	}
	log.Printf("resource_api_object.go: Create routine called. Object built:\n%s\n", obj.toString())

	err = obj.createObject()
	if err == nil {
		/* Setting terraform ID tells terraform the object was created or it exists */
		d.SetId(obj.id)
		setResourceState(obj, d)
		/* Only set during create for APIs that don't return sensitive data on subsequent retrieval */
		d.Set("create_response", obj.apiResponse)
	}
	return err
}

func resourceRestAPIRead(d *schema.ResourceData, meta interface{}) error {
	obj, err := makeAPIObject(d, meta)
	if err != nil {
		if strings.Contains(err.Error(), "error parsing data provided") {
			log.Printf("resource_api_object.go: WARNING! The data passed from Terraform's state is invalid! %v", err)
			log.Printf("resource_api_object.go: Continuing with partially constructed object...")
		} else {
			return err
		}
	}
	log.Printf("resource_api_object.go: Read routine called. Object built:\n%s\n", obj.toString())

	err = obj.readObject()
	if err == nil {
		/* Setting terraform ID tells terraform the object was created or it exists */
		log.Printf("resource_api_object.go: Read resource. Returned id is '%s'\n", obj.id)
		d.SetId(obj.id)
		setResourceState(obj, d)
	}
	return err
}

func resourceRestAPIUpdate(d *schema.ResourceData, meta interface{}) error {
	obj, err := makeAPIObject(d, meta)
	if err != nil {
		d.Partial(true)
		return err
	}

	/* If copy_keys is not empty, we have to grab the latest
	   data so we can copy anything needed before the update */
	client := meta.(*APIClient)
	if len(client.copyKeys) > 0 {
		err = obj.readObject()
		if err != nil {
			return err
		}
	}

	log.Printf("resource_api_object.go: Update routine called. Object built:\n%s\n", obj.toString())

	err = obj.updateObject()
	if err == nil {
		setResourceState(obj, d)
	} else {
		d.Partial(true)
	}
	return err
}

func resourceRestAPIDelete(d *schema.ResourceData, meta interface{}) error {
	obj, err := makeAPIObject(d, meta)
	if err != nil {
		return err
	}
	log.Printf("resource_api_object.go: Delete routine called. Object built:\n%s\n", obj.toString())

	err = obj.deleteObject()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			/* 404 means it doesn't exist. Call that good enough */
			err = nil
		}
	}
	return err
}

func resourceRestAPIExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	obj, err := makeAPIObject(d, meta)
	if err != nil {
		if strings.Contains(err.Error(), "error parsing data provided") {
			log.Printf("resource_api_object.go: WARNING! The data passed from Terraform's state is invalid! %v", err)
			log.Printf("resource_api_object.go: Continuing with partially constructed object...")
		} else {
			return exists, err
		}
	}
	log.Printf("resource_api_object.go: Exists routine called. Object built: %s\n", obj.toString())

	/* Assume all errors indicate the object just doesn't exist.
	This may not be a good assumption... */
	err = obj.readObject()
	if err == nil {
		exists = true
	}
	return exists, err
}

/*
Simple helper routine to build an api_object struct

	for the various calls terraform will use. Unfortunately,
	terraform cannot just reuse objects, so each CRUD operation
	results in a new object created
*/
func makeAPIObject(d *schema.ResourceData, meta interface{}) (*APIObject, error) {
	opts, err := buildAPIObjectOpts(d)
	if err != nil {
		return nil, err
	}

	caller := "unknown"
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		parts := strings.Split(details.Name(), ".")
		caller = parts[len(parts)-1]
	}
	log.Printf("resource_rest_api.go: Constructing new APIObject in makeAPIObject (called by %s)", caller)

	obj, err := NewAPIObject(meta.(*APIClient), opts)

	return obj, err
}

func buildAPIObjectOpts(d *schema.ResourceData) (*apiObjectOpts, error) {
	opts := &apiObjectOpts{
		path: d.Get("path").(string),
	}

	/* Allow user to override provider-level id_attribute */
	if v, ok := d.GetOk("id_attribute"); ok {
		opts.idAttribute = v.(string)
	}

	/* Allow user to specify the ID manually */
	if v, ok := d.GetOk("object_id"); ok {
		opts.id = v.(string)
	} else {
		/* If not specified, see if terraform has an ID */
		opts.id = d.Id()
	}

	log.Printf("resource_rest_api.go: buildAPIObjectOpts routine called for id '%s'\n", opts.id)

	if v, ok := d.GetOk("create_path"); ok {
		opts.postPath = v.(string)
	}
	if v, ok := d.GetOk("read_path"); ok {
		opts.getPath = v.(string)
	}
	if v, ok := d.GetOk("update_path"); ok {
		opts.putPath = v.(string)
	}
	if v, ok := d.GetOk("create_method"); ok {
		opts.createMethod = v.(string)
	}
	if v, ok := d.GetOk("read_method"); ok {
		opts.readMethod = v.(string)
	}
	if v, ok := d.GetOk("read_data"); ok {
		opts.readData = v.(string)
	}
	if v, ok := d.GetOk("update_method"); ok {
		opts.updateMethod = v.(string)
	}
	if v, ok := d.GetOk("update_data"); ok {
		opts.updateData = v.(string)
	}
	if v, ok := d.GetOk("destroy_method"); ok {
		opts.destroyMethod = v.(string)
	}
	if v, ok := d.GetOk("destroy_data"); ok {
		opts.destroyData = v.(string)
	}
	if v, ok := d.GetOk("destroy_path"); ok {
		opts.deletePath = v.(string)
	}
	if v, ok := d.GetOk("query_string"); ok {
		opts.queryString = v.(string)
	}

	readSearch := expandReadSearch(d.Get("read_search").(map[string]interface{}))
	opts.readSearch = readSearch

	// Set filter_keys if provided
	if v, ok := d.GetOk("filter_keys"); ok {
		filterKeys := make([]string, 0)
		for _, key := range v.([]interface{}) {
			filterKeys = append(filterKeys, key.(string))
		}
		opts.filterKeys = filterKeys
	}

	opts.data = d.Get("data").(string)
	opts.debug = d.Get("debug").(bool)

	return opts, nil
}

func expandReadSearch(v map[string]interface{}) (readSearch map[string]string) {
	readSearch = make(map[string]string)
	for key, val := range v {
		readSearch[key] = val.(string)
	}

	return
}
