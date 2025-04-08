package restapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type apiObjectOpts struct {
	path          string
	getPath       string
	postPath      string
	putPath       string
	createMethod  string
	readMethod    string
	readData      string
	updateMethod  string
	updateData    string
	destroyMethod string
	destroyData   string
	deletePath    string
	searchPath    string
	queryString   string
	debug         bool
	readSearch    map[string]string
	id            string
	idAttribute   string
	data          string
	filterKeys    []string
}

/*APIObject is the state holding struct for a restapi_object resource*/
type APIObject struct {
	apiClient        *APIClient
	getPath          string
	postPath         string
	putPath          string
	createMethod     string
	readMethod       string
	updateMethod     string
	destroyMethod    string
	deletePath       string
	searchPath       string
	queryString      string
	debug            bool
	readSearch       map[string]string
	id               string
	idAttribute      string
	skipStateRefresh bool     /* Used for testing to skip final state refresh */
	filterKeys       []string /* Keys to filter out when parsing the API response */

	/* Set internally */
	data        map[string]interface{} /* Data as managed by the user */
	readData    map[string]interface{} /* Read data as managed by the user */
	updateData  map[string]interface{} /* Update data as managed by the user */
	destroyData map[string]interface{} /* Destroy data as managed by the user */
	apiData     map[string]interface{} /* Data as available from the API */
	apiResponse string
}

