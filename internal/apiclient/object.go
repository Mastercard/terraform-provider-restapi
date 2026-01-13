package restapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type APIObjectOpts struct {
	Path          string
	CreatePath    string
	CreateMethod  string
	ReadMethod    string
	ReadPath      string
	ReadData      string
	UpdateMethod  string
	UpdatePath    string
	UpdateData    string
	DestroyMethod string
	DestroyData   string
	DestroyPath   string
	SearchPath    string
	QueryString   string
	Debug         bool
	ReadSearch    map[string]string
	ID            string
	IDAttribute   string
	Data          string
}

// APIObject is the state holding struct for a restapi_object resource
type APIObject struct {
	apiClient     *APIClient
	createMethod  string
	createPath    string
	readMethod    string
	readPath      string
	updateMethod  string
	updatePath    string
	destroyMethod string
	deletePath    string
	searchPath    string
	queryString   string
	debug         bool
	readSearch    map[string]string
	ID            string
	IDAttribute   string

	// Set internally
	mux         sync.RWMutex           // Protects data and apiData fields
	data        map[string]interface{} // Data as managed by the user
	readData    map[string]interface{} // Data to send during Read operation
	updateData  map[string]interface{} // Data to send during Update operation
	destroyData map[string]interface{} // Data to send during Destroy operation
	apiData     map[string]interface{} // Data from the most recent read operation of the API object, as massaged to a map
	apiResponse string                 // Raw API response from most recent read operation
	searchPatch jsonpatch.Patch        // Pre-compiled JSON Patch for search_patch transformation
}

