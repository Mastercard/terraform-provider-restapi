package restapi

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

/* After any operation that returns API data, we'll stuff
   all the k,v pairs into the api_data map so users can
   consume the values elsewhere if they'd like */
func set_resource_state(obj *api_object, d *schema.ResourceData) {
	api_data := make(map[string]string)
	for k, v := range obj.api_data {
		api_data[k] = fmt.Sprintf("%v", v)
	}
	d.Set("api_data", api_data)
	d.Set("api_response", obj.api_response)
}

/* Using GetObjectAtKey, this function verifies the resulting
   object is either a JSON string or Number and returns it as a string */
func GetStringAtKey(data map[string]interface{}, path string, debug bool) (string, error) {
	res, err := GetObjectAtKey(data, path, debug)
	if err != nil {
		return "", err
	}

	/* JSON supports strings, numbers, objects and arrays. Allow a string OR number here */
	t := fmt.Sprintf("%T", res)
	if t == "string" {
		return res.(string), nil
	} else if t == "float64" {
		return strconv.FormatFloat(res.(float64), 'f', -1, 64), nil
	} else {
		return "", fmt.Errorf("Object at path '%s' is not a JSON string or number (float64). The go fmt package says it is '%T'", path, res)
	}
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
	if debug {
		log.Printf("common.go:GetObjectAtKey: Locating results_key in parts: %v...", parts)
	}

	for len(parts) > 1 {
		/* AKA, Slice...*/
		part, parts = parts[0], parts[1:]

		/* Protect against double slashes by mistake */
		if "" == part {
			continue
		}

		/* See if this key exists in the hash at this point */
		if _, ok := hash[part]; ok {
			if debug {
				log.Printf("common.go:GetObjectAtKey:  %s - exists", part)
			}
			seen += "/" + part
			if tmp, ok := hash[part].(map[string]interface{}); ok {
				if debug {
					log.Printf("common.go:GetObjectAtKey:    %s - is a map", part)
				}
				hash = tmp
			} else if tmp, ok := hash[part].([]interface{}); ok {
				if debug {
					log.Printf("common.go:GetObjectAtKey:    %s - is a list", part)
				}
				mapString := make(map[string]interface{})
				for key, value := range tmp {
					strKey := fmt.Sprintf("%v", key)
					mapString[strKey] = value
				}
				hash = mapString
			} else {
				if debug {
					log.Printf("common.go:GetObjectAtKey:    %s - is a %T", part, hash[part])
				}
				return nil, fmt.Errorf("GetObjectAtKey: Object at '%s' is not a map. Is this the right path?", seen)
			}
		} else {
			if debug {
				log.Printf("common.go:GetObjectAtKey:  %s - MISSING", part)
			}
			return nil, fmt.Errorf("GetObjectAtKey: Failed to find '%s' in returned data structure after finding '%s'. Available: %s", part, seen, strings.Join(GetKeys(hash), ","))
		}
	} /* End Loop through parts */

	/* We have found the containing map of the value we want */
	part, parts = parts[0], parts[1:] /* One last time */
	if _, ok := hash[part]; !ok {
		if debug {
			log.Printf("common.go:GetObjectAtKey:  %s - MISSING (available: %s)", part, strings.Join(GetKeys(hash), ","))
		}
		return nil, fmt.Errorf("GetObjectAtKey: Resulting map at '%s' does not have key '%s'. Available: %s", seen, part, strings.Join(GetKeys(hash), ","))
	}

	if debug {
		log.Printf("common.go:GetObjectAtKey:  %s - exists (%v)", part, hash[part])
	}

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

/* GetEnvOrDefault is a helper function that returns the value of the
given environment variable, if one exists, or the default value */
func GetEnvOrDefault(k string, defaultvalue string) string {
	v := os.Getenv(k)
	if v == "" {
		return defaultvalue
	}
	return v
}