// NewAPIObject makes an APIobject to manage a RESTful object in an API
func NewAPIObject(iClient *APIClient, opts *apiObjectOpts) (*APIObject, error) {
	if opts.debug {
		log.Printf("api_object.go: Constructing debug api_object\n")
		log.Printf(" id: %s\n", opts.id)
	}

	/* id_attribute can be set either on the client (to apply for all calls with the server)
	   or on a per object basis (for only calls to this kind of object).
	   Permit overridding from the API client here by using the client-wide value only
	   if a per-object value is not set */
	if opts.idAttribute == "" {
		opts.idAttribute = iClient.idAttribute
	}

	if opts.createMethod == "" {
		opts.createMethod = iClient.createMethod
	}
	if opts.readMethod == "" {
		opts.readMethod = iClient.readMethod
	}
	if opts.readData == "" {
		opts.readData = iClient.readData
	}
	if opts.updateMethod == "" {
		opts.updateMethod = iClient.updateMethod
	}
	if opts.updateData == "" {
		opts.updateData = iClient.updateData
	}
	if opts.destroyMethod == "" {
		opts.destroyMethod = iClient.destroyMethod
	}
	if opts.destroyData == "" {
		opts.destroyData = iClient.destroyData
	}
	if opts.postPath == "" {
		opts.postPath = opts.path
	}
	if opts.getPath == "" {
		opts.getPath = opts.path + "/{id}"
	}
	if opts.putPath == "" {
		opts.putPath = opts.path + "/{id}"
	}
	if opts.deletePath == "" {
		opts.deletePath = opts.path + "/{id}"
	}
	if opts.searchPath == "" {
		opts.searchPath = opts.path
	}

	obj := APIObject{
		apiClient:     iClient,
		getPath:       opts.getPath,
		postPath:      opts.postPath,
		putPath:       opts.putPath,
		createMethod:  opts.createMethod,
		readMethod:    opts.readMethod,
		updateMethod:  opts.updateMethod,
		destroyMethod: opts.destroyMethod,
		deletePath:    opts.deletePath,
		searchPath:    opts.searchPath,
		queryString:   opts.queryString,
		debug:         opts.debug,
		readSearch:    opts.readSearch,
		id:            opts.id,
		idAttribute:   opts.idAttribute,
		filterKeys:    opts.filterKeys,
		data:          make(map[string]interface{}),
		readData:      make(map[string]interface{}),
		updateData:    make(map[string]interface{}),
		destroyData:   make(map[string]interface{}),
		apiData:       make(map[string]interface{}),
	}

	if opts.data != "" {
		if opts.debug {
			log.Printf("api_object.go: Parsing data: '%s'", opts.data)
		}

		err := json.Unmarshal([]byte(opts.data), &obj.data)
		if err != nil {
			return &obj, fmt.Errorf("api_object.go: error parsing data provided: %v", err.Error())
		}

		/* Opportunistically set the object's ID if it is provided in the data.
		   If it is not set, we will get it later in synchronize_state */
		if obj.id == "" {
			var tmp string
			tmp, err := GetStringAtKey(obj.data, obj.idAttribute, obj.debug)
			if err == nil {
				if opts.debug {
					log.Printf("api_object.go: opportunisticly set id from data provided.")
				}
				obj.id = tmp
			} else if !obj.apiClient.writeReturnsObject && !obj.apiClient.createReturnsObject && obj.searchPath == "" {
				/* If the id is not set and we cannot obtain it
				   later, error out to be safe */
				return &obj, fmt.Errorf("provided data does not have %s attribute for the object's id and the client is not configured to read the object from a POST response; without an id, the object cannot be managed", obj.idAttribute)
			}
		}
	}

	if opts.readData != "" {
		if opts.debug {
			log.Printf("api_object.go: Parsing read data: '%s'", opts.readData)
		}

		err := json.Unmarshal([]byte(opts.readData), &obj.readData)
		if err != nil {
			return &obj, fmt.Errorf("api_object.go: error parsing read data provided: %v", err.Error())
		}
	}

	if opts.updateData != "" {
		if opts.debug {
			log.Printf("api_object.go: Parsing update data: '%s'", opts.updateData)
		}

		err := json.Unmarshal([]byte(opts.updateData), &obj.updateData)
		if err != nil {
			return &obj, fmt.Errorf("api_object.go: error parsing update data provided: %v", err.Error())
		}
	}

	if opts.destroyData != "" {
		if opts.debug {
			log.Printf("api_object.go: Parsing destroy data: '%s'", opts.destroyData)
		}

		err := json.Unmarshal([]byte(opts.destroyData), &obj.destroyData)
		if err != nil {
			return &obj, fmt.Errorf("api_object.go: error parsing destroy data provided: %v", err.Error())
		}
	}

	if opts.debug {
		log.Printf("api_object.go: Constructed object: %s", obj.toString())
	}
	return &obj, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *APIObject) toString() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("id: %s\n", obj.id))
	buffer.WriteString(fmt.Sprintf("get_path: %s\n", obj.getPath))
	buffer.WriteString(fmt.Sprintf("post_path: %s\n", obj.postPath))
	buffer.WriteString(fmt.Sprintf("put_path: %s\n", obj.putPath))
	buffer.WriteString(fmt.Sprintf("delete_path: %s\n", obj.deletePath))
	buffer.WriteString(fmt.Sprintf("query_string: %s\n", obj.queryString))
	buffer.WriteString(fmt.Sprintf("create_method: %s\n", obj.createMethod))
	buffer.WriteString(fmt.Sprintf("read_method: %s\n", obj.readMethod))
	buffer.WriteString(fmt.Sprintf("update_method: %s\n", obj.updateMethod))
	buffer.WriteString(fmt.Sprintf("destroy_method: %s\n", obj.destroyMethod))
	buffer.WriteString(fmt.Sprintf("debug: %t\n", obj.debug))
	buffer.WriteString(fmt.Sprintf("read_search: %s\n", spew.Sdump(obj.readSearch)))
	buffer.WriteString(fmt.Sprintf("filter_keys: %s\n", spew.Sdump(obj.filterKeys)))
	buffer.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(obj.data)))
	buffer.WriteString(fmt.Sprintf("read_data: %s\n", spew.Sdump(obj.readData)))
	buffer.WriteString(fmt.Sprintf("update_data: %s\n", spew.Sdump(obj.updateData)))
	buffer.WriteString(fmt.Sprintf("destroy_data: %s\n", spew.Sdump(obj.destroyData)))
	buffer.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(obj.apiData)))
	return buffer.String()
}

