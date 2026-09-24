package apiclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateObject_WriteObjectKey tests that write_object_key wraps POST request bodies
func TestCreateObject_WriteObjectKey(t *testing.T) {
	tests := []struct {
		name           string
		writeObjectKey string
		data           string
		expectedBody   map[string]interface{}
	}{
		{
			name:           "simple_wrapping",
			writeObjectKey: "entry",
			data:           `{"id": "test-1", "name": "foo"}`,
			expectedBody: map[string]interface{}{
				"entry": map[string]interface{}{
					"id":   "test-1",
					"name": "foo",
				},
			},
		},
		{
			name:           "nested_wrapping",
			writeObjectKey: "request/data",
			data:           `{"id": "test-2", "value": "bar"}`,
			expectedBody: map[string]interface{}{
				"request": map[string]interface{}{
					"data": map[string]interface{}{
						"id":    "test-2",
						"value": "bar",
					},
				},
			},
		},
		{
			name:           "no_wrapping_empty_key",
			writeObjectKey: "",
			data:           `{"id": "test-3", "name": "baz"}`,
			expectedBody: map[string]interface{}{
				"id":   "test-3",
				"name": "baz",
			},
		},
		{
			name:           "deep_nested_wrapping",
			writeObjectKey: "api/v2/payload",
			data:           `{"id": "test-4", "config": {"enabled": true}}`,
			expectedBody: map[string]interface{}{
				"api": map[string]interface{}{
					"v2": map[string]interface{}{
						"payload": map[string]interface{}{
							"id": "test-4",
							"config": map[string]interface{}{
								"enabled": true,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedBody map[string]interface{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					body, _ := io.ReadAll(r.Body)
					json.Unmarshal(body, &receivedBody)
				}
				// Return the unwrapped object (as if API strips the wrapper)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":   "test-1",
					"name": "foo",
				})
			}))
			defer server.Close()

			client, err := NewAPIClient(&APIClientOpt{
				URI:                server.URL,
				Timeout:            2,
				WriteReturnsObject: true,
			})
			if err != nil {
				t.Fatalf("Failed to create API client: %v", err)
			}

			obj, err := NewAPIObject(client, &APIObjectOpts{
				Path:           "/api/objects",
				WriteObjectKey: tt.writeObjectKey,
				Data:           tt.data,
			})
			if err != nil {
				t.Fatalf("Failed to create API object: %v", err)
			}

			err = obj.CreateObject(context.Background())
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the request body was wrapped correctly
			expectedJSON, _ := json.Marshal(tt.expectedBody)
			receivedJSON, _ := json.Marshal(receivedBody)

			var expectedNorm, receivedNorm interface{}
			json.Unmarshal(expectedJSON, &expectedNorm)
			json.Unmarshal(receivedJSON, &receivedNorm)

			expectedStr, _ := json.Marshal(expectedNorm)
			receivedStr, _ := json.Marshal(receivedNorm)

			if string(expectedStr) != string(receivedStr) {
				t.Errorf("Request body mismatch.\nExpected: %s\nReceived: %s", string(expectedStr), string(receivedStr))
			}
		})
	}
}

// TestUpdateObject_WriteObjectKey tests that write_object_key wraps PUT request bodies
func TestUpdateObject_WriteObjectKey(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "update-1",
			"name": "updated",
		})
	}))
	defer server.Close()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                server.URL,
		Timeout:            2,
		WriteReturnsObject: true,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:           "/api/objects",
		WriteObjectKey: "entry",
		ID:             "update-1",
		Data:           `{"id": "update-1", "name": "updated"}`,
	})
	if err != nil {
		t.Fatalf("Failed to create API object: %v", err)
	}

	err = obj.UpdateObject(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the PUT body was wrapped under "entry"
	entry, ok := receivedBody["entry"]
	if !ok {
		t.Fatalf("Expected request body to have 'entry' key, got: %v", receivedBody)
	}
	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected 'entry' to be a map, got: %T", entry)
	}
	if entryMap["id"] != "update-1" {
		t.Errorf("Expected id 'update-1', got: %v", entryMap["id"])
	}
	if entryMap["name"] != "updated" {
		t.Errorf("Expected name 'updated', got: %v", entryMap["name"])
	}
}

