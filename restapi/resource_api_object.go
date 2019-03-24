package restapi

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
)

func resourceRestApi() *schema.Resource {
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
			},
			"debug": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Whether to emit verbose debug output while working with the API object on the server.",
				Optional:    true,
			},
			"api_data": &schema.Schema{
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).",
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
	n := strings.LastIndex(input, "/")
	if n == -1 {
		return imported, fmt.Errorf("Invalid path to import api_object '%s'. Must be /<full path from server root>/<object id>", input)
	}

	path := input[0:n]
	d.Set("path", path)

	id := input[n+1 : len(input)]
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

	if err := obj.read_object(); err == nil {
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
	if err := obj.read_object(); err == nil {
		exists = true
	}
	return exists, err
}

type resourceRestApiOpts struct {
	get_path     string
	post_path    string
	put_path     string
	delete_path  string
	search_path  string
	debug        bool
	id           string
	id_attribute string
	data         string
}

/* Simple helper routine to build an api_object struct
   for the various calls terraform will use. Unfortunately,
   terraform cannot just reuse objects, so each CRUD operation
   results in a new object created */
func make_api_object(d *schema.ResourceData, meta interface{}) (*api_object, error) {
	opts, err := buildResourceRestApiOpts(d)
	if err != nil {
		return nil, err
	}

	obj, err := NewAPIObject(meta.(*api_client), opts)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func buildResourceRestApiOpts(d *schema.ResourceData) (*resourceRestApiOpts, error) {
	opts := &resourceRestApiOpts{}

	if v := d.Get("path").(string); v != "" {
		opts.post_path = v
		opts.get_path = v + "/{id}"
		opts.put_path = v + "/{id}"
		opts.delete_path = v + "/{id}"
	}

	/* Allow user to override provider-level id_attribute */
	opts.id_attribute = d.Get("id_attribute").(string)

	/* Allow user to specify the ID manually */
	if v := d.Get("object_id").(string); v != "" {
		opts.id = v
	} else {
		/* If not specified, see if terraform has an ID */
		opts.id = d.Id()
	}

	log.Printf("common.go: make_api_object routine called for id '%s'\n", opts.id)

	if v := d.Get("create_path"); v != "" {
		opts.post_path = v.(string)
	}
	if v := d.Get("read_path"); v != "" {
		opts.post_path = v.(string)
	}
	if v := d.Get("update_path"); v != "" {
		opts.post_path = v.(string)
	}
	if v := d.Get("destroy_path"); v != "" {
		opts.post_path = v.(string)
	}

	opts.data = d.Get("data").(string)
	opts.debug = d.Get("debug").(bool)

	return opts, nil
}
