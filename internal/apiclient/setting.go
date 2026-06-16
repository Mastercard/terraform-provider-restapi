package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// APISetting is the state holding struct for a restapi_setting resource
type APISetting struct {
	APIBase

	// Set internally
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
		APIBase: APIBase{
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
		},
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
	return obj.baseString()
}

// SetDataFromMap sets the object's internal state from a map
// This allows more fine-grained manipulation of an object's data, outside of reads to the API
func (obj *APISetting) SetDataFromMap(d map[string]interface{}) error {
	foundDataJSON, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal found data: %w", err)
	}
	return obj.baseUpdateInternalState(string(foundDataJSON))
}

func (obj *APISetting) CreateSetting(ctx context.Context) error {
	// Read the object for initial validation
	getPath := obj.buildPath(obj.readPath)
	resultString, err := obj.sendRead(ctx, getPath)
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
	getPath := obj.buildPath(obj.readPath)
	resultString, err := obj.sendRead(ctx, getPath)
	if err != nil {
		return err
	}
	return obj.baseUpdateInternalState(resultString)
}

func (obj *APISetting) UpdateSetting(ctx context.Context) error {
	putPath := obj.buildPath(obj.updatePath)
	resultString, err := obj.sendUpdate(ctx, putPath)
	if err != nil {
		return err
	}

	if obj.apiClient.writeReturnsObject {
		tflog.Debug(ctx, "Parsing response from PUT to update internal structures", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.baseUpdateInternalState(resultString)
	} else {
		tflog.Debug(ctx, "Requesting updated object from API", map[string]interface{}{"write_returns_object": obj.apiClient.writeReturnsObject})
		err = obj.ReadSetting(ctx)
	}
	return err
}

func (obj *APISetting) DeleteSetting(ctx context.Context) error {
	deletePath := obj.buildPath(obj.updatePath)

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