// TestWriteObjectKey_WithReadObjectKey tests that both directions work together:
// write_object_key wraps POST/PUT bodies, read_object_key unwraps GET responses
func TestWriteObjectKey_WithReadObjectKey(t *testing.T) {
	var createBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "POST":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &createBody)
			// API responds with wrapped response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   "combo-1",
				"name": "combo-test",
			})
		case "GET":
			// API returns wrapped GET response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": map[string]interface{}{
					"id":   "combo-1",
					"name": "combo-test",
				},
				"success": true,
			})
		}
	}))
	defer server.Close()

	client, err := NewAPIClient(&APIClientOpt{
		URI:                 server.URL,
		Timeout:             2,
		CreateReturnsObject: true,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:           "/api/objects",
		WriteObjectKey: "entry",
		ReadObjectKey:  "result",
		Data:           `{"id": "combo-1", "name": "combo-test"}`,
	})
	if err != nil {
		t.Fatalf("Failed to create API object: %v", err)
	}

	// Test CREATE: body should be wrapped
	err = obj.CreateObject(context.Background())
	if err != nil {
		t.Fatalf("Unexpected create error: %v", err)
	}

	entry, ok := createBody["entry"]
	if !ok {
		t.Fatalf("POST body should have 'entry' wrapper, got: %v", createBody)
	}
	entryMap := entry.(map[string]interface{})
	if entryMap["name"] != "combo-test" {
		t.Errorf("POST body entry.name should be 'combo-test', got: %v", entryMap["name"])
	}

	// Test READ: response should be unwrapped
	err = obj.ReadObject(context.Background())
	if err != nil {
		t.Fatalf("Unexpected read error: %v", err)
	}

	apiData := obj.GetApiData()
	if apiData["id"] != "combo-1" {
		t.Errorf("Expected id 'combo-1' after unwrapping, got: %v", apiData["id"])
	}
	if apiData["name"] != "combo-test" {
		t.Errorf("Expected name 'combo-test' after unwrapping, got: %v", apiData["name"])
	}
	// Should NOT have 'success' or 'result' keys (they were stripped by read_object_key)
	if _, exists := apiData["success"]; exists {
		t.Error("Should not have 'success' key after read_object_key extraction")
	}
}

// TestWriteObjectKey_CascadeFromProvider tests that write_object_key falls back to client-level config
func TestWriteObjectKey_CascadeFromProvider(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "cascade-1", "name": "test"})
	}))
	defer server.Close()

	// Set write_object_key on the CLIENT (simulating provider-level config)
	client, err := NewAPIClient(&APIClientOpt{
		URI:                server.URL,
		Timeout:            2,
		WriteReturnsObject: true,
		WriteObjectKey:     "wrapper",
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Object does NOT set write_object_key — should inherit from client
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		Data: `{"id": "cascade-1", "name": "test"}`,
	})
	if err != nil {
		t.Fatalf("Failed to create API object: %v", err)
	}

	err = obj.CreateObject(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, ok := receivedBody["wrapper"]; !ok {
		t.Fatalf("Expected client-level write_object_key 'wrapper' to be applied, got: %v", receivedBody)
	}
}

// TestWriteObjectKey_ResourceOverridesProvider tests that resource-level write_object_key overrides provider
func TestWriteObjectKey_ResourceOverridesProvider(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "override-1", "name": "test"})
	}))
	defer server.Close()

	// Provider sets "provider_wrapper"
	client, err := NewAPIClient(&APIClientOpt{
		URI:                server.URL,
		Timeout:            2,
		WriteReturnsObject: true,
		WriteObjectKey:     "provider_wrapper",
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Resource overrides with "resource_wrapper"
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:           "/api/objects",
		WriteObjectKey: "resource_wrapper",
		Data:           `{"id": "override-1", "name": "test"}`,
	})
	if err != nil {
		t.Fatalf("Failed to create API object: %v", err)
	}

	err = obj.CreateObject(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, ok := receivedBody["resource_wrapper"]; !ok {
		t.Fatalf("Expected resource-level 'resource_wrapper' to override provider, got: %v", receivedBody)
	}
	if _, ok := receivedBody["provider_wrapper"]; ok {
		t.Fatal("Provider-level 'provider_wrapper' should NOT be present when resource overrides")
	}
}
