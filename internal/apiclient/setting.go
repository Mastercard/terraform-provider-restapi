package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type APISettingOpts struct {
	ReadMethod   string
	ReadPath     string
	ReadData     string
	UpdateMethod string
	UpdatePath   string
	UpdateData   string
	InitialState string
	QueryString  string
	Debug        bool
	ID           string
	IDAttribute  string
	Data         string
	Headers      map[string]string
}

// APISetting is the state holding struct for a restapi_object resource
type APISetting struct {
	apiClient    *APIClient
	readMethod   string
	readPath     string
	updateMethod string
	updatePath   string
	queryString  string
	debug        bool
	ID           string
	IDAttribute  string
	headers      map[string]string

	// Set internally
	mux          sync.RWMutex           // Protects data and apiData fields
	data         map[string]interface{} // Data as managed by the user
	readData     map[string]interface{} // Data to send during Read operation
	updateData   map[string]interface{} // Data to send during Update operation
	apiData      map[string]interface{} // Data from the most recent read operation of the API object, as massaged to a map
	apiResponse  string                 // Raw API response from most recent read operation
	initialState map[string]interface{} // The initial state of the object as read from the API, used for diffing during updates
}

// NewAPISetting makes an APISetting to manage a RESTful object in an API
func NewAPISetting(iClient *APIClient, opts *APISettingOpts) (*APISetting, error) {
	ctx := context.Background()
	tflog.Debug(ctx, "Constructing api_setting", map[string]interface{}{"id": opts.ID})

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

	obj := APISetting{
		apiClient:    iClient,
		readPath:     opts.ReadPath,
		updatePath:   opts.UpdatePath,
		readMethod:   opts.ReadMethod,
		updateMethod: opts.UpdateMethod,
		queryString:  opts.QueryString,
		debug:        opts.Debug,
		ID:           opts.ID,
		IDAttribute:  opts.IDAttribute,
		headers:      opts.Headers,
		data:         make(map[string]interface{}),
		readData:     nil,
		updateData:   nil,
		apiData:      make(map[string]interface{}),
		initialState: nil,
	}

	if opts.Data != "" {
		tflog.Debug(ctx, "Parsing data", map[string]interface{}{"data": opts.Data})

		err := json.Unmarshal([]byte(opts.Data), &obj.data)
		if err != nil {
			return &obj, fmt.Errorf("error parsing data provided: %v", err.Error())
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

	if opts.InitialState != "" {
		tflog.Debug(ctx, "Parsing initial state", map[string]interface{}{"initialState": opts.InitialState})

		err := json.Unmarshal([]byte(opts.InitialState), &obj.initialState)
		if err != nil {
			return &obj, fmt.Errorf("error parsing initial state provided: %v", err.Error())
		}
	}

	tflog.Debug(ctx, "Constructed object", map[string]interface{}{"object": obj.String()})

	return &obj, nil
}

// Convert the important bits about this object to string representation
// This is useful for debugging.
func (obj *APISetting) String() string {
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("id: %s\n", obj.ID))
	buffer.WriteString(fmt.Sprintf("get_path: %s\n", obj.readPath))
	buffer.WriteString(fmt.Sprintf("put_path: %s\n", obj.updatePath))
	buffer.WriteString(fmt.Sprintf("query_string: %s\n", obj.queryString))
	buffer.WriteString(fmt.Sprintf("read_method: %s\n", obj.readMethod))
	buffer.WriteString(fmt.Sprintf("update_method: %s\n", obj.updateMethod))
	buffer.WriteString(fmt.Sprintf("debug: %t\n", obj.debug))
	buffer.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(obj.data)))
	buffer.WriteString(fmt.Sprintf("read_data: %s\n", spew.Sdump(obj.readData)))
	buffer.WriteString(fmt.Sprintf("update_data: %s\n", spew.Sdump(obj.updateData)))
	buffer.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(obj.apiData)))
	return buffer.String()
}

// SetDataFromMap sets the object's internal state from a map
// This allows more fine-grained manipulation of an object's data, outside of reads to the API
func (obj *APISetting) SetDataFromMap(d map[string]interface{}) error {
	foundDataJSON, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal found data: %w", err)
	}
	return obj.updateInternalState(string(foundDataJSON))
}