/*
Centralized function to ensure that our data as managed by

	the api_object is updated with data that has come back from
	the API
*/
func (obj *APIObject) updateState(state string) error {
	if obj.debug {
		log.Printf("api_object.go: Updating API object state to '%s'\n", state)
	}

	/* Other option - Decode as JSON Numbers instead of golang datatypes
	d := json.NewDecoder(strings.NewReader(res_str))
	d.UseNumber()
	err = d.Decode(&obj.api_data)
	*/
	err := json.Unmarshal([]byte(state), &obj.apiData)
	if err != nil {
		return err
	}

	/* Store response body for parsing via jsondecode() */
	obj.apiResponse = state

	/* Filter out keys that should be excluded from state tracking */
	if len(obj.filterKeys) > 0 {
		if obj.debug {
			log.Printf("api_object.go: Filtering out keys: %v", obj.filterKeys)
		}
		// Use iterative approach to filter keys at any level
		obj.filterKeysFromData()
	}

	/* A usable ID was not passed (in constructor or here),
	   so we have to guess what it is from the data structure */
	if obj.id == "" {
		val, err := GetStringAtKey(obj.apiData, obj.idAttribute, obj.debug)
		if err != nil {
			return fmt.Errorf("api_object.go: Error extracting ID from data element: %s", err)
		}
		obj.id = val
	} else if obj.debug {
		log.Printf("api_object.go: Not updating id. It is already set to '%s'\n", obj.id)
	}

	/* Any keys that come from the data we want to copy are done here */
	if len(obj.apiClient.copyKeys) > 0 {
		for _, key := range obj.apiClient.copyKeys {
			if obj.debug {
				log.Printf("api_object.go: Copying key '%s' from api_data (%v) to data (%v)\n", key, obj.apiData[key], obj.data[key])
			}
			obj.data[key] = obj.apiData[key]
		}
	} else if obj.debug {
		log.Printf("api_object.go: copy_keys is empty - not attempting to copy data")
	}

	if obj.debug {
		log.Printf("api_object.go: final object after synchronization of state:\n%+v\n", obj.toString())
	}
	return err
}

func (obj *APIObject) createObject() error {
	/* Failsafe: The constructor should prevent this situation, but
	   protect here also. If no id is set, and the API does not respond
	   with the id of whatever gets created, we have no way to know what
	   the object's id will be. Abandon this attempt */
	if obj.id == "" && !obj.apiClient.writeReturnsObject && !obj.apiClient.createReturnsObject {
		return fmt.Errorf("provided object does not have an id set and the client is not configured to read the object from a POST or PUT response; please set write_returns_object to true, or include an id in the object's data")
	}

	b, _ := json.Marshal(obj.data)

	postPath := obj.postPath
	if obj.queryString != "" {
		if obj.debug {
			log.Printf("api_object.go: Adding query string '%s'", obj.queryString)
		}
		postPath = fmt.Sprintf("%s?%s", obj.postPath, obj.queryString)
	}

	resultString, err := obj.apiClient.sendRequest(obj.createMethod, strings.Replace(postPath, "{id}", obj.id, -1), string(b))
	if err != nil {
		return err
	}

	/* We will need to sync state as well as get the object's ID */
	if obj.apiClient.writeReturnsObject || obj.apiClient.createReturnsObject {
		if obj.debug {
			log.Printf("api_object.go: Parsing response from POST to update internal structures (write_returns_object=%t, create_returns_object=%t)...\n",
				obj.apiClient.writeReturnsObject, obj.apiClient.createReturnsObject)
		}
		err = obj.updateState(resultString)
		/* Yet another failsafe. In case something terrible went wrong internally,
		   bail out so the user at least knows that the ID did not get set. */
		if obj.id == "" {
			return fmt.Errorf("internal validation failed; object ID is not set, but *may* have been created; this should never happen")
		}
	} else {
		if obj.debug {
			log.Printf("api_object.go: Requesting created object from API (write_returns_object=%t, create_returns_object=%t)...\n",
				obj.apiClient.writeReturnsObject, obj.apiClient.createReturnsObject)
		}
		err = obj.readObject()
	}
	return err
}