// NewAPIObject makes an APIobject to manage a RESTful object in an API
func NewAPIObject(iClient *APIClient, opts *APIObjectOpts) (*APIObject, error) {
	ctx := context.Background()
	tflog.Debug(ctx, "Constructing api_object", map[string]interface{}{"id": opts.ID})

	// id_attribute can be set either on the client (to apply for all calls with the server)
	// or on a per object basis (for only calls to this kind of object).
	// Permit overridding from the API client here by using the client-wide value only
	// if a per-object value is not set
	if opts.IDAttribute == "" {
		opts.IDAttribute = iClient.idAttribute
	}

	if opts.CreateMethod == "" {
		opts.CreateMethod = iClient.createMethod
	}
	if opts.ReadMethod == "" {
		opts.ReadMethod = iClient.readMethod
	}
	if opts.ReadData == "" {
		opts.ReadData = iClient.readData
	}
	if opts.UpdateMethod == "" {
		opts.UpdateMethod = iClient.updateMethod
	}
	if opts.UpdateData == "" {
		opts.UpdateData = iClient.updateData
	}
	if opts.DestroyMethod == "" {
		opts.DestroyMethod = iClient.destroyMethod
	}
	if opts.DestroyData == "" {
		opts.DestroyData = iClient.destroyData
	}
	if opts.CreatePath == "" {
		opts.CreatePath = opts.Path
	}
	if opts.ReadPath == "" {
		opts.ReadPath = appendIdToPath(opts.Path)
	}
	if opts.UpdatePath == "" {
		opts.UpdatePath = appendIdToPath(opts.Path)
	}
	if opts.DestroyPath == "" {
		opts.DestroyPath = appendIdToPath(opts.Path)
	}
	if opts.SearchPath == "" {
		opts.SearchPath = opts.Path
	}

	obj := APIObject{
		apiClient:     iClient,
		readPath:      opts.ReadPath,
		createPath:    opts.CreatePath,
		updatePath:    opts.UpdatePath,
		createMethod:  opts.CreateMethod,
		readMethod:    opts.ReadMethod,
		updateMethod:  opts.UpdateMethod,
		destroyMethod: opts.DestroyMethod,
		deletePath:    opts.DestroyPath,
		searchPath:    opts.SearchPath,
		queryString:   opts.QueryString,
		debug:         opts.Debug,
		readSearch:    opts.ReadSearch,
		ID:            opts.ID,
		IDAttribute:   opts.IDAttribute,
		data:          make(map[string]interface{}),
		readData:      nil,
		updateData:    nil,
		destroyData:   nil,
		apiData:       make(map[string]interface{}),
	}

	if opts.Data != "" {
		tflog.Debug(ctx, "Parsing data", map[string]interface{}{"data": opts.Data})

		err := json.Unmarshal([]byte(opts.Data), &obj.data)
		if err != nil {
			return &obj, fmt.Errorf("error parsing data provided: %v", err.Error())
		}

		// Opportunistically extract ID from provided data if present.
		// If not present, we'll attempt to get it later from the API response
		// (when write_returns_object or create_returns_object is true) or from search.
		if obj.ID == "" {
			var tmp string
			tmp, err := GetStringAtKey(ctx, obj.data, obj.IDAttribute)
			if err == nil {
				tflog.Debug(ctx, "opportunisticly set id from data provided", map[string]interface{}{"id": tmp})
				obj.ID = tmp
			} else if !obj.apiClient.writeReturnsObject && !obj.apiClient.createReturnsObject && obj.searchPath == "" {
				// If the id is not set and we cannot obtain it
				// later, error out to be safe
				return &obj, fmt.Errorf("provided data does not have %s attribute for the object's id and the client is not configured to read the object from a POST response; without an id, the object cannot be managed", obj.IDAttribute)
			}
		}
	}

	if opts.ReadData != "" {
		tflog.Debug(ctx, "Parsing read data", map[string]interface{}{"readData": opts.ReadData})

		err := json.Unmarshal([]byte(opts.ReadData), &obj.readData)
		if err != nil {
			return &obj, fmt.Errorf("error parsing read data provided: %v", err.Error())
		}
	}

	if opts.UpdateData != "" {
		tflog.Debug(ctx, "Parsing update data", map[string]interface{}{"updateData": opts.UpdateData})

		err := json.Unmarshal([]byte(opts.UpdateData), &obj.updateData)
		if err != nil {
			return &obj, fmt.Errorf("error parsing update data provided: %v", err.Error())
		}
	}

	if opts.DestroyData != "" {
		tflog.Debug(ctx, "Parsing destroy data", map[string]interface{}{"destroyData": opts.DestroyData})

		err := json.Unmarshal([]byte(opts.DestroyData), &obj.destroyData)
		if err != nil {
			return &obj, fmt.Errorf("error parsing destroy data provided: %v", err.Error())
		}
	}

	if searchPatchStr := opts.ReadSearch["search_patch"]; searchPatchStr != "" {
		tflog.Debug(ctx, "Compiling search_patch", map[string]interface{}{"search_patch": searchPatchStr})
		patch, err := jsonpatch.DecodePatch([]byte(searchPatchStr))
		if err != nil {
			return &obj, fmt.Errorf("failed to compile search_patch: %w", err)
		}
		obj.searchPatch = patch
	}

	tflog.Debug(ctx, "Constructed object", map[string]interface{}{"object": obj.String()})

	return &obj, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *APIObject) String() string {
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("id: %s\n", obj.ID))
	buffer.WriteString(fmt.Sprintf("get_path: %s\n", obj.readPath))
	buffer.WriteString(fmt.Sprintf("post_path: %s\n", obj.createPath))
	buffer.WriteString(fmt.Sprintf("put_path: %s\n", obj.updatePath))
	buffer.WriteString(fmt.Sprintf("delete_path: %s\n", obj.deletePath))
	buffer.WriteString(fmt.Sprintf("query_string: %s\n", obj.queryString))
	buffer.WriteString(fmt.Sprintf("create_method: %s\n", obj.createMethod))
	buffer.WriteString(fmt.Sprintf("read_method: %s\n", obj.readMethod))
	buffer.WriteString(fmt.Sprintf("update_method: %s\n", obj.updateMethod))
	buffer.WriteString(fmt.Sprintf("destroy_method: %s\n", obj.destroyMethod))
	buffer.WriteString(fmt.Sprintf("debug: %t\n", obj.debug))
	buffer.WriteString(fmt.Sprintf("read_search: %s\n", spew.Sdump(obj.readSearch)))
	buffer.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(obj.data)))
	buffer.WriteString(fmt.Sprintf("read_data: %s\n", spew.Sdump(obj.readData)))
	buffer.WriteString(fmt.Sprintf("update_data: %s\n", spew.Sdump(obj.updateData)))
	buffer.WriteString(fmt.Sprintf("destroy_data: %s\n", spew.Sdump(obj.destroyData)))
	buffer.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(obj.apiData)))
	return buffer.String()
}

