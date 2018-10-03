package restapi

import (
  "github.com/hashicorp/terraform/helper/schema"
  "fmt"
  "log"
  "strings"
)

/* Simple helper routine to build an api_object struct
   for the various calls terraform will use. Unfortunately,
   terraform cannot just reuse objects, so each CRUD operation
   results in a new object created */
func make_api_object(d *schema.ResourceData, m interface{}) (*api_object, error) {

  post_path := d.Get("path").(string)
  get_path := d.Get("path").(string) + "/{id}"
  put_path := d.Get("path").(string) + "/{id}"
  delete_path := d.Get("path").(string) + "/{id}"

  /* Allow user to override provider-level id_attribute */
  id_attribute := m.(*api_client).id_attribute
  if "" != d.Get("id_attribute").(string) {
    id_attribute = d.Get("id_attribute").(string)
  }

  /* Allow user to specify the ID manually */
  id := d.Get("object_id").(string)
  if id == "" {
    /* If not specified, see if terraform has an ID */
    id = d.Id()
  }

  log.Printf("common.go: make_api_object routine called for id '%s'\n", id)

  if "" != d.Get("create_path")  { post_path   = d.Get("create_path").(string) }
  if "" != d.Get("read_path")    { get_path    = d.Get("read_path").(string) }
  if "" != d.Get("update_path")  { put_path    = d.Get("update_path").(string) }
  if "" != d.Get("destroy_path") { delete_path = d.Get("destroy_path").(string) }

  obj, err := NewAPIObject (
    m.(*api_client),
    get_path,
    post_path,
    put_path,
    delete_path,
    id,
    id_attribute,
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

/* Using GetObjectAtKey, this function verifies the resulting
   object is either a JSON string or Number and returns it as a string */
func GetStringAtKey(data map[string]interface{}, path string, debug bool) (string, error) {
  res, err := GetObjectAtKey(data, path, debug)
  if err != nil { return "", err }

  /* JSON supports strings, numbers, objects and arrays. Allow a string OR number here */
  t := fmt.Sprintf("%T", res)
  if t != "string" && t != "float64" {
    return "", fmt.Errorf("Object at path '%s' is not a JSON string or number (float64). The go fmt package says it is '%T'", path, res)
  }

  /* Since it might be a number, coax it to a string with fmt */
  return fmt.Sprintf("%v", res), nil
}

/* Handy helper that will dig through a map and find something
   at the defined key. The returned data is not type checked
   Example:
   Given:
   {
     "attrs": {
       "id": 1234
     },
     "config": {
       "foo": "abc",
       "bar": "xyz"
     }
  }

  Result:
  attrs/id => 1234
  config/foo => "abc"
*/
func GetObjectAtKey(data map[string]interface{}, path string, debug bool) (interface{}, error) {
  hash := data

  parts := strings.Split(path, "/")
  part := ""
  seen := ""
  if debug { log.Printf("common.go:GetObjectAtKey: Locating results_key in parts: %v...", parts) }

  for len(parts) > 1 {
    /* AKA, Slice...*/
    part, parts = parts[0], parts[1:]

    /* Protect against double slashes by mistake */
    if "" == part { continue }

    /* See if this key exists in the hash at this point */
    if _, ok := hash[part]; ok {
      if debug { log.Printf("common.go:GetObjectAtKey:  %s - exists", part) }
      seen += "/" + part
      if tmp, ok := hash[part].(map[string]interface{});ok {
        if debug { log.Printf("common.go:GetObjectAtKey:    %s - is a map", part) }
        hash = tmp
      } else {
        if debug { log.Printf("common.go:GetObjectAtKey:    %s - is a %T", part, hash[part]) }
        return nil, fmt.Errorf("GetObjectAtKey: Object at '%s' is not a map. Is this the right path?", seen)
      }
    } else {
      if debug { log.Printf("common.go:GetObjectAtKey:  %s - MISSING", part) }
      return nil, fmt.Errorf("GetObjectAtKey: Failed to find %s in returned data structure after finding '%s'. Available: %s", part, seen, strings.Join(GetKeys(hash), ","))
    }
  } /* End Loop through parts */

  /* We have found the containing map of the value we want */
  part, parts = parts[0], parts[1:] /* One last time */
  if _, ok := hash[part]; !ok {
    if debug {
      log.Printf("common.go:GetObjectAtKey:  %s - MISSING (available: %s)", part, strings.Join(GetKeys(hash), ","))
    }
    return nil, fmt.Errorf("GetObjectAtKey: Resulting map at %s does not have key %s. Available: %s", seen, part, strings.Join(GetKeys(hash), ","))
  }

  if debug { log.Printf("common.go:GetObjectAtKey:  %s - exists", part) }

  return hash[part], nil
}


/* Handy helper to just dump the keys of a map into a slice */
func GetKeys(hash map[string]interface{}) []string {
  keys := make([]string, 0)
  for k := range hash {
    keys = append(keys, k)
  }
  return keys
}
