package restapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type APIObjectOpts struct {
	Path          string
	GetPath       string
	PostPath      string
	PutPath       string
	CreateMethod  string
	ReadMethod    string
	ReadData      string
	UpdateMethod  string
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
	getPath       string
	postPath      string
	putPath       string
	createMethod  string
	readMethod    string
	updateMethod  string
	destroyMethod string
	deletePath    string
	searchPath    string
	queryString   string
	debug         bool
	readSearch    map[string]string
	ID            string
	IDAttribute   string

	// Set internally
	data        map[string]interface{} // Data as managed by the user
	readData    map[string]interface{} // Read data as managed by the user
	updateData  map[string]interface{} // Update data as managed by the user
	destroyData map[string]interface{} // Destroy data as managed by the user
	apiData     map[string]interface{} // Data as available from the API
	APIResponse string
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
	if opts.PostPath == "" {
		opts.PostPath = opts.Path
	}
	if opts.GetPath == "" {
		opts.GetPath = opts.Path + "/{id}"
	}
	if opts.PutPath == "" {
		opts.PutPath = opts.Path + "/{id}"
	}
	if opts.DestroyPath == "" {
		opts.DestroyPath = opts.Path + "/{id}"
	}
	if opts.SearchPath == "" {
		opts.SearchPath = opts.Path
	}

	obj := APIObject{
		apiClient:     iClient,
		getPath:       opts.GetPath,
		postPath:      opts.PostPath,
		putPath:       opts.PutPath,
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
		readData:      make(map[string]interface{}),
		updateData:    make(map[string]interface{}),
		destroyData:   make(map[string]interface{}),
		apiData:       make(map[string]interface{}),
	}

	if opts.Data != "" {
		tflog.Debug(ctx, "Parsing data", map[string]interface{}{"data": opts.Data})

		err := json.Unmarshal([]byte(opts.Data), &obj.data)
		if err != nil {
			return &obj, fmt.Errorf("error parsing data provided: %v", err.Error())
		}

		// Opportunistically set the object's ID if it is provided in the data.
		// If it is not set, we will get it later in synchronize_state
		if obj.ID == "" {
			var tmp string
			tmp, err := GetStringAtKey(obj.data, obj.IDAttribute, obj.debug)
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

	tflog.Debug(ctx, "Constructed object", map[string]interface{}{"object": obj.String()})

	return &obj, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *APIObject) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("id: %s\n", obj.ID))
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
	buffer.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(obj.data)))
	buffer.WriteString(fmt.Sprintf("read_data: %s\n", spew.Sdump(obj.readData)))
	buffer.WriteString(fmt.Sprintf("update_data: %s\n", spew.Sdump(obj.updateData)))
	buffer.WriteString(fmt.Sprintf("destroy_data: %s\n", spew.Sdump(obj.destroyData)))
	buffer.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(obj.apiData)))
	return buffer.String()
}

// updateState is a centralized function to ensure that our data as managed by
// the api_object is updated with data that has come back from the API
func (obj *APIObject) updateState(state string) error {
	ctx := context.Background()
	tflog.Debug(ctx, "Updating API object state to '%s'\n", map[string]interface{}{"state": state})

	err := json.Unmarshal([]byte(state), &obj.apiData)
	if err != nil {
		return err
	}

	obj.APIResponse = state

	// A usable ID was not passed (in constructor or here),
	// so we have to guess what it is from the data structure
	if obj.ID == "" {
		val, err := GetStringAtKey(obj.apiData, obj.IDAttribute, obj.debug)
		if err != nil {
			return fmt.Errorf("error extracting ID from data element: %s", err)
		}
		obj.ID = val
	} else {
		tflog.Debug(ctx, "Not updating id. It is already set to '%s'\n", map[string]interface{}{"id": obj.ID})
	}

	// Any keys that come from the data we want to copy are done here
	if len(obj.apiClient.copyKeys) > 0 {
		for _, key := range obj.apiClient.copyKeys {
			tflog.Debug(ctx, "Copying key from api_data to data\n", map[string]interface{}{"key": key, "new": obj.apiData[key], "old": obj.data[key]})
			obj.data[key] = obj.apiData[key]
		}
	} else {
		tflog.Debug(ctx, "copy_keys is empty - not attempting to copy data", nil)
	}

	tflog.Debug(ctx, "final object after synchronization of state", map[string]interface{}{"object": obj.String()})

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

	b, _ := json.Marshal(obj.data)

	postPath := obj.postPath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		postPath = fmt.Sprintf("%s?%s", obj.postPath, obj.queryString)
	}

	resultString, err := obj.apiClient.SendRequest(ctx, obj.createMethod, strings.Replace(postPath, "{id}", obj.ID, -1), string(b))
	if err != nil {
		return err
	}

	// We will need to sync state as well as get the object's ID
	if obj.apiClient.writeReturnsObject || obj.apiClient.createReturnsObject {
		tflog.Debug(ctx, "Parsing response from POST to update internal structures", map[string]interface{}{
			"write_returns_object":  obj.apiClient.writeReturnsObject,
			"create_returns_object": obj.apiClient.createReturnsObject,
		})
		err = obj.updateState(resultString)
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

	getPath := obj.getPath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		getPath = fmt.Sprintf("%s?%s", obj.getPath, obj.queryString)
	}

	send := ""
	if len(obj.readData) > 0 {
		readData, _ := json.Marshal(obj.readData)
		send = string(readData)
		tflog.Debug(ctx, "Using read data", map[string]interface{}{"read_data": send})
	}

	resultString, err := obj.apiClient.SendRequest(ctx, obj.readMethod, strings.Replace(getPath, "{id}", obj.ID, -1), send)
	if err != nil {
		if strings.Contains(err.Error(), "unexpected response code '404'") {
			tflog.Warn(ctx, "404 error while refreshing state. Removing from state.", map[string]interface{}{"id": obj.ID, "path": obj.getPath})
			obj.ID = ""
			return nil
		}
		return err
	}

	searchKey := obj.readSearch["search_key"]
	searchValue := obj.readSearch["search_value"]

	if searchKey != "" && searchValue != "" {
		obj.searchPath = strings.Replace(obj.getPath, "{id}", obj.ID, -1)

		queryString := obj.readSearch["query_string"]
		if obj.queryString != "" {
			tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
			queryString = fmt.Sprintf("%s&%s", obj.readSearch["query_string"], obj.queryString)
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
			tflog.Info(ctx, "Search did not find object", map[string]interface{}{"search_key": searchKey, "search_value": searchValue})
			obj.ID = ""
			return nil
		}
		objFoundString, _ := json.Marshal(objFound)
		return obj.updateState(string(objFoundString))
	}

	return obj.updateState(resultString)
}

