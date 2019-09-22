package restapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type apiObjectOpts struct {
	path         string
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

type api_object struct {
	api_client   *api_client
	get_path     string
	post_path    string
	put_path     string
	delete_path  string
	search_path  string
	debug        bool
	id           string
	id_attribute string

	/* Set internally */
	data         map[string]interface{} /* Data as managed by the user */
	api_data     map[string]interface{} /* Data as available from the API */
	api_response string
}

// Make an api_object to manage a RESTful object in an API
func NewAPIObject(i_client *api_client, opts *apiObjectOpts) (*api_object, error) {
	if opts.debug {
		log.Printf("api_object.go: Constructing debug api_object\n")
		log.Printf(" id: %s\n", opts.id)
	}

	/* id_attribute can be set either on the client (to apply for all calls with the server)
	   or on a per object basis (for only calls to this kind of object).
	   Permit overridding from the API client here by using the client-wide value only
	   if a per-object value is not set */
	if opts.id_attribute == "" {
		opts.id_attribute = i_client.id_attribute
	}

	if opts.post_path == "" {
		opts.post_path = opts.path
	}
	if opts.get_path == "" {
		opts.get_path = opts.path + "/{id}"
	}
	if opts.put_path == "" {
		opts.put_path = opts.path + "/{id}"
	}
	if opts.delete_path == "" {
		opts.delete_path = opts.path + "/{id}"
	}
	if opts.search_path == "" {
		opts.search_path = opts.path
	}

	obj := api_object{
		api_client:   i_client,
		get_path:     opts.get_path,
		post_path:    opts.post_path,
		put_path:     opts.put_path,
		delete_path:  opts.delete_path,
		search_path:  opts.search_path,
		debug:        opts.debug,
		id:           opts.id,
		id_attribute: opts.id_attribute,
		data:         make(map[string]interface{}),
		api_data:     make(map[string]interface{}),
	}

	if opts.data != "" {
		if opts.debug {
			log.Printf("api_object.go: Parsing data: '%s'", opts.data)
		}

		err := json.Unmarshal([]byte(opts.data), &obj.data)
		if err != nil {
			return nil, err
		}

		/* Opportunistically set the object's ID if it is provided in the data.
		   If it is not set, we will get it later in synchronize_state */
		if obj.id == "" {
			var tmp string
			tmp, err := GetStringAtKey(obj.data, obj.id_attribute, obj.debug)
			if err == nil {
				if opts.debug {
					log.Printf("api_object.go: opportunisticly set id from data provided.")
				}
				obj.id = tmp
			} else if !obj.api_client.write_returns_object && !obj.api_client.create_returns_object && obj.search_path == "" {
				/* If the id is not set and we cannot obtain it
				   later, error out to be safe */
				return nil, errors.New(fmt.Sprintf("Provided data does not have %s attribute for the object's id and the client is not configured to read the object from a POST response. Without an id, the object cannot be managed.", obj.id_attribute))
			}
		}
	}

	if opts.debug {
		log.Printf("api_object.go: Constructed object: %s", obj.toString())
	}
	return &obj, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *api_object) toString() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("id: %s\n", obj.id))
	buffer.WriteString(fmt.Sprintf("get_path: %s\n", obj.get_path))
	buffer.WriteString(fmt.Sprintf("post_path: %s\n", obj.post_path))
	buffer.WriteString(fmt.Sprintf("put_path: %s\n", obj.put_path))
	buffer.WriteString(fmt.Sprintf("delete_path: %s\n", obj.delete_path))
	buffer.WriteString(fmt.Sprintf("debug: %t\n", obj.debug))
	buffer.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(obj.data)))
	buffer.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(obj.api_data)))
	return buffer.String()
}

/* Centralized function to ensure that our data as managed by
   the api_object is updated with data that has come back from
   the API */