func (obj *APIObject) readObject() error {
	if obj.id == "" {
		return fmt.Errorf("cannot read an object unless the ID has been set")
	}

	getPath := obj.getPath
	if obj.queryString != "" {
		if obj.debug {
			log.Printf("api_object.go: Adding query string '%s'", obj.queryString)
		}
		getPath = fmt.Sprintf("%s?%s", obj.getPath, obj.queryString)
	}

	send := ""
	if len(obj.readData) > 0 {
		readData, _ := json.Marshal(obj.readData)
		send = string(readData)
		if obj.debug {
			log.Printf("api_object.go: Using read data '%s'", send)
		}
	}

	resultString, err := obj.apiClient.sendRequest(obj.readMethod, strings.Replace(getPath, "{id}", obj.id, -1), send)
	if err != nil {
		if strings.Contains(err.Error(), "unexpected response code '404'") {
			log.Printf("api_object.go: 404 error while refreshing state for '%s' at path '%s'. Removing from state.", obj.id, obj.getPath)
			obj.id = ""
			return nil
		}
		return err
	}

	searchKey := obj.readSearch["search_key"]
	searchValue := obj.readSearch["search_value"]

	if searchKey != "" && searchValue != "" {

		obj.searchPath = strings.Replace(obj.getPath, "{id}", obj.id, -1)

		queryString := obj.readSearch["query_string"]
		if obj.queryString != "" {
			if obj.debug {
				log.Printf("api_object.go: Adding query string '%s'", obj.queryString)
			}
			queryString = fmt.Sprintf("%s&%s", obj.readSearch["query_string"], obj.queryString)
		}
		resultsKey := obj.readSearch["results_key"]
		objFound, err := obj.findObject(queryString, searchKey, searchValue, resultsKey)
		if err != nil {
			obj.id = ""
			return nil
		}
		objFoundString, _ := json.Marshal(objFound)
		return obj.updateState(string(objFoundString))
	}

	return obj.updateState(resultString)
}

func (obj *APIObject) updateObject() error {
	if obj.id == "" {
		return fmt.Errorf("cannot update an object unless the ID has been set")
	}

	log.Printf("api_object.go: Updating object with id '%s' using method '%s'", obj.id, obj.updateMethod)
	// For Midpoint integration, we need to detect changes and send them via PATCH
	if obj.updateMethod == "PATCH" {
		// If apiData is empty, fetch current state to compare with desired state
		if len(obj.apiData) == 0 {
			err := obj.readObject()
			if err != nil {
				return fmt.Errorf("failed to read object for PATCH operation: %v", err)
			}
		}

		// We have apiData (current) and obj.data (desired)
		// Now calculate what changed and form appropriate PATCH requests
		return obj.patchMidpointObject()
	}

	// Original PUT behavior
	send := ""
	if len(obj.updateData) > 0 {
		updateData, _ := json.Marshal(obj.updateData)
		send = string(updateData)
		if obj.debug {
			log.Printf("api_object.go: Using update data '%s'", send)
		}
	} else {
		b, _ := json.Marshal(obj.data)
		send = string(b)
	}

	putPath := obj.putPath
	if obj.queryString != "" {
		if obj.debug {
			log.Printf("api_object.go: Adding query string '%s'", obj.queryString)
		}
		putPath = fmt.Sprintf("%s?%s", obj.putPath, obj.queryString)
	}

	resultString, err := obj.apiClient.sendRequest(obj.updateMethod, strings.Replace(putPath, "{id}", obj.id, -1), send)
	if err != nil {
		return err
	}

	if obj.apiClient.writeReturnsObject {
		if obj.debug {
			log.Printf("api_object.go: Parsing response from PUT to update internal structures (write_returns_object=true)...\n")
		}
		err = obj.updateState(resultString)
	} else {
		if obj.debug {
			log.Printf("api_object.go: Requesting updated object from API (write_returns_object=false)...\n")
		}
		err = obj.readObject()
	}
	return err
}