// SetDataFromMap sets the object's internal state from a map
// This allows more fine-grained manipulation of an object's data, outside of reads to the API
func (obj *APIObject) SetDataFromMap(d map[string]interface{}) error {
	foundDataJSON, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal found data: %w", err)
	}
	return obj.updateInternalState(string(foundDataJSON))
}

// updateInternalState is a centralized function to ensure that our data as managed by
// the api_object is updated with data that has come back from the API
func (obj *APIObject) updateInternalState(state string) error {
	ctx := context.Background()
	tflog.Debug(ctx, "Updating API object state to '%s'\n", map[string]interface{}{"state": state})

	obj.mux.Lock()
	defer obj.mux.Unlock()

	err := json.Unmarshal([]byte(state), &obj.apiData)
	if err != nil {
		return err
	}

	obj.apiResponse = state

	// A usable ID was not passed (in constructor or here),
	// so we have to guess what it is from the data structure
	if obj.ID == "" {
		val, err := GetStringAtKey(ctx, obj.apiData, obj.IDAttribute)
		if err != nil {
			return fmt.Errorf("error extracting ID from data element: %s", err)
		}
		obj.ID = val
	} else {
		tflog.Debug(ctx, "Not updating id. It is already set to '%s'\n", map[string]interface{}{"id": obj.ID})
	}

	// Copy specific keys from API response back to our managed data.
	// This is useful when the API generates values (e.g., timestamps, computed fields, revision number)
	// that need to be included in subsequent requests.
	if len(obj.apiClient.copyKeys) > 0 {
		for _, key := range obj.apiClient.copyKeys {
			tflog.Debug(ctx, "Copying key from api_data to data\n", map[string]interface{}{"key": key, "new": obj.apiData[key], "old": obj.data[key]})
			obj.data[key] = obj.apiData[key]
		}
	} else {
		tflog.Debug(ctx, "copy_keys is empty - not attempting to copy data", nil)
	}

	return err
}

func (obj *APIObject) CreateObject(ctx context.Context) error {
	// Failsafe: The constructor should prevent this situation, but
	// protect here also. If no id is set, and the API does not respond
	// with the id of whatever gets created, we have no way to know what
	// the object's id will be. Abandon this attempt
	if obj.ID == "" && !obj.apiClient.writeReturnsObject && !obj.apiClient.createReturnsObject {
		return fmt.Errorf("provided object does not have an id set and the client is not configured to read the object from a POST or PUT response; please set write_returns_object to true, or include an id in the object's data")
	}

	obj.mux.RLock()
	b, _ := json.Marshal(obj.data)
	obj.mux.RUnlock()

	postPath := obj.createPath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		postPath = fmt.Sprintf("%s?%s", obj.createPath, obj.queryString)
	}

	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.createMethod, strings.Replace(postPath, "{id}", obj.ID, -1), string(b), obj.debug)
	if err != nil {
		return err
	}

	// We will need to sync state as well as get the object's ID
	if obj.apiClient.writeReturnsObject || obj.apiClient.createReturnsObject {
		tflog.Debug(ctx, "Parsing response from POST to update internal structures", map[string]interface{}{
			"write_returns_object":  obj.apiClient.writeReturnsObject,
			"create_returns_object": obj.apiClient.createReturnsObject,
		})
		err = obj.updateInternalState(resultString)
		// Yet another failsafe. In case something terrible went wrong internally,
		// bail out so the user at least knows that the ID did not get set.
		if obj.ID == "" {
			return fmt.Errorf("internal validation failed; object ID is not set, but *may* have been created; this should never happen")
		}
	} else {
		tflog.Debug(ctx, "Requesting created object from API", map[string]interface{}{
			"write_returns_object":  obj.apiClient.writeReturnsObject,
			"create_returns_object": obj.apiClient.createReturnsObject,
		})
		err = obj.ReadObject(ctx)
	}
	return err
}

