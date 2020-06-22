package restapi

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRestApi() *schema.Resource {
	// Consider data sensitive if env variables is set to true.
	is_data_sensitive, _ := strconv.ParseBool(GetEnvOrDefault("API_DATA_IS_SENSITIVE", "false"))

	return &schema.Resource{
		Create: resourceRestApiCreate,
		Read:   resourceRestApiRead,
		Update: resourceRestApiUpdate,
		Delete: resourceRestApiDelete,
		Exists: resourceRestApiExists,

		Importer: &schema.ResourceImporter{
			State: resourceRestApiImport,
		},

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
				Required:    true,
			},
			"create_path": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `path`. The API path that represents where to CREATE (POST) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object if the data contains the `id_attribute`.",
				Optional:    true,
			},
			"read_path": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `path/{id}`. The API path that represents where to READ (GET) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"update_path": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `path/{id}`. The API path that represents where to UPDATE (PUT) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"create_method": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `create_method` set on the provider. Allows per-resource override of `create_method` (see `create_method` provider config documentation)",
				Optional:    true,
			},
			"read_method": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `read_method` set on the provider. Allows per-resource override of `read_method` (see `read_method` provider config documentation)",
				Optional:    true,
			},
			"update_method": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `update_method` set on the provider. Allows per-resource override of `update_method` (see `update_method` provider config documentation)",
				Optional:    true,
			},
			"destroy_method": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `destroy_method` set on the provider. Allows per-resource override of `destroy_method` (see `destroy_method` provider config documentation)",
				Optional:    true,
			},
			"destroy_path": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `path/{id}`. The API path that represents where to DESTROY (DELETE) objects of this type on the API server. The string `{id}` will be replaced with the terraform ID of the object.",
				Optional:    true,
			},
			"id_attribute": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
				Optional:    true,
			},
			"object_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to the id learned by the provider during normal operations and `id_attribute`. Allows you to set the id manually. This is used in conjunction with the `*_path` attributes.",
				Optional:    true,
			},
			"data": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Valid JSON data that this provider will manage with the API server.",
				Required:    true,
				Sensitive:   is_data_sensitive,
			},
			"debug": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Whether to emit verbose debug output while working with the API object on the server.",
				Optional:    true,
			},
			"read_search": &schema.Schema{
				Type:        schema.TypeMap,
				Description: "Custom search for `read_path`. This map will take `search_key`, `search_value`, `results_key` and `query_string` (see datasource config documentation)",
				Optional:    true,
			},
			"api_data": &schema.Schema{
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).",
				Computed:    true,
			},
			"api_response": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The raw body of the HTTP response from the last read of the object.",
				Computed:    true,
			},
			"create_response": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The raw body of the HTTP response returned when creating the object.",
				Computed:    true,
			},
			"force_new": &schema.Schema{
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				ForceNew:    true,
				Description: "Any changes to these values will result in recreating the resource instead of updating.",
			},
		}, /* End schema */

	}
}

/* Since there is nothing in the ResourceData structure other
   than the "id" passed on the command line, we have to use an opinionated
   view of the API paths to figure out how to read that object
   from the API */
func resourceRestApiImport(d *schema.ResourceData, meta interface{}) (imported []*schema.ResourceData, err error) {
	input := d.Id()

	hasTrailingSlash := strings.LastIndex(input, "/") == len(input)-1
	var n int
	if hasTrailingSlash {
		n = strings.LastIndex(input[0:len(input)-1], "/")
	} else {
		n = strings.LastIndex(input, "/")
	}

	if n == -1 {
		return imported, fmt.Errorf("Invalid path to import api_object '%s'. Must be /<full path from server root>/<object id>", input)
	}

	path := input[0:n]
	d.Set("path", path)

	var id string
	if hasTrailingSlash {
		id = input[n+1 : len(input)-1]
	} else {
		id = input[n+1 : len(input)]
	}

	d.Set("data", fmt.Sprintf(`{ "id": "%s" }`, id))
	d.SetId(id)

	/* Troubleshooting is hard enough. Emit log messages so TF_LOG
	   has useful information in case an import isn't working */
	d.Set("debug", true)

	obj, err := make_api_object(d, meta)
	if err != nil {
		return imported, err
	}
	log.Printf("resource_api_object.go: Import routine called. Object built:\n%s\n", obj.toString())

	err = obj.read_object()
	if err == nil {
		set_resource_state(obj, d)
		/* Data that we set in the state above must be passed along
		   as an item in the stack of imported data */
		imported = append(imported, d)
	}

	return imported, err
}

