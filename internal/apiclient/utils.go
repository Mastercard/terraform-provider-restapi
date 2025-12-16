package restapi

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// SetResourceState sets terraform resource data from APIObject
// After any operation that returns API data, we'll stuff all the k,v pairs into the api_data map so users can consume the values elsewhere if they'd like
func SetResourceState(obj *APIObject, d *schema.ResourceData) {
	apiData := make(map[string]string)
	for k, v := range obj.apiData {
		apiData[k] = fmt.Sprintf("%v", v)
	}
	d.Set("api_data", apiData)
	d.Set("api_response", obj.APIResponse)
}

// GetStringAtKey uses GetObjectAtKey to verify the resulting object is either a JSON string or Number and returns it as a string
func GetStringAtKey(data map[string]interface{}, path string, debug bool) (string, error) {
	res, err := GetObjectAtKey(data, path, debug)
	if err != nil {
		return "", err
	}

	/* JSON supports strings, numbers, objects and arrays. Allow a string OR number here */
	switch tmp := res.(type) {
	case string:
		return tmp, nil
	case float64:
		return strconv.FormatFloat(tmp, 'f', -1, 64), nil
	case bool:
		return fmt.Sprintf("%v", res), nil
	default:
		return "", fmt.Errorf("object at path '%s' is not a JSON string or number (float64) - the go fmt package says it is '%T'", path, res)
	}
}

// GetObjectAtKey is a handy helper that will dig through a map and find something at the defined key. The returned data is not type checked.
/*
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
	ctx := context.TODO()
	hash := data

	parts := strings.Split(path, "/")
	part := ""
	seen := ""
	tflog.Debug(ctx, "GetObjectAtKey: Locating results_key", map[string]interface{}{"parts": parts})

	for len(parts) > 1 {
		/* AKA, Slice...*/
		part, parts = parts[0], parts[1:]

		/* Protect against double slashes by mistake */
		if part == "" {
			continue
		}

		/* See if this key exists in the hash at this point */
		if _, ok := hash[part]; ok {
			tflog.Debug(ctx, "GetObjectAtKey: key exists", map[string]interface{}{"key": part})
			seen += "/" + part
			if tmp, ok := hash[part].(map[string]interface{}); ok {
				tflog.Debug(ctx, "GetObjectAtKey:    is a map", map[string]interface{}{"key": part})
				hash = tmp
			} else if tmp, ok := hash[part].([]interface{}); ok {
				tflog.Debug(ctx, "GetObjectAtKey:    is a list", map[string]interface{}{"key": part})
				mapString := make(map[string]interface{})
				for key, value := range tmp {
					strKey := fmt.Sprintf("%v", key)
					mapString[strKey] = value
				}
				hash = mapString
			} else {
				tflog.Debug(ctx, "GetObjectAtKey:    is not a map", map[string]interface{}{"key": part, "type": fmt.Sprintf("%T", hash[part])})
				return nil, fmt.Errorf("GetObjectAtKey: Object at '%s' is not a map. Is this the right path?", seen)
			}
		} else {
			tflog.Debug(ctx, "GetObjectAtKey:  MISSING", map[string]interface{}{"key": part})

			return nil, fmt.Errorf("GetObjectAtKey: Failed to find '%s' in returned data structure after finding '%s'. Available: %s", part, seen, strings.Join(GetKeys(hash), ","))
		}
	} /* End Loop through parts */

	/* We have found the containing map of the value we want */
	part = parts[0] /* One last time */
	if _, ok := hash[part]; !ok {
		tflog.Debug(ctx, "GetObjectAtKey:  MISSING", map[string]interface{}{"key": part, "available": strings.Join(GetKeys(hash), ",")})
		return nil, fmt.Errorf("GetObjectAtKey: Resulting map at '%s' does not have key '%s'. Available: %s", seen, part, strings.Join(GetKeys(hash), ","))
	}

	tflog.Debug(ctx, "GetObjectAtKey:  exists", map[string]interface{}{"key": part, "value": hash[part]})

	return hash[part], nil
}

// GetKeys is a handy helper to just dump the keys of a map into a slice
func GetKeys(hash map[string]interface{}) []string {
	keys := make([]string, 0)
	for k := range hash {
		keys = append(keys, k)
	}
	return keys
}

// GetEnvOrDefault is a helper function that returns the value of the given environment variable, if one exists, or the default value
func GetEnvOrDefault(k string, defaultvalue string) string {
	v := os.Getenv(k)
	if v == "" {
		return defaultvalue
	}
	return v
}

// ExpandStringSet converts a slice of interfaces to a slice of strings
func ExpandStringSet(configured []interface{}) []string {
	return expandStringList(configured)
}

func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, v.(string))
		}
	}
	return vs
}