func (obj *APIObject) ReadObject(ctx context.Context) error {
	if obj.ID == "" {
		return fmt.Errorf("cannot read an object unless the ID has been set")
	}

	// If read_search is configured, use FindObject to locate the resource by search criteria
	// instead of using the ID directly. This handles APIs that require searching for objects.
	searchKey := obj.readSearch["search_key"]
	searchValue := obj.readSearch["search_value"]

	// Support {id} placeholder substitution in search_value
	searchValue = strings.ReplaceAll(searchValue, "{id}", obj.ID)

	if searchKey != "" && searchValue != "" {
		// Ensure searchPath is set correctly. If not explicitly set, derive it from readPath
		// by removing the /{id} suffix to get the collection endpoint.
		if obj.searchPath == "" {
			obj.searchPath = strings.TrimSuffix(obj.readPath, "/{id}")
		}

		queryString := obj.readSearch["query_string"]
		// Merge object-level query string with search-specific query string
		if obj.queryString != "" {
			tflog.Debug(ctx, "Adding object-level query string to search", map[string]interface{}{"object_query_string": obj.queryString})
			if queryString != "" {
				queryString = fmt.Sprintf("%s&%s", queryString, obj.queryString)
			} else {
				queryString = obj.queryString
			}
		}

		searchData := ""
		if len(obj.readSearch["search_data"]) > 0 {
			tmpData, _ := json.Marshal(obj.readSearch["search_data"])
			searchData = string(tmpData)
			tflog.Debug(ctx, "Using search data", map[string]interface{}{"search_data": searchData})
		}

		resultsKey := obj.readSearch["results_key"]
		objFound, err := obj.FindObject(ctx, queryString, searchKey, searchValue, resultsKey, searchData)
		if err != nil || objFound == nil {
			// Object not found in search results - treat as deleted, remove from state
			tflog.Info(ctx, "Search did not find object", map[string]interface{}{"search_key": searchKey, "search_value": searchValue})
			obj.ID = ""
			return nil
		}

		// Apply search_patch if configured
		if obj.searchPatch != nil {
			tflog.Debug(ctx, "Applying search_patch transformation")
			patchedObj, err := ApplyJSONPatch(ctx, objFound, obj.searchPatch)
			if err != nil {
				return fmt.Errorf("failed to apply search_patch: %w", err)
			}
			objFound = patchedObj
			tflog.Debug(ctx, "Successfully applied search_patch")
		}

		objFoundString, _ := json.Marshal(objFound)
		return obj.updateInternalState(string(objFoundString))
	}

	// Normal read path (no search configured)
	getPath := obj.readPath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		getPath = fmt.Sprintf("%s?%s", obj.readPath, obj.queryString)
	}

	send := ""
	if obj.readData != nil {
		readData, _ := json.Marshal(obj.readData)
		send = string(readData)
		tflog.Debug(ctx, "Using read data", map[string]interface{}{"read_data": send})
	}

	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.readMethod, strings.Replace(getPath, "{id}", obj.ID, -1), send, obj.debug)
	if err != nil {
		// 404 during refresh means the object was deleted outside Terraform.
		// Clear the ID to remove it from state gracefully.
		if strings.Contains(err.Error(), "unexpected response code '404'") {
			tflog.Warn(ctx, "404 error while refreshing state. Removing from state.", map[string]interface{}{"id": obj.ID, "path": obj.readPath})
			obj.ID = ""
			return nil
		}
		return err
	}

	return obj.updateInternalState(resultString)
}