func resourceRestApiCreate(d *schema.ResourceData, meta interface{}) error {
	obj, err := make_api_object(d, meta)
	if err != nil {
		return err
	}
	log.Printf("resource_api_object.go: Create routine called. Object built:\n%s\n", obj.toString())

	err = obj.create_object()
	if err == nil {
		/* Setting terraform ID tells terraform the object was created or it exists */
		d.SetId(obj.id)
		set_resource_state(obj, d)
		/* Only set during create for APIs that don't return sensitive data on subsequent retrieval */
		d.Set("create_response", obj.api_response)
	}
	return err
}

func resourceRestApiRead(d *schema.ResourceData, meta interface{}) error {
	obj, err := make_api_object(d, meta)
	if err != nil {
		return err
	}
	log.Printf("resource_api_object.go: Read routine called. Object built:\n%s\n", obj.toString())

	err = obj.read_object()
	if err == nil {
		/* Setting terraform ID tells terraform the object was created or it exists */
		log.Printf("resource_api_object.go: Read resource. Returned id is '%s'\n", obj.id)
		d.SetId(obj.id)
		set_resource_state(obj, d)
	}
	return err
}

func resourceRestApiUpdate(d *schema.ResourceData, meta interface{}) error {
	obj, err := make_api_object(d, meta)
	if err != nil {
		return err
	}

	/* If copy_keys is not empty, we have to grab the latest
	   data so we can copy anything needed before the update */
	client := meta.(*api_client)
	if len(client.copy_keys) > 0 {
		err = obj.read_object()
		if err != nil {
			return err
		}
	}

	log.Printf("resource_api_object.go: Update routine called. Object built:\n%s\n", obj.toString())

	err = obj.update_object()
	if err == nil {
		set_resource_state(obj, d)
	}
	return err
}

func resourceRestApiDelete(d *schema.ResourceData, meta interface{}) error {
	obj, err := make_api_object(d, meta)
	if err != nil {
		return err
	}
	log.Printf("resource_api_object.go: Delete routine called. Object built:\n%s\n", obj.toString())

	err = obj.delete_object()
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			/* 404 means it doesn't exist. Call that good enough */
			err = nil
		}
	}
	return err
}

func resourceRestApiExists(d *schema.ResourceData, meta interface{}) (exists bool, err error) {
	obj, err := make_api_object(d, meta)
	if err != nil {
		return exists, err
	}
	log.Printf("resource_api_object.go: Exists routine called. Object built: %s\n", obj.toString())

	/* Assume all errors indicate the object just doesn't exist.
	This may not be a good assumption... */
	err = obj.read_object()
	if err == nil {
		exists = true
	}
	return exists, err
}

/* Simple helper routine to build an api_object struct
   for the various calls terraform will use. Unfortunately,
   terraform cannot just reuse objects, so each CRUD operation
   results in a new object created */
func make_api_object(d *schema.ResourceData, meta interface{}) (*api_object, error) {
	opts, err := buildApiObjectOpts(d)
	if err != nil {
		return nil, err
	}

	obj, err := NewAPIObject(meta.(*api_client), opts)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func buildApiObjectOpts(d *schema.ResourceData) (*apiObjectOpts, error) {
	opts := &apiObjectOpts{
		path: d.Get("path").(string),
	}

	/* Allow user to override provider-level id_attribute */
	if v, ok := d.GetOk("id_attribute"); ok {
		opts.id_attribute = v.(string)
	}

	/* Allow user to specify the ID manually */
	if v, ok := d.GetOk("object_id"); ok {
		opts.id = v.(string)
	} else {
		/* If not specified, see if terraform has an ID */
		opts.id = d.Id()
	}

	log.Printf("common.go: make_api_object routine called for id '%s'\n", opts.id)

	log.Printf("create_path: %s", d.Get("create_path"))
	if v, ok := d.GetOk("create_path"); ok {
		opts.post_path = v.(string)
	}
	if v, ok := d.GetOk("read_path"); ok {
		opts.get_path = v.(string)
	}
	if v, ok := d.GetOk("update_path"); ok {
		opts.put_path = v.(string)
	}
	if v, ok := d.GetOk("create_method"); ok {
		opts.create_method = v.(string)
	}
	if v, ok := d.GetOk("read_method"); ok {
		opts.read_method = v.(string)
	}
	if v, ok := d.GetOk("update_method"); ok {
		opts.update_method = v.(string)
	}
	if v, ok := d.GetOk("destroy_method"); ok {
		opts.destroy_method = v.(string)
	}
	if v, ok := d.GetOk("destroy_path"); ok {
		opts.delete_path = v.(string)
	}

	read_search := expandReadSearch(d.Get("read_search").(map[string]interface{}))
	opts.read_search = read_search

	opts.data = d.Get("data").(string)
	opts.debug = d.Get("debug").(bool)

	return opts, nil
}

func expandReadSearch(v map[string]interface{}) (read_search map[string]string) {
	read_search = make(map[string]string)
	for key, val := range v {
		read_search[key] = val.(string)
	}

	return
}