// updateInternalState is a centralized function to ensure that our data as managed by
// the api_object is updated with data that has come back from the API
func (obj *APISetting) updateInternalState(state string) error {
	ctx := context.Background()
	tflog.Debug(ctx, "Updating API object state to '%s'\n", map[string]interface{}{"state": state})

	obj.mux.Lock()
	defer obj.mux.Unlock()

	err := json.Unmarshal([]byte(state), &obj.apiData)
	if err != nil {
		return err
	}

	obj.apiResponse = state

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

func (obj *APISetting) CreateSetting(ctx context.Context) error {
	// Read the object for initial validation
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

	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.readMethod, getPath, send, obj.debug, obj.headers)
	if err != nil {
		return err
	}

	obj.mux.Lock()
	err = json.Unmarshal([]byte(resultString), &obj.initialState)
	obj.mux.Unlock()

	obj.SetDataFromMap(obj.initialState)
	err = obj.UpdateSetting(ctx)
	return err
}

func (obj *APISetting) ReadSetting(ctx context.Context) error {
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

	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.readMethod, getPath, send, obj.debug, obj.headers)
	if err != nil {
		return err
	}

	return obj.updateInternalState(resultString)
}

func (obj *APISetting) UpdateSetting(ctx context.Context) error {
	send := ""

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

	resultString, _, err := obj.apiClient.SendRequest(ctx, obj.updateMethod, putPath, send, obj.debug, obj.headers)
	if err != nil {
		return err
	}

	if obj.apiClient.writeReturnsObject {
		tflog.Debug(ctx, "Parsing response from PUT to update internal structures", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.updateInternalState(resultString)
	} else {
		tflog.Debug(ctx, "Requesting updated object from API", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.ReadSetting(ctx)
	}
	return err
}

func (obj *APISetting) DeleteSetting(ctx context.Context) error {
	deletePath := obj.updatePath
	if obj.queryString != "" {
		tflog.Debug(ctx, "Adding query string", map[string]interface{}{"query_string": obj.queryString})
		deletePath = fmt.Sprintf("%s?%s", obj.updatePath, obj.queryString)
	}

	send, _ := json.Marshal(obj.getRestorePayload())

	_, code, err := obj.apiClient.SendRequest(ctx, obj.updateMethod, deletePath, string(send), obj.debug, obj.headers)
	if err != nil {
		// 404 (Not Found) or 410 (Gone) during delete is acceptable -
		// the object is already gone, which is the desired end state.
		if code == http.StatusNotFound || code == http.StatusGone {
			tflog.Warn(ctx, "404/410 error while restoring setting to initial state. Assuming already updated.", map[string]interface{}{"id": obj.ID, "path": obj.updatePath})
			err = nil
		}
	}

	return err
}

// getRestorePayload limits restore to keys managed in data; arrays are restored as full values.
func (obj *APISetting) getRestorePayload() map[string]interface{} {
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	if obj.initialState == nil {
		return map[string]interface{}{}
	}

	return filterInitialStateByData(obj.initialState, obj.data)
}

func filterInitialStateByData(initial map[string]interface{}, data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if initial == nil || data == nil {
		return result
	}

	for key, dataValue := range data {
		initialValue, ok := initial[key]
		if !ok {
			continue
		}

		if nestedData, ok := dataValue.(map[string]interface{}); ok {
			if nestedInitial, ok := initialValue.(map[string]interface{}); ok {
				result[key] = filterInitialStateByData(nestedInitial, nestedData)
				continue
			}
		}

		result[key] = initialValue
	}

	return result
}

// GetApiData returns a copy of the api_data map from the APISetting
func (obj *APISetting) GetApiData() map[string]string {
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	apiData := make(map[string]string)
	for k, v := range obj.apiData {
		apiData[k] = fmt.Sprintf("%v", v)
	}
	return apiData
}

// GetApiResponse returns a copy of the raw API response from the APISetting
func (obj *APISetting) GetApiResponse() string {
	return obj.apiResponse
}

// GetInitialStateResponse returns a JSON string of the captured initial state.
func (obj *APISetting) GetInitialStateResponse() string {
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	if obj.initialState == nil {
		return ""
	}

	b, err := json.Marshal(obj.initialState)
	if err != nil {
		return ""
	}

	return string(b)
}