func (obj *APIObject) UpdateObject(ctx context.Context) error {
	if obj.ID == "" {
		return fmt.Errorf("cannot update an object unless the ID has been set")
	}

	send := ""
	// If update_data is configured, use it for the update payload.
	// Otherwise, use the full managed data. This allows for partial updates.
	obj.mux.RLock()
	if obj.updateData != nil {
		updateData, _ := json.Marshal(obj.updateData)
		send = string(updateData)
		tflog.Debug(ctx, "Using update data", map[string]interface{}{"update_data": send})
	} else {
		b, _ := json.Marshal(obj.data)
		send = string(b)
	}
	obj.mux.RUnlock()

	putPath := obj.updatePath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		putPath = fmt.Sprintf("%s?%s", obj.updatePath, obj.queryString)
	}

	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.updateMethod, strings.Replace(putPath, "{id}", obj.ID, -1), send, obj.debug)
	if err != nil {
		return err
	}

	if obj.apiClient.writeReturnsObject {
		tflog.Debug(ctx, "Parsing response from PUT to update internal structures", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.updateInternalState(resultString)
	} else {
		tflog.Debug(ctx, "Requesting updated object from API", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.ReadObject(ctx)
	}
	return err
}

func (obj *APIObject) DeleteObject(ctx context.Context) error {
	if obj.ID == "" {
		tflog.Warn(ctx, "Attempting to delete an object that has no id set. Assuming this is OK.", map[string]interface{}{})
		return nil
	}

	deletePath := obj.deletePath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		deletePath = fmt.Sprintf("%s?%s", obj.deletePath, obj.queryString)
	}

	send := ""
	if obj.destroyData != nil {
		destroyData, _ := json.Marshal(obj.destroyData)
		send = string(destroyData)
		tflog.Debug(ctx, "Using destroy data", map[string]interface{}{"destroy_data": string(destroyData)})
	}

	_, code, err := obj.apiClient.SendRequest(ctx, obj.destroyMethod, strings.Replace(deletePath, "{id}", obj.ID, -1), send, obj.debug)
	if err != nil {
		// 404 (Not Found) or 410 (Gone) during delete is acceptable -
		// the object is already gone, which is the desired end state.
		if code == http.StatusNotFound || code == http.StatusGone {
			tflog.Warn(ctx, "404/410 error while deleting object. Assuming already deleted.", map[string]interface{}{"id": obj.ID, "path": obj.deletePath})
			err = nil
		}
	}

	return err
}