func (obj *APIObject) UpdateObject(ctx context.Context) error {
	if obj.ID == "" {
		return fmt.Errorf("cannot update an object unless the ID has been set")
	}

	send := ""
	if len(obj.updateData) > 0 {
		updateData, _ := json.Marshal(obj.updateData)
		send = string(updateData)
		tflog.Debug(ctx, "Using update data", map[string]interface{}{"update_data": send})
	} else {
		b, _ := json.Marshal(obj.data)
		send = string(b)
	}

	putPath := obj.putPath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		putPath = fmt.Sprintf("%s?%s", obj.putPath, obj.queryString)
	}

	resultString, err := obj.apiClient.SendRequest(ctx, obj.updateMethod, strings.Replace(putPath, "{id}", obj.ID, -1), send)
	if err != nil {
		return err
	}

	if obj.apiClient.writeReturnsObject {
		tflog.Debug(ctx, "Parsing response from PUT to update internal structures", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.updateState(resultString)
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
	if len(obj.destroyData) > 0 {
		destroyData, _ := json.Marshal(obj.destroyData)
		send = string(destroyData)
		tflog.Debug(ctx, "Using destroy data", map[string]interface{}{"destroy_data": string(destroyData)})
	}

	_, err := obj.apiClient.SendRequest(ctx, obj.destroyMethod, strings.Replace(deletePath, "{id}", obj.ID, -1), send)
	if err != nil {
		return err
	}

	return nil
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
	resultString, err := obj.apiClient.SendRequest(ctx, obj.apiClient.readMethod, searchPath, searchData)
	if err != nil {
		return objFound, err
	}

	// Parse it seeking JSON data
	tflog.Debug(ctx, "Response received... parsing", nil)
	var result interface{}
	err = json.Unmarshal([]byte(resultString), &result)
	if err != nil {
		return objFound, err
	}

	if resultsKey != "" {
		var tmp interface{}

		tflog.Debug(ctx, "Locating results_key in the results", map[string]interface{}{"results_key": resultsKey})

		// First verify the data we got back is a map
		if _, ok = result.(map[string]interface{}); !ok {
			return objFound, fmt.Errorf("the results of a GET to '%s' did not return a map. Cannot search within for results_key '%s'", searchPath, resultsKey)
		}

		tmp, err = GetObjectAtKey(result.(map[string]interface{}), resultsKey, obj.debug)
		if err != nil {
			return objFound, fmt.Errorf("error finding results_key: %s", err)
		}
		if dataArray, ok = tmp.([]interface{}); !ok {
			return objFound, fmt.Errorf("the data at results_key location '%s' is not an array. It is a '%s'", resultsKey, reflect.TypeOf(tmp))
		}
	} else {
		tflog.Debug(ctx, "results_key is not set - coaxing data to array of interfaces", nil)
		if dataArray, ok = result.([]interface{}); !ok {
			return objFound, fmt.Errorf("the results of a GET to '%s' did not return an array. It is a '%s'. Perhaps you meant to add a results_key?", searchPath, reflect.TypeOf(result))
		}
	}

	// Loop through all of the results seeking the specific record
	for _, item := range dataArray {
		var hash map[string]interface{}

		if hash, ok = item.(map[string]interface{}); !ok {
			return objFound, fmt.Errorf("the elements being searched for data are not a map of key value pairs")
		}

		tflog.Debug(ctx, "Examining item in results array", map[string]interface{}{"item": hash})
		tflog.Debug(ctx, "Comparing search value to item value", map[string]interface{}{"search_value": searchValue, "search_key": searchKey})

		tmp, err := GetStringAtKey(hash, searchKey, obj.debug)
		if err != nil {
			return objFound, fmt.Errorf("failed to get the value of '%s' in the results array at '%s': %s", searchKey, resultsKey, err)
		}

		// We found our record
		if tmp == searchValue {
			objFound = hash
			obj.ID, err = GetStringAtKey(hash, obj.IDAttribute, obj.debug)
			if err != nil {
				return objFound, fmt.Errorf("failed to find id_attribute '%s' in the record: %s", obj.IDAttribute, err)
			}

			tflog.Debug(ctx, "Found ID '%s'", map[string]interface{}{"id": obj.ID})

			// But there is no id attribute???
			if obj.ID == "" {
				return objFound, fmt.Errorf("the object for '%s'='%s' did not have the id attribute '%s', or the value was empty", searchKey, searchValue, obj.IDAttribute)
			}
			break
		}
	}

	if obj.ID == "" {
		return objFound, fmt.Errorf("failed to find an object with the '%s' key = '%s' at %s", searchKey, searchValue, searchPath)
	}

	return objFound, nil
}