func (obj *APIObject) deleteObject() error {
	if obj.id == "" {
		log.Printf("WARNING: Attempting to delete an object that has no id set. Assuming this is OK.\n")
		return nil
	}

	deletePath := obj.deletePath
	if obj.queryString != "" {
		if obj.debug {
			log.Printf("api_object.go: Adding query string '%s'", obj.queryString)
		}
		deletePath = fmt.Sprintf("%s?%s", obj.deletePath, obj.queryString)
	}

	send := ""
	if len(obj.destroyData) > 0 {
		destroyData, _ := json.Marshal(obj.destroyData)
		send = string(destroyData)
		if obj.debug {
			log.Printf("api_object.go: Using destroy data '%s'", string(destroyData))
		}
	}

	_, err := obj.apiClient.sendRequest(obj.destroyMethod, strings.Replace(deletePath, "{id}", obj.id, -1), send)
	if err != nil {
		return err
	}

	return nil
}

// patchMidpointObject calculates differences between current and desired state
// and makes PATCH requests for each modification needed using Midpoint's ObjectModificationType format
func (obj *APIObject) patchMidpointObject() error {
	if obj.debug {
		log.Printf("api_object.go: Calculating differences for PATCH operation")
	}

	// Track if we made any changes
	changesApplied := false

	var rootKey string
	var rootInnerMap map[string]interface{}

	if len(obj.data) > 0 {
		for key, innerMapInterface := range obj.data {
			rootKey = key
			if innerMap, ok := innerMapInterface.(map[string]interface{}); ok {
				rootInnerMap = innerMap
			} else {
				// If it's not a map, use the whole data object
				rootKey = ""
				rootInnerMap = obj.data
			}
			break
		}
	} else {
		rootKey = ""
		rootInnerMap = obj.data
	}

	// Process each top-level key in the desired state
	for key, desiredValue := range rootInnerMap {

		var currentValue interface{}
		var exists bool

		if rootKey != "" {
			if apiDataMap, ok := obj.apiData[rootKey].(map[string]interface{}); ok {
				currentValue, exists = apiDataMap[key]
			} else {
				exists = false
			}
		} else {
			currentValue, exists = obj.apiData[key]
		}

		// Handle additions and modifications
		if !exists {
			// Key doesn't exist in current state - add it
			if obj.debug {
				log.Printf("api_object.go: Adding new attribute '%s'/'%s'", rootKey, key)
			}

			err := obj.sendMidpointPatch("add", key, desiredValue)
			if err != nil {
				return fmt.Errorf("failed to add attribute '%s'/'%s': %v", rootKey, key, err)
			}
			changesApplied = true
		} else if !reflect.DeepEqual(currentValue, desiredValue) {
			// Key exists but value is different - replace it
			if obj.debug {

				log.Printf("api_object.go: Replacing attribute '%s'/'%s'", rootKey, key)
				log.Printf("api_object.go: OLD is %s", currentValue)
				log.Printf("api_object.go: NEW is %s", desiredValue)
			}

			err := obj.sendMidpointPatch("replace", key, desiredValue)
			if err != nil {
				return fmt.Errorf("failed to replace attribute '%s'/'%s': %v", rootKey, key, err)
			}
			changesApplied = true
		}
	}

	// Check for deletions - keys that exist in current state but not in desired state
	for key := range obj.apiData {
		if _, exists := obj.data[key]; !exists {
			// Skip the ID attribute - we don't want to delete that
			if key == obj.idAttribute {
				continue
			}

			if obj.debug {
				log.Printf("api_object.go: Deleting attribute '%s'/'%s'", rootKey, key)
			}

			err := obj.sendMidpointPatch("delete", key, nil)
			if err != nil {
				return fmt.Errorf("failed to delete attribute '%s'/'%s': %v", rootKey, key, err)
			}
			changesApplied = true
		}
	}

	// If we made any changes, read the object to ensure state is current
	if !changesApplied && !obj.skipStateRefresh {
		if obj.debug {
			log.Printf("api_object.go: refreshing state")
		}
		return obj.readObject()
	}

	return nil

}

