package restapi

import (
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func dataSourceRestApi() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRestApiRead,

		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
				Required:    true,
			},
			"query_string": &schema.Schema{
				Type:        schema.TypeString,
				Description: "An optional query string to send when performing the search.",
				Optional:    true,
			},
			"search_key": &schema.Schema{
				Type:        schema.TypeString,
				Description: "When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'. Similar to results_key, the value may be in the format of 'field/field/field' to search for data deeper in the returned object.",
				Required:    true,
			},
			"search_value": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.",
				Required:    true,
			},
			"results_key": &schema.Schema{
				Type:        schema.TypeString,
				Description: "When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are already an array and are to be used exactly as-is.",
				Optional:    true,
			},
			"id_attribute": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
				Optional:    true,
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
			"api_response": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The raw body of the HTTP response from the last read of the object.",
				Computed:    true,
			},
		}, /* End schema */

	}
}

func dataSourceRestApiRead(d *schema.ResourceData, meta interface{}) error {
	path := d.Get("path").(string)
	query_string := d.Get("query_string").(string)
	debug := d.Get("debug").(bool)
	client := meta.(*api_client)
	if debug {
		log.Printf("datasource_api_object.go: Data routine called.")
	}

	search_key := d.Get("search_key").(string)
	search_value := d.Get("search_value").(string)
	results_key := d.Get("results_key").(string)
	id_attribute := d.Get("id_attribute").(string)

	if debug {
		log.Printf("datasource_api_object.go:\npath: %s\nquery_string: %s\nsearch_key: %s\nsearch_value: %s\nresults_key: %s\nid_attribute: %s", path, query_string, search_key, search_value, results_key, id_attribute)
	}

	opts := &apiObjectOpts{
		path:         path,
		debug:        debug,
		id_attribute: id_attribute,
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		return err
	}

	if _, err := obj.find_object(query_string, search_key, search_value, results_key); err != nil {
		return err
	}

	/* Back to terraform-specific stuff. Create an api_object with the ID and refresh it object */
	if debug {
		log.Printf("datasource_api_object.go: Attempting to construct api_object to refresh data")
	}

	d.SetId(obj.id)

	err = obj.read_object()
	if err == nil {
		/* Setting terraform ID tells terraform the object was created or it exists */
		log.Printf("datasource_api_object.go: Data resource. Returned id is '%s'\n", obj.id)
		d.SetId(obj.id)
		set_resource_state(obj, d)
	}
	return err
}
