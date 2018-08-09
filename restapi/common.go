package restapi

import (
  "github.com/hashicorp/terraform/helper/schema"
  "fmt"
  "log"
)

/* Simple helper routine to build an api_object struct
   for the various calls terraform will use. Unfortunately,
   terraform cannot just reuse objects, so each CRUD operation
   results in a new object created */
func make_api_object(d *schema.ResourceData, m interface{}) (*api_object, error) {
  log.Printf("resource_api_object.go: make_api_object routine called for id '%s'\n", d.Id())

  obj, err := NewAPIObject (
    m.(*api_client),
    d.Get("path").(string),
    d.Id(),
    d.Get("data").(string),
    d.Get("debug").(bool),
  )
  return obj, err
}

/* After any operation that returns API data, we'll stuff
   all the k,v pairs into the api_data map so users can
   consume the values elsewhere if they'd like */
func set_resource_state(obj *api_object, d *schema.ResourceData) {
  api_data := make(map[string]string)
  for k, v := range obj.api_data {
    api_data[k] = fmt.Sprintf("%v", v)
  }
  d.Set("api_data", api_data)
}
