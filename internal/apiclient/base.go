package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// APIBase contains fields and methods shared between APIObject and APISetting.
type APIBase struct {
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

	mux         sync.RWMutex
	data        map[string]interface{}
	readData    map[string]interface{}
	updateData  map[string]interface{}
	apiData     map[string]interface{}
	apiResponse string
}

// buildPath appends the configured query string to basePath.
func (b *APIBase) buildPath(basePath string) string {
	if b.queryString != "" {
		return fmt.Sprintf("%s?%s", basePath, b.queryString)
	}
	return basePath
}

// baseUpdateInternalState unmarshals the API response into apiData, stores the raw
// response, and copies any configured copy_keys back into data.
// This is the shared implementation used by both APIObject and APISetting.
func (b *APIBase) baseUpdateInternalState(state string) error {
	ctx := context.Background()
	tflog.Debug(ctx, "Updating internal state", map[string]interface{}{"state": state})

	b.mux.Lock()
	defer b.mux.Unlock()

	err := json.Unmarshal([]byte(state), &b.apiData)
	if err != nil {
		return err
	}

	b.apiResponse = state

	if len(b.apiClient.copyKeys) > 0 {
		for _, key := range b.apiClient.copyKeys {
			tflog.Debug(ctx, "Copying key from api_data to data", map[string]interface{}{"key": key, "new": b.apiData[key], "old": b.data[key]})
			b.data[key] = b.apiData[key]
		}
	} else {
		tflog.Debug(ctx, "copy_keys is empty - not attempting to copy data", nil)
	}

	return nil
}

// sendRead sends a read request to path and returns the response body.
func (b *APIBase) sendRead(ctx context.Context, path string) (string, error) {
	send := ""
	if b.readData != nil {
		readData, _ := json.Marshal(b.readData)
		send = string(readData)
		tflog.Debug(ctx, "Using read data", map[string]interface{}{"read_data": send})
	}
	resultString, _, err := b.apiClient.SendRequest(ctx, b.readMethod, path, send, b.debug, b.headers)
	return resultString, err
}

// sendUpdate sends an update request to path. Uses update_data if configured,
// otherwise sends the full managed data payload.
func (b *APIBase) sendUpdate(ctx context.Context, path string) (string, error) {
	send := ""
	b.mux.RLock()
	if b.updateData != nil {
		updateData, _ := json.Marshal(b.updateData)
		send = string(updateData)
		tflog.Debug(ctx, "Using update data", map[string]interface{}{"update_data": send})
	} else {
		d, _ := json.Marshal(b.data)
		send = string(d)
	}
	b.mux.RUnlock()
	resultString, _, err := b.apiClient.SendRequest(ctx, b.updateMethod, path, send, b.debug, b.headers)
	return resultString, err
}

// GetApiData returns a copy of the api_data map.
func (b *APIBase) GetApiData() map[string]string {
	b.mux.RLock()
	defer b.mux.RUnlock()

	apiData := make(map[string]string)
	for k, v := range b.apiData {
		apiData[k] = fmt.Sprintf("%v", v)
	}
	return apiData
}

// GetApiResponse returns the raw API response string from the most recent read.
func (b *APIBase) GetApiResponse() string {
	return b.apiResponse
}

// baseString returns a string representation of the shared fields.
// The caller is responsible for any locking needed for additional fields.
func (b *APIBase) baseString() string {
	b.mux.RLock()
	defer b.mux.RUnlock()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("id: %s\n", b.ID))
	buf.WriteString(fmt.Sprintf("get_path: %s\n", b.readPath))
	buf.WriteString(fmt.Sprintf("put_path: %s\n", b.updatePath))
	buf.WriteString(fmt.Sprintf("query_string: %s\n", b.queryString))
	buf.WriteString(fmt.Sprintf("read_method: %s\n", b.readMethod))
	buf.WriteString(fmt.Sprintf("update_method: %s\n", b.updateMethod))
	buf.WriteString(fmt.Sprintf("debug: %t\n", b.debug))
	buf.WriteString(fmt.Sprintf("data: %s\n", spew.Sdump(b.data)))
	buf.WriteString(fmt.Sprintf("read_data: %s\n", spew.Sdump(b.readData)))
	buf.WriteString(fmt.Sprintf("update_data: %s\n", spew.Sdump(b.updateData)))
	buf.WriteString(fmt.Sprintf("api_data: %s\n", spew.Sdump(b.apiData)))
	return buf.String()
}
