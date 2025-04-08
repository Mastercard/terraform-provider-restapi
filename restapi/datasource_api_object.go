package restapi

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRestAPI() *schema.Resource {
	return &schema.Resource{
		Read:        dataSourceRestAPIRead,
		Description: "Performs a cURL get command on the specified url.",

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
				Required:    true,
			},
			"search_path": {
				Type:        schema.TypeString,
				Description: "The API path on top of the base URL set in the provider that represents the location to search for objects of this type on the API server. If not set, defaults to the value of path.",
				Optional:    true,
			},
			"query_string": {
				Type:        schema.TypeString,
				Description: "An optional query string to send when performing the search.",
				Optional:    true,
			},
			"read_query_string": {
				Type: schema.TypeString,
				/* Setting to "not-set" helps differentiate between the cases where
				read_query_string is explicitly set to zero-value for string ("") and
				when read_query_string is not set at all in the configuration. */
				Default:     "not-set",
				Description: "Defaults to `query_string` set on data source. This key allows setting a different or empty query string for reading the object.",
				Optional:    true,
			},
			"search_key": {
				Type:        schema.TypeString,
				Description: "When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'. Similar to results_key, the value may be in the format of 'field/field/field' to search for data deeper in the returned object.",
				Required:    true,
			},
			"search_value": {
				Type:        schema.TypeString,
				Description: "The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.",
				Required:    true,
			},
			"results_key": {
				Type:        schema.TypeString,
				Description: "When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are already an array and are to be used exactly as-is.",
				Optional:    true,
			},
			"id_attribute": {
				Type:        schema.TypeString,
				Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
				Optional:    true,
			},
			"debug": {
				Type:        schema.TypeBool,
				Description: "Whether to emit verbose debug output while working with the API object on the server.",
				Optional:    true,
			},
			"filter_keys": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "A list of keys to filter out when parsing the API response. These keys will be removed from the state at any level in the JSON hierarchy.",
			},
			"api_data": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).",
				Computed:    true,
			},
			"api_response": {
				Type:        schema.TypeString,
				Description: "The raw body of the HTTP response from the last read of the object.",
				Computed:    true,
			},
		}, /* End schema */

	}
}

func dataSourceRestAPIRead(d *schema.ResourceData, meta interface{}) error {
	path := d.Get("path").(string)
	searchPath := d.Get("search_path").(string)
	queryString := d.Get("query_string").(string)
	debug := d.Get("debug").(bool)
	client := meta.(*APIClient)
	if debug {
		log.Printf("datasource_api_object.go: Data routine called.")
	}

	readQueryString := d.Get("read_query_string").(string)
	if readQueryString == "not-set" {
		readQueryString = queryString
	}

	searchKey := d.Get("search_key").(string)
	searchValue := d.Get("search_value").(string)
	resultsKey := d.Get("results_key").(string)
	idAttribute := d.Get("id_attribute").(string)

	if debug {
		log.Printf("datasource_api_object.go:\npath: %s\nsearch_path: %s\nquery_string: %s\nsearch_key: %s\nsearch_value: %s\nresults_key: %s\nid_attribute: %s", path, searchPath, queryString, searchKey, searchValue, resultsKey, idAttribute)
	}

	opts := &apiObjectOpts{
		path:        path,
		searchPath:  searchPath,
		debug:       debug,
		queryString: readQueryString,
		idAttribute: idAttribute,
	}
	
	// Set filter_keys if provided
	if v, ok := d.GetOk("filter_keys"); ok {
		filterKeys := make([]string, 0)
		for _, key := range v.([]interface{}) {
			filterKeys = append(filterKeys, key.(string))
		}
		opts.filterKeys = filterKeys
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		return err
	}

	if _, err := obj.findObject(queryString, searchKey, searchValue, resultsKey); err != nil {
		return err
	}

	/* Back to terraform-specific stuff. Create an api_object with the ID and refresh it object */
	if debug {
		log.Printf("datasource_api_object.go: Attempting to construct api_object to refresh data")
	}

	d.SetId(obj.id)

	err = obj.readObject()
	if err == nil {
		/* Setting terraform ID tells terraform the object was created or it exists */
		log.Printf("datasource_api_object.go: Data resource. Returned id is '%s'\n", obj.id)
		d.SetId(obj.id)
		setResourceState(obj, d)
	}
	return err
}