func (obj *api_object) update_state(state string) error {
	if obj.debug {
		log.Printf("api_object.go: Updating API object state to '%s'\n", state)
	}

	/* Other option - Decode as JSON Numbers instead of golang datatypes
	d := json.NewDecoder(strings.NewReader(res_str))
	d.UseNumber()
	err = d.Decode(&obj.api_data)
	*/
	err := json.Unmarshal([]byte(state), &obj.api_data)
	if err != nil {
		return err
	}

	/* Store response body for parsing via jsondecode() */
	obj.api_response = state

	/* A usable ID was not passed (in constructor or here),
	   so we have to guess what it is from the data structure */
	if obj.id == "" {
		val, err := GetStringAtKey(obj.api_data, obj.id_attribute, obj.debug)
		if err != nil {
			return fmt.Errorf("api_object.go: Error extracting ID from data element: %s", err)
		}
		obj.id = val
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
	res_str, err := obj.api_client.send_request(obj.api_client.create_method, strings.Replace(obj.post_path, "{id}", obj.id, -1), string(b))
	if err != nil {
		return err
	}

	/* We will need to sync state as well as get the object's ID */
	if obj.api_client.write_returns_object || obj.api_client.create_returns_object {
		if obj.debug {
			log.Printf("api_object.go: Parsing response from POST to update internal structures (write_returns_object=%t, create_returns_object=%t)...\n",
				obj.api_client.write_returns_object, obj.api_client.create_returns_object)
		}
		err = obj.update_state(res_str)
		/* Yet another failsafe. In case something terrible went wrong internally,
		   bail out so the user at least knows that the ID did not get set. */
		if obj.id == "" {
			return errors.New("Internal validation failed. Object ID is not set, but *may* have been created. This should never happen!")
		}
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

	res_str, err := obj.api_client.send_request(obj.api_client.read_method, strings.Replace(obj.get_path, "{id}", obj.id, -1), "")
	if err != nil {
		return err
	}

	err = obj.update_state(res_str)
	return err
}

func (obj *api_object) update_object() error {
	if obj.id == "" {
		return errors.New("Cannot update an object unless the ID has been set.")
	}

	b, _ := json.Marshal(obj.data)
	res_str, err := obj.api_client.send_request(obj.api_client.update_method, strings.Replace(obj.put_path, "{id}", obj.id, -1), string(b))
	if err != nil {
		return err
	}

	if obj.api_client.write_returns_object {
		if obj.debug {
			log.Printf("api_object.go: Parsing response from PUT to update internal structures (write_returns_object=true)...\n")
		}
		err = obj.update_state(res_str)
	} else {
		if obj.debug {
			log.Printf("api_object.go: Requesting updated object from API (write_returns_object=false)...\n")
		}
		err = obj.read_object()
	}
	return err
}

func (obj *api_object) delete_object() error {
	if obj.id == "" {
		log.Printf("WARNING: Attempting to delete an object that has no id set. Assuming this is OK.\n")
		return nil
	}

	_, err := obj.api_client.send_request(obj.api_client.destroy_method, strings.Replace(obj.delete_path, "{id}", obj.id, -1), "")
	if err != nil {
		return err
	}

	return nil
}

func (obj *api_object) find_object(query_string string, search_key string, search_value string, results_key string) error {
	var data_array []interface{}
	var ok bool

	/*
	   Issue a GET to the base path and expect results to come back
	*/
	search_path := obj.search_path
	if "" != query_string {
		if obj.debug {
			log.Printf("api_object.go: Adding query string '%s'", query_string)
		}
		search_path = fmt.Sprintf("%s?%s", obj.search_path, query_string)
	}

	if obj.debug {
		log.Printf("api_object.go: Calling API on path '%s'", search_path)
	}
	res_str, err := obj.api_client.send_request(obj.api_client.read_method, search_path, "")
	if err != nil {
		return err
	}

	/*
	   Parse it seeking JSON data
	*/
	if obj.debug {
		log.Printf("api_object.go: Response received... parsing")
	}
	var result interface{}
	err = json.Unmarshal([]byte(res_str), &result)
	if err != nil {
		return err
	}

	if "" != results_key {
		var tmp interface{}

		if obj.debug {
			log.Printf("api_object.go: Locating '%s' in the results", results_key)
		}

		/* First verify the data we got back is a hash */
		if _, ok = result.(map[string]interface{}); !ok {
			return fmt.Errorf("api_object.go: The results of a GET to '%s' did not return a hash. Cannot search within for results_key '%s'", search_path, results_key)
		}

		tmp, err = GetObjectAtKey(result.(map[string]interface{}), results_key, obj.debug)
		if err != nil {
			return fmt.Errorf("api_object.go: Error finding results_key: %s", err)
		}
		if data_array, ok = tmp.([]interface{}); !ok {
			return fmt.Errorf("api_object.go: The data at results_key location '%s' is not an array. It is a '%s'", results_key, reflect.TypeOf(tmp))
		}
	} else {
		if obj.debug {
			log.Printf("api_object.go: results_key is not set - coaxing data to array of interfaces")
		}
		if data_array, ok = result.([]interface{}); !ok {
			return fmt.Errorf("api_object.go: The results of a GET to '%s' did not return an array. It is a '%s'. Perhaps you meant to add a results_key?", search_path, reflect.TypeOf(result))
		}
	}

	/* Loop through all of the results seeking the specific record */
	for _, item := range data_array {
		var hash map[string]interface{}

		if hash, ok = item.(map[string]interface{}); !ok {
			return fmt.Errorf("api_object.go: The elements being searched for data are not a map of key value pairs.")
		}

		if obj.debug {
			log.Printf("api_object.go: Examining %v", hash)
			log.Printf("api_object.go:   Comparing '%s' to the value in '%s'", search_value, search_key)
		}

		tmp, err := GetStringAtKey(hash, search_key, obj.debug)
		if err != nil {
			return (fmt.Errorf("Failed to get the value of '%s' in the results array at '%s': %s", search_key, results_key, err))
		}

		/* We found our record */
		if tmp == search_value {
			obj.id, err = GetStringAtKey(hash, obj.id_attribute, obj.debug)
			if err != nil {
				return (fmt.Errorf("Failed to find id_attribute '%s' in the record: %s", obj.id_attribute, err))
			}

			if obj.debug {
				log.Printf("api_object.go:   Found ID '%s'", obj.id)
			}

			/* But there is no id attribute??? */
			if "" == obj.id {
				return (errors.New(fmt.Sprintf("The object for '%s'='%s' did not have the id attribute '%s', or the value was empty.", search_key, search_value, obj.id_attribute)))
			}
			break
		}
	}

	if "" == obj.id {
		return (fmt.Errorf("Failed to find an object with the '%s' key = '%s' at %s", search_key, search_value, search_path))
	}

	return nil
}
