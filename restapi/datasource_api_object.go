package restapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"reflect"
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

	/* Allow user to override provider-level id_attribute */
	id_attribute := client.id_attribute
	if "" != d.Get("id_attribute").(string) {
		id_attribute = d.Get("id_attribute").(string)
	}

	if debug {
		log.Printf("datasource_api_object.go:\npath: %s\nquery_string: %s\nsearch_key: %s\nsearch_value: %s\nresults_key: %s\nid_attribute: %s", path, query_string, search_key, search_value, results_key, id_attribute)
	}

	id := ""
	var data_array []interface{}
	var ok bool

	/*
	   Issue a GET to the base path and expect results to come back
	*/
	search_path := path
	if "" != query_string {
		if debug {
			log.Printf("datasource_api_object.go: Adding query string '%s'", query_string)
		}
		search_path = fmt.Sprintf("%s?%s", search_path, query_string)
	}

	if debug {
		log.Printf("datasource_api_object.go: Calling API on path '%s'", search_path)
	}
	res_str, err := client.send_request("GET", search_path, "")
	if err != nil {
		return err
	}

	/*
	   Parse it seeking JSON data
	*/
	if debug {
		log.Printf("datasource_api_object.go: Response recieved... parsing")
	}
	var result interface{}
	err = json.Unmarshal([]byte(res_str), &result)
	if err != nil {
		return err
	}

	if "" != results_key {
		var tmp interface{}

		if debug {
			log.Printf("datasource_api_object.go: Locating '%s' in the results", results_key)
		}

		/* First verify the data we got back is a hash */
		if _, ok = result.(map[string]interface{}); !ok {
			return fmt.Errorf("datasource_api_object.go: The results of a GET to '%s' did not return a hash. Cannot search within for results_key '%s'", search_path, results_key)
		}

		tmp, err = GetObjectAtKey(result.(map[string]interface{}), results_key, debug)
		if err != nil {
			return fmt.Errorf("datasource_api_object.go: Error finding results_key: %s", err)
		}
		if data_array, ok = tmp.([]interface{}); !ok {
			return fmt.Errorf("datasource_api_object.go: The data at results_key location '%s' is not an array. It is a '%s'", results_key, reflect.TypeOf(tmp))
		}
	} else {
		if debug {
			log.Printf("datasource_api_object.go: results_key is not set - coaxing data to array of interfaces")
		}
		if data_array, ok = result.([]interface{}); !ok {
			return fmt.Errorf("datasource_api_object.go: The results of a GET to '%s' did not return an array. It is a '%s'. Perhaps you meant to add a results_key?", search_path, reflect.TypeOf(result))
		}
	}

	/* Loop through all of the results seeking the specific record */
	for _, item := range data_array {
		var hash map[string]interface{}

		if hash, ok = item.(map[string]interface{}); !ok {
			return fmt.Errorf("datasource_api_object.go: The elements being searched for data are not a map of key value pairs.")
		}

		if debug {
			log.Printf("datasource_api_object.go: Examining %v", hash)
			log.Printf("datasource_api_object.go:   Comparing '%s' to the value in '%s'", search_value, search_key)
		}

		tmp, err := GetStringAtKey(hash, search_key, debug)
		if err != nil {
			return (fmt.Errorf("Failed to get the value of '%s' in the results array at '%s': %s", search_key, results_key, err))
		}

		/* We found our record */
		if tmp == search_value {
			id, err = GetStringAtKey(hash, id_attribute, debug)
			if err != nil {
				return (fmt.Errorf("Failed to find id_attribute '%s' in the record: %s", id_attribute, err))
			}

			if debug {
				log.Printf("datasource_api_object.go:   Found ID '%s'", id)
			}

			/* But there is no id attribute??? */
			if "" == id {
				return (errors.New(fmt.Sprintf("The object for '%s'='%s' did not have the id attribute '%s', or the value was empty.", search_key, search_value, id_attribute)))
			}
			break
		}
	}

	if "" == id {
		return (fmt.Errorf("Failed to find an object with the '%s' key = '%s' at %s", search_key, search_value, search_path))
	}

	/* Back to terraform-specific stuff. Create an api_object with the ID and refresh it object */
	if debug {
		log.Printf("datasource_api_object.go: Attempting to construct api_object to refresh data")
	}
	obj, err := NewAPIObject(
		client,
		path+"/{id}",
		path,
		path+"/{id}",
		path+"/{id}",
		id,
		id_attribute,
		"{}",
		debug,
	)
	if err != nil {
		return err
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
