package restapi

import (
  "github.com/hashicorp/terraform/helper/schema"
  "fmt"
  "errors"
  "log"
  "strings"
  "encoding/json"
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
        Description: "When reading search results from the API, this key is used to identify the specific record to read. This should be a unique record such as 'name'.",
        Required:    true,
      },
      "search_value": &schema.Schema{
        Type:        schema.TypeString,
        Description: "The value of 'search_key' will be compared to this value to determine if the correct object was found. Example: if 'search_key' is 'name' and 'search_value' is 'foo', the record in the array returned by the API with name=foo will be used.",
        Required:    true,
      },
      "results_key": &schema.Schema{
        Type:        schema.TypeString,
        Description: "When issuing a GET to the path, this JSON key is used to locate the results array. The format is 'field/field/field'. Example: 'results/values'. If omitted, it is assumed the results coming back are to be used exactly as-is.",
        Optional:    true,
      },
      "single_object": &schema.Schema{
        Type:        schema.TypeBool,
        Description: "Whether the API returns a single object as a map. In this case, we don't need to search for matching object.",
        Optional:    true,
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
  /* Datasource really only uses GET... but our constructor
  supports different paths per action. Just use path for all of them */
  path := d.Get("path").(string)

  obj, err := NewAPIObject (
    meta.(*api_client),
    path,
    path,
    path,
    path,
    "0", // We temporarily set ID of the new object to 0
    "{}",
    d.Get("debug").(bool),
  )

  if err != nil { return err }
  log.Printf("datasource_api_object.go: Data routine called. Object built:\n%s\n", obj.toString())

  single_obj   := d.Get("single_object").(bool)
  search_key   := d.Get("search_key").(string)
  search_val   := d.Get("search_value").(string)
  results_key  := d.Get("results_key").(string)
  id_attribute := obj.api_client.id_attribute
  id := ""
  var data_array []interface{}
  var single_object_data map[string]interface{}

  /*
    Issue a GET to the base path and expect results to come back
  */
  res_headers, res_body, err := obj.api_client.send_request("GET", path, "")
  if err != nil { return err }

  /*
    Parse it seeking JSON data
  */
  var result interface{}
  err = json.Unmarshal([]byte(res_body), &result)
  if err != nil { return err }

  if "" != results_key && !single_obj {
    ptr := &result
    parts := strings.Split(results_key, "/")
    part := ""
    seen := ""

    for len(parts) > 0 {
      /* AKA, Slice...*/
      part, parts = parts[0], parts[1:]

      /* Protect against double slashes by mistake */
      if "" == part { break }

      hash := (*ptr).(map[string]interface{})
      if _, ok := hash[part]; ok {
        fmt.Printf("hash[part] exists\n")
        v := hash[part]
        ptr = &v
        seen += "/" + part
      } else {
        return(errors.New(fmt.Sprintf("Failed to find %s in returned data structure after finding '%s'", part, seen)))
      }
    } /* End Loop through parts */

    data_array = (*ptr).([]interface{})
  } else if "" == results_key && !single_obj {
    data_array = result.([]interface{})
  } else { // single_obj
    ptr := &result
    single_object_data = (*ptr).(map[string]interface{})
  }

  if !single_obj {

    /* Loop through all of the results seeking the specific record */
    for _, item := range data_array {
      hash := item.(map[string]interface{})

      // Parse search_key
      search_parts := strings.Split(search_key, "/")
      search_part := ""
      search_seen := ""
      search_hash := hash

      // Loop through search parts
      for len(search_parts) > 1 {
        log.Printf("datasource_api_object.go: Looping through search_parts\n")
        search_part, search_parts = search_parts[0], search_parts[1:]

        // Protect against double slashes by mistake
        if "" == search_part { break }

        if _, ok := search_hash[search_part]; ok {
          log.Printf("search_hash[search_part] exists: '%s'\n", search_hash[search_part])
          search_hash = search_hash[search_part].(map[string]interface{})
          search_seen += "/" + search_part
        } else {
          log.Printf("Failed to find %s in returned data structure after finding '%s'", search_part, search_seen)
        }
      } // end search_parts loop

      search_part, search_parts = search_parts[0], search_parts[1:]
      search_data_map := search_hash
      log.Printf("search_part is set to '%s'\n", search_part)
      if search_data_map[search_part] == search_val {
        /* We found our record */
        log.Printf("datasource_api_object.go: Found our record\n")
        id = search_data_map[id_attribute].(string)
        /* But there is no id attribute??? */
        if "" == id {
          log.Printf("datasource_api_object.go: But record did not have '%s' attribute\n", id_attribute)
          return(errors.New(fmt.Sprintf("The object for '%s'='%s' did not have the id attribute '%s'", search_key, search_val, id_attribute)))
        }
        break
      } // end if for checking search_key against search_val
    } // end data_array loop

    // Change get_path to include ID
    obj.get_path = path + "/{id}"

  } else {
    // Assume id_attribute at top of single object
    id = single_object_data[id_attribute].(string)
    log.Printf("datasource_api_object.go: Single object ID: '%s'\n", id)
  }

  /* Back to terraform-specific stuff. Set the id and refresh the object */
  log.Printf("datasource_api_object.go: Setting object ID to '%s'\n", id)
  obj.id = id

  err = obj.read_object()
  if err == nil {
    /* Setting terraform ID tells terraform the object was created or it exists */
    log.Printf("datasource_api_object.go: Data resource. Returned id is '%s'\n", obj.id);
    d.SetId(obj.id)
    set_resource_state(obj, d)
  }
  return err
}
