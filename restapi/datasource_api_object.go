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
  obj, err := NewAPIObject (
    meta.(*api_client),
    d.Get("path").(string),
    d.Id(),
    "{}",
    d.Get("debug").(bool),
  )

  if err != nil { return err }
  log.Printf("resource_api_object.go: Data routine called. Object built:\n%s\n", obj.toString())

  search_key   := d.Get("search_key").(string)
  search_val   := d.Get("search_value").(string)
  results_key  := d.Get("results_key").(string)
  id_attribute := obj.api_client.id_attribute
  id := ""
  var data_array []interface{}

  /*
    Issue a GET to the base path and expect results to come back
  */
  res_str, err := obj.api_client.send_request("GET", obj.path, "")
  if err != nil { return err }

  /*
    Parse it seeking JSON data
  */
  var result interface{}
  err = json.Unmarshal([]byte(res_str), &result)
  if err != nil { return err }

  if "" != results_key {
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
        fmt.Printf("  exists\n")
        v := hash[part]
        ptr = &v
        seen += "/" + part
      } else {
        return(errors.New(fmt.Sprintf("Failed to find %s in returned data structure after finding '%s'", part, seen)))
      }
    } /* End Loop through parts */

    data_array = (*ptr).([]interface{})
  } else {
    data_array = result.([]interface{})
  }

  /* Loop through all of the results seeking the specific record */
  for _, item := range data_array {
    hash := item.(map[string]interface{})

    /* We found our record */
    if hash[search_key] == search_val {
      id = hash[id_attribute].(string)

      /* But there is no id attribute??? */
      if "" == id {
        return(errors.New(fmt.Sprintf("The object for '%s'='%s' did not have the id attribute '%s'", search_key, search_val, id_attribute)))
      }
      break
    }
  }

  /* Back to terraform-specific stuff. Set the id and refresh the object */
  d.SetId(obj.id)
  obj.id = id

  err = obj.read_object()
  if err == nil {
    /* Setting terraform ID tells terraform the object was created or it exists */
    log.Printf("resource_api_object.go: Data resource. Returned id is '%s'\n", obj.id);
    d.SetId(obj.id)
    set_resource_state(obj, d)
  }
  return err
}
