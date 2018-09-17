package restapi

import (
  "github.com/hashicorp/terraform/helper/schema"
  "fmt"
  "errors"
  "log"
  "encoding/json"
  "strings"
)

func dataSourceRestApi() *schema.Resource {
  return &schema.Resource{
    Read:   dataSourceRestApiRead,

    Schema: map[string]*schema.Schema{
      "path": &schema.Schema{
        Type:        schema.TypeString,
        Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
        Required:    true,
      },
      "search_key": &schema.Schema{
        Type:        schema.TypeString,
        Description: "When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name' or a path to such an field in the form 'field/field/field'.",
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
        Type: schema.TypeString,
        Description: "Defaults to `id_attribute` set on the provider. Allows per-resource override of `id_attribute` (see `id_attribute` provider config documentation)",
        Optional: true,
      },
      "debug": &schema.Schema{
        Type:        schema.TypeBool,
        Description: "Whether to emit verbose debug output while working with the API object on the server.",
        Optional:    true,
      },
      "api_data": &schema.Schema{
        Type:        schema.TypeMap,
	Elem:        &schema.Schema{ Type: schema.TypeString },
        Description: "After data from the API server is read, this map will include k/v pairs usable in other terraform resources as readable objects. Currently the value is the golang fmt package's representation of the value (simple primitives are set as expected, but complex types like arrays and maps contain golang formatting).",
	Computed:    true,
      },
    }, /* End schema */

  }
}


func dataSourceRestApiRead(d *schema.ResourceData, meta interface{}) error {
  path := d.Get("path").(string)
  debug := d.Get("debug").(bool)
  client := meta.(*api_client)
  log.Printf("datasource_api_object.go: Data routine called.")

  search_key   := d.Get("search_key").(string)
  search_value := d.Get("search_value").(string)
  results_key  := d.Get("results_key").(string)
  if debug { log.Printf("datasource_api_object.go:\npath: %s\nsearch_key: %s\nsearch_value: %s\nresults_key: %s", path, search_key, search_value, results_key) }

  /* Allow user to override provider-level id_attribute */
  id_attribute := client.id_attribute
  if "" != d.Get("id_attribute").(string) {
    id_attribute = d.Get("id_attribute").(string)
  }

  id := ""
  var data_array []interface{}
  var ok bool

  /*
    Issue a GET to the base path and expect results to come back
  */
  if debug { log.Printf("datasource_api_object.go: Calling API on path '%s'", path) }
  res_str, err := client.send_request("GET", path, "")
  if err != nil { return err }

  /*
    Parse it seeking JSON data
  */
  if debug { log.Printf("datasource_api_object.go: Response recieved... parsing") }
  var result interface{}
  err = json.Unmarshal([]byte(res_str), &result)
  if err != nil { return err }

  if "" != results_key {
    var tmp interface{}

    if debug { log.Printf("datasource_api_object.go: Locating '%s' in the results", results_key) }
    /* First verify the data we got back is a hash */
    if _, ok = result.(map[string]interface{}); !ok {
      return fmt.Errorf("datasource_api_object.go: The results of a GET to '%s' did not return a hash. Cannot search within for results_key '%s'", path, results_key)
    }

    tmp, err = GetObjectAtKey(result.(map[string]interface{}), results_key, debug)
    if err != nil {
      return fmt.Errorf("datasource_api_object.go: Error finding results_key: %s", err)
    }
    if data_array, ok = tmp.([]interface{}); !ok {
      return fmt.Errorf("datasource_api_object.go: The data at results_key location '%s' is not an array.", results_key)
    }
  } else {
    if debug { log.Printf("datasource_api_object.go: results_key is not set - coaxing data to array of interfaces") }
    if data_array, ok = result.([]interface{}); !ok {
      return fmt.Errorf("datasource_api_object.go: The results of a GET to '%s' did not return an array. Perhaps you meant to add a results_key?", path)
    }
  }

  /* Loop through all of the results seeking the specific record */
  for _, item := range data_array {
    hash := item.(map[string]interface{})

    /* Parse search_key in case it has multiple levels*/
    var tmp interface{}
    var search_hash map[string]interface{}
    var reduced_search_key string
    search_parts := strings.Split(search_key, "/")
    final_search_part := search_parts[len(search_parts) - 1]
    if debug { log.Printf("datasource_api_object.go: final_search_part: '%s'", final_search_part) }
    if len(search_parts) > 1 {
      /* strip off last search_part and recombine */
      search_parts = search_parts[:len(search_parts) - 1]
      reduced_search_key = strings.Join(search_parts, "/")
      if debug { log.Printf("datasource_api_object.go: reduced_search_key: '%s'", reduced_search_key) }
      tmp, err = GetObjectAtKey(hash, reduced_search_key, debug)
      if err != nil {
        return fmt.Errorf("datasource_api_object.go: Error parsing seach_key: %s", err)
      }
      if search_hash, ok = tmp.(map[string]interface{}); !ok {
        return fmt.Errorf("datasource_api_object.go: The results of parsing '%s' did not return a hash.", reduced_search_key)
      }
    } else if len(search_parts) == 1 { // search_key only had one segment
      tmp, err = GetObjectAtKey(hash, search_key, debug)
      if err != nil {
        return fmt.Errorf("datasource_api_object.go: Error parsing seach_key: %s", err)
      }
      if search_hash, ok = tmp.(map[string]interface{}); !ok {
        return fmt.Errorf("datasource_api_object.go: The results of parsing '%s' did not return a hash.", search_key)
      }
    } else { // search_key was empty
      search_hash = hash
    }

    /* We found our record */
    if search_hash[final_search_part] == search_value {
      id = fmt.Sprintf("%v", search_hash[id_attribute])
      if debug { log.Printf("datasource_api_object.go: Found ID %s", id) }

      /* But there is no id attribute??? */
      if "" == id {
        return(errors.New(fmt.Sprintf("The object for '%s'='%s' did not have the id attribute '%s'", search_key, search_value, id_attribute)))
      }
      break
    }
  }

  /* Back to terraform-specific stuff. Create an api_object with the ID and refresh it object */
  if debug { log.Printf("datasource_api_object.go: Attempting to construct api_object to refresh data") }
  obj, err := NewAPIObject (
    client,
    path + "/{id}",
    path,
    path + "/{id}",
    path + "/{id}",
    id,
    id_attribute,
    "{}",
    debug,
  )
  if err != nil { return err }
  d.SetId(obj.id)

  err = obj.read_object()
  if err == nil {
    /* Setting terraform ID tells terraform the object was created or it exists */
    log.Printf("datasource_api_object.go: Data resource. Returned id is '%s'\n", obj.id);
    d.SetId(obj.id)
    set_resource_state(obj, d)
  }
  return err
}