func (obj *APIObject) FindObject(ctx context.Context, queryString string, searchKey string, searchValue string, resultsKey string, searchData string) (map[string]interface{}, error) {
	var objFound map[string]interface{}
	var dataArray []interface{}
	var ok bool

	// Issue a GET to the base path and expect results to come back
	searchPath := obj.searchPath
	if queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": queryString})
		searchPath = fmt.Sprintf("%s?%s", obj.searchPath, queryString)
	}

	tflog.Debug(ctx, "Calling API on path", map[string]interface{}{"path": searchPath})
	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.apiClient.readMethod, searchPath, searchData, obj.debug)
	if err != nil {
		return nil, err
	}

	// Parse it seeking JSON data
	tflog.Debug(ctx, "Response received... parsing", nil)
	var result interface{}
	err = json.Unmarshal([]byte(resultString), &result)
	if err != nil {
		return nil, err
	}

	if resultsKey != "" {
		var tmp interface{}

		tflog.Debug(ctx, "Locating results_key in the results", map[string]interface{}{"results_key": resultsKey})

		// results_key points to a nested location in the response where the array of objects lives.
		// For example, if the API returns {"data": [{...}, {...}]}, results_key would be "data".
		if _, ok = result.(map[string]interface{}); !ok {
			return nil, fmt.Errorf("the results of a GET to '%s' did not return a map. Cannot search within for results_key '%s'", searchPath, resultsKey)
		}

		tmp, err = GetObjectAtKey(ctx, result.(map[string]interface{}), resultsKey)
		if err != nil {
			return nil, fmt.Errorf("error finding results_key: %s", err)
		}
		if dataArray, ok = tmp.([]interface{}); !ok {
			return nil, fmt.Errorf("the data at results_key location '%s' is not an array. It is a '%s'", resultsKey, reflect.TypeOf(tmp))
		}
	} else {
		tflog.Debug(ctx, "results_key is not set - coaxing data to array of interfaces", nil)
		if dataArray, ok = result.([]interface{}); !ok {
			return nil, fmt.Errorf("the results of a GET to '%s' did not return an array. It is a '%s'. Perhaps you meant to add a results_key?", searchPath, reflect.TypeOf(result))
		}
	}

	// Loop through all of the results seeking the specific record
	for _, item := range dataArray {
		var hash map[string]interface{}

		if hash, ok = item.(map[string]interface{}); !ok {
			return nil, fmt.Errorf("the elements being searched for data are not a map of key value pairs")
		}

		tflog.Debug(ctx, "Examining item in results array", map[string]interface{}{"item": hash})
		tflog.Debug(ctx, "Comparing search value to item value", map[string]interface{}{"search_value": searchValue, "search_key": searchKey})

		tmp, err := GetStringAtKey(ctx, hash, searchKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get the value of '%s' in the results array at '%s': %s", searchKey, resultsKey, err)
		}

		// We found our record
		if tmp == searchValue {
			objFound = hash
			obj.ID, err = GetStringAtKey(ctx, hash, obj.IDAttribute)
			if err != nil {
				return nil, fmt.Errorf("failed to find id_attribute '%s' in the record: %s", obj.IDAttribute, err)
			}

			tflog.Debug(ctx, "Found ID '%s'", map[string]interface{}{"id": obj.ID})

			// But there is no id attribute???
			if obj.ID == "" {
				return nil, fmt.Errorf("the object for '%s'='%s' did not have the id attribute '%s', or the value was empty", searchKey, searchValue, obj.IDAttribute)
			}
			break
		}
	}

	if obj.ID == "" {
		return nil, fmt.Errorf("failed to find an object with the '%s' key = '%s' at %s", searchKey, searchValue, searchPath)
	}

	return objFound, nil
}

// GetApiData returns a copy of the api_data map from the APIObject
func (obj *APIObject) GetApiData() map[string]string {
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	apiData := make(map[string]string)
	for k, v := range obj.apiData {
		apiData[k] = fmt.Sprintf("%v", v)
	}
	return apiData
}

// GetApiResponse returns a copy of the raw API response from the APIObject
func (obj *APIObject) GetApiResponse() string {
	return obj.apiResponse
}

// GetReadSearch returns a copy of the read_search configuration
func (obj *APIObject) GetReadSearch() map[string]string {
	if obj.readSearch == nil {
		return nil
	}
	readSearch := make(map[string]string)
	for k, v := range obj.readSearch {
		readSearch[k] = v
	}
	return readSearch
}

// appendIdToPath appends /{id} to a path in a robust way that handles query strings.
// If the path already contains {id}, it returns the path unchanged.
// If the path contains query parameters, it inserts /{id} before the query string.
// Otherwise, it appends /{id} to the end of the path.
func appendIdToPath(path string) string {
	if strings.Contains(path, "{id}") {
		return path
	}
	if strings.Contains(path, "?") {
		parts := strings.SplitN(path, "?", 2)
		return parts[0] + "/{id}?" + parts[1]
	}
	return path + "/{id}"
}
