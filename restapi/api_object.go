package restapi

import (
  "log"
  "errors"
  "fmt"
  "encoding/json"
  "bytes"
  "github.com/davecgh/go-spew/spew"
)

type api_object struct {
  api_client           *api_client
  path                 string
  debug                bool
  id                   string

  /* Set internally */
  data         map[string]interface{} /* Data as managed by the user */
  api_data     map[string]interface{} /* Data as available from the API */
}

// Make an api_object to manage a RESTful object in an API
func NewAPIObject (i_client *api_client, i_path string, i_id string, i_data string, i_debug bool) (*api_object, error) {
  if i_debug {
    log.Printf("api_object.go: Constructing debug api_object\n")
    log.Printf(" path: %s\n", i_path)
    log.Printf(" id: %s\n", i_id)
  }

  obj := api_object{
    api_client: i_client,
    path: i_path,
    debug: i_debug,
    id: i_id,
    data: make(map[string]interface{}),
    api_data: make(map[string]interface{}),
  }

  if "" == i_path { return nil, errors.New("No path passed to api_object constructor") }
  if "" == i_data { return nil, errors.New("No data passed to api_object constructor") }

  if i_data != ""{
    if i_debug { log.Printf("api_object.go: Parsing data: '%s'", i_data) }

    err := json.Unmarshal([]byte(i_data), &obj.data)
    if err != nil {
      return nil, err
    }

    /* Opportunistically set the object's ID if it is provided in the data.
       If it is not set, we will get it later in synchronize_state */
    if obj.id == "" {
      val, ok := obj.data[obj.api_client.id_attribute]
      if ok {
        obj.id = fmt.Sprintf("%v", val)
      } else if !obj.api_client.write_returns_object && !obj.api_client.create_returns_object {
        /* If the id is not set and we cannot obtain it
	   later, error out to be safe */
        return nil, errors.New(fmt.Sprintf("Provided data does not have %s attribute for the object's id and the client is not configured to read the object from a POST response. Without an id, the object cannot be managed.", obj.api_client.id_attribute))
      }
    }
  }

  if obj.debug { log.Printf("api_object.go: Constructed object: %s", obj.toString()) }
  return &obj, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *api_object) toString() string {
  var buffer bytes.Buffer
  buffer.WriteString(fmt.Sprintf("id: %s\n", obj.id))
  buffer.WriteString(fmt.Sprintf("path: %s\n", obj.path))
  buffer.WriteString(fmt.Sprintf("debug: %t\n", obj.debug))
  buffer.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(obj.data)))
  buffer.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(obj.api_data)))
  return buffer.String()
}

/* Centralized function to ensure that our data as managed by
   the api_object is updated with data that has come back from
   the API */
func (obj *api_object) update_state(state string) error {
  if obj.debug { log.Printf("api_object.go: Updating API object state to '%s'\n", state) }

  /* Other option - Decode as JSON Numbers instead of golang datatypes
  d := json.NewDecoder(strings.NewReader(res_str))
  d.UseNumber()
  err = d.Decode(&obj.api_data)
  */
  err := json.Unmarshal([]byte(state), &obj.api_data)
  if err != nil { return err }

  /* A usable ID was not passed (in constructor or here), 
     so we have to guess what it is from the data structure */
  if obj.id == "" {
    val, ok := obj.api_data[obj.api_client.id_attribute]
    if ok {
      /* Coax to string */
      obj.id = fmt.Sprintf("%v", val)
      log.Printf("api_object.go: Updating object id (unset) to '%s'\n", obj.id)
    } else {
      /* An ID is REQUIRED to manage the object. We canot proceed */
      err_message := fmt.Sprintf("api_object.go: Error: %s is not in the data presented nor passed in the constructor.\n", obj.api_client.id_attribute)
      err_message += fmt.Sprintf("List of keys available:\n")
      for k := range obj.data { err_message += fmt.Sprintf("  %s\n", k) }
      errors.New(err_message)
    }
  } else if obj.debug {
    log.Printf("api_object.go: Not updating id. It is already set to '%s'\n", obj.id)
  }

  /* Any keys that come from the data we want to copy are done here */
  if len(obj.api_client.copy_keys) > 0 {
    for _, key := range obj.api_client.copy_keys {
      if obj.debug {
        log.Printf("api_object.go: Copying key '%s' from api_data (%v) to data (%v)\n", key, obj.api_data[key], obj.data[key])
      }
      obj.data[key] = obj.api_data[key]
    }
  } else if obj.debug {
    log.Printf("api_object.go: copy_keys is empty - not attempting to copy data")
  }

  if obj.debug {
    log.Printf("api_object.go: final object after synchronization of state:\n%+v\n", obj.toString())
  }
  return err
}

func (obj *api_object) create_object() error {
  /* Failsafe: The constructor should prevent this situation, but
     protect here also. If no id is set, and the API does not respond
     with the id of whatever gets created, we have no way to know what
     the object's id will be. Abandon this attempt */
  if obj.id == "" && !obj.api_client.write_returns_object && !obj.api_client.create_returns_object {
    return errors.New("ERROR: Provided object does not have an id set and the client is not configured to read the object from a POST or PUT response. Without an id, the object cannot be managed.")
  }

  b, _ := json.Marshal(obj.data)
  res_str, err := obj.api_client.send_request("POST", obj.path, string(b))
  if err != nil { return err }

  /* We will need to sync state as well as get the object's ID */
  if obj.api_client.write_returns_object || obj.api_client.create_returns_object {
    if obj.debug {
      log.Printf("api_object.go: Parsing response from POST to update internal structures (write_returns_object=%t, create_returns_object=%t)...\n",
        obj.api_client.write_returns_object, obj.api_client.create_returns_object)
    }
    err = obj.update_state(res_str)
    /* Yet another failsafe. In case something terrible went wrong internally,
       bail out so the user at least knows that the ID did not get set. */
    if obj.id == "" { return errors.New("Internal validation failed. Object ID is not set, but *may* have been created. This should never happen!") }
  } else {
    if obj.debug {
      log.Printf("api_object.go: Requesting created object from API (write_returns_object=%t, create_returns_object=%t)...\n",
        obj.api_client.write_returns_object, obj.api_client.create_returns_object)
    }
    err = obj.read_object()
  }
  return err
}

func (obj *api_object) read_object() error {
  if obj.id == "" {
    return errors.New("Cannot read an object unless the ID has been set.")
  }

  res_str, err := obj.api_client.send_request("GET", obj.path + "/" + obj.id, "")
  if err != nil { return err }

  err = obj.update_state(res_str)
  return err
}

func (obj *api_object) update_object() error {
  if obj.id == "" {
    return errors.New("Cannot update an object unless the ID has been set.")
  }

  b, _ := json.Marshal(obj.data)
  res_str, err := obj.api_client.send_request("PUT", obj.path + "/" + obj.id, string(b))
  if err != nil { return err }

  if obj.api_client.write_returns_object {
    if obj.debug { log.Printf("api_object.go: Parsing response from PUT to update internal structures (write_returns_object=true)...\n") }
    err = obj.update_state(res_str)
  } else {
    if obj.debug { log.Printf("api_object.go: Requesting updated object from API (write_returns_object=false)...\n") }
    err = obj.read_object()
  }
  return err
}

func (obj *api_object) delete_object() error {
  if obj.id == "" {
    log.Printf("WARNING: Attempting to delete an object that has no id set. Assuming this is OK.\n")
    return nil
  }

  _, err := obj.api_client.send_request("DELETE", obj.path + "/" + obj.id, "")
  if err != nil { return err }

  return nil
}