// sendMidpointPatch sends a single PATCH request for the specified modification
func (obj *APIObject) sendMidpointPatch(modificationType string, path string, value interface{}) error {
	// Build the ObjectModificationType payload
	modification := make(map[string]interface{})

	// Structure for the itemDelta
	itemDelta := make(map[string]interface{})
	itemDelta["modificationType"] = modificationType
	itemDelta["path"] = path

	// Add value for add and replace operations
	if modificationType != "delete" && value != nil {
		itemDelta["value"] = value
	}

	// Complete the structure
	modification["objectModification"] = map[string]interface{}{
		"itemDelta": itemDelta,
	}

	// Convert to JSON
	modificationJSON, err := json.Marshal(modification)
	if err != nil {
		return fmt.Errorf("failed to marshal modification to JSON: %v", err)
	}

	if obj.debug {
		log.Printf("api_object.go: Sending PATCH with payload: %s", string(modificationJSON))
	}

	// Construct the PATCH path
	patchPath := obj.putPath // reuse the PUT path
	if obj.queryString != "" {
		patchPath = fmt.Sprintf("%s?%s", patchPath, obj.queryString)
	}

	// Send the PATCH request
	resultString, err := obj.apiClient.sendRequest("PATCH", strings.Replace(patchPath, "{id}", obj.id, -1), string(modificationJSON))
	if err != nil {
		return err
	}

	// Update internal state if the API returns the updated object
	if obj.apiClient.writeReturnsObject {
		if obj.debug {
			log.Printf("api_object.go: Parsing response from PATCH to update internal structures (write_returns_object=true)...\n")
		}
		return obj.updateState(resultString)
	}

	return nil
}

// filterKeysFromData iteratively filters out specified keys at any level in the JSON hierarchy
func (obj *APIObject) filterKeysFromData() {
	// Use a stack-based approach to traverse the JSON structure
	type processItem struct {
		path string
		data interface{}
	}

	stack := []processItem{
		{path: "", data: obj.apiData},
	}

	for len(stack) > 0 {
		// Pop item from stack
		n := len(stack) - 1
		current := stack[n]
		stack = stack[:n]

		// Process based on type
		switch v := current.data.(type) {
		case map[string]interface{}:
			// For maps, check each key
			for key, value := range v {
				// Check if this key should be filtered
				shouldFilter := false
				for _, filterKey := range obj.filterKeys {
					if key == filterKey {
						shouldFilter = true
						if obj.debug {
							log.Printf("api_object.go: Filtering out key '%s' at path '%s'", key, current.path)
						}
						delete(v, key)
						break
					}
				}

				if !shouldFilter {
					// Add to stack for further processing if it's a container type
					itemPath := key
					if current.path != "" {
						itemPath = current.path + "." + key
					}

					switch value.(type) {
					case map[string]interface{}, []interface{}:
						stack = append(stack, processItem{path: itemPath, data: value})
					}
				}
			}
		case []interface{}:
			// For arrays, check each element
			for i, item := range v {
				itemPath := fmt.Sprintf("%s[%d]", current.path, i)

				// Add container types to stack for further processing
				switch item.(type) {
				case map[string]interface{}, []interface{}:
					stack = append(stack, processItem{path: itemPath, data: item})
				}
			}
		}
	}
}

func (obj *APIObject) findObject(queryString string, searchKey string, searchValue string, resultsKey string) (map[string]interface{}, error) {
	var objFound map[string]interface{}
	var dataArray []interface{}
	var ok bool

	/*
	   Issue a GET to the base path and expect results to come back
	*/
	searchPath := obj.searchPath
	if queryString != "" {
		if obj.debug {
			log.Printf("api_object.go: Adding query string '%s'", queryString)
		}
		searchPath = fmt.Sprintf("%s?%s", obj.searchPath, queryString)
	}

	if obj.debug {
		log.Printf("api_object.go: Calling API on path '%s'", searchPath)
	}
	resultString, err := obj.apiClient.sendRequest(obj.apiClient.readMethod, searchPath, "")
	if err != nil {
		return objFound, err
	}

	/*
	   Parse it seeking JSON data
	*/
	if obj.debug {
		log.Printf("api_object.go: Response received... parsing")
	}
	var result interface{}
	err = json.Unmarshal([]byte(resultString), &result)
	if err != nil {
		return objFound, err
	}

	if resultsKey != "" {
		var tmp interface{}

		if obj.debug {
			log.Printf("api_object.go: Locating '%s' in the results", resultsKey)
		}

		/* First verify the data we got back is a hash */
		if _, ok = result.(map[string]interface{}); !ok {
			return objFound, fmt.Errorf("api_object.go: The results of a GET to '%s' did not return a hash. Cannot search within for results_key '%s'", searchPath, resultsKey)
		}

		tmp, err = GetObjectAtKey(result.(map[string]interface{}), resultsKey, obj.debug)
		if err != nil {
			return objFound, fmt.Errorf("api_object.go: Error finding results_key: %s", err)
		}
		if dataArray, ok = tmp.([]interface{}); !ok {
			return objFound, fmt.Errorf("api_object.go: The data at results_key location '%s' is not an array. It is a '%s'", resultsKey, reflect.TypeOf(tmp))
		}
	} else {
		if obj.debug {
			log.Printf("api_object.go: results_key is not set - coaxing data to array of interfaces")
		}
		if dataArray, ok = result.([]interface{}); !ok {
			return objFound, fmt.Errorf("api_object.go: The results of a GET to '%s' did not return an array. It is a '%s'. Perhaps you meant to add a results_key?", searchPath, reflect.TypeOf(result))
		}
	}

	/* Loop through all of the results seeking the specific record */
	for _, item := range dataArray {
		var hash map[string]interface{}

		if hash, ok = item.(map[string]interface{}); !ok {
			return objFound, fmt.Errorf("api_object.go: The elements being searched for data are not a map of key value pairs")
		}

		if obj.debug {
			log.Printf("api_object.go: Examining %v", hash)
			log.Printf("api_object.go:   Comparing '%s' to the value in '%s'", searchValue, searchKey)
		}

		tmp, err := GetStringAtKey(hash, searchKey, obj.debug)
		if err != nil {
			return objFound, (fmt.Errorf("failed to get the value of '%s' in the results array at '%s': %s", searchKey, resultsKey, err))
		}

		/* We found our record */
		if tmp == searchValue {
			objFound = hash
			obj.id, err = GetStringAtKey(hash, obj.idAttribute, obj.debug)
			if err != nil {
				return objFound, (fmt.Errorf("failed to find id_attribute '%s' in the record: %s", obj.idAttribute, err))
			}

			if obj.debug {
				log.Printf("api_object.go: Found ID '%s'", obj.id)
			}

			/* But there is no id attribute??? */
			if obj.id == "" {
				return objFound, (fmt.Errorf(fmt.Sprintf("The object for '%s'='%s' did not have the id attribute '%s', or the value was empty.", searchKey, searchValue, obj.idAttribute)))
			}
			break
		}
	}

	if obj.id == "" {
		return objFound, (fmt.Errorf("failed to find an object with the '%s' key = '%s' at %s", searchKey, searchValue, searchPath))
	}

	return objFound, nil
}
