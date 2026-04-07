package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestReadObject_ReadObjectKey tests the read_object_key extraction feature
func TestReadObject_ReadObjectKey(t *testing.T) {
	tests := []struct {
		name          string
		readObjectKey string
		apiResponse   map[string]interface{}
		expectedData  map[string]interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:          "simple_wrapper_extraction",
			readObjectKey: "result",
			apiResponse: map[string]interface{}{
				"result": map[string]interface{}{
					"id":   "test-123",
					"name": "test-object",
				},
			},
			expectedData: map[string]interface{}{
				"id":   "test-123",
				"name": "test-object",
			},
			expectError: false,
		},
		{
			name:          "nested_path_extraction",
			readObjectKey: "data/item",
			apiResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"item": map[string]interface{}{
						"id":    "nested-456",
						"value": "nested-data",
					},
				},
			},
			expectedData: map[string]interface{}{
				"id":    "nested-456",
				"value": "nested-data",
			},
			expectError: false,
		},
		{
			name:          "no_extraction_empty_key",
			readObjectKey: "",
			apiResponse: map[string]interface{}{
				"id":   "direct-789",
				"name": "direct-object",
			},
			expectedData: map[string]interface{}{
				"id":   "direct-789",
				"name": "direct-object",
			},
			expectError: false,
		},
		{
			name:          "error_key_not_found",
			readObjectKey: "missing_key",
			apiResponse: map[string]interface{}{
				"result": map[string]interface{}{
					"id": "test-123",
				},
			},
			expectedData:  nil,
			expectError:   true,
			errorContains: "failed to extract read_object_key 'missing_key'",
		},
		{
			name:          "deep_nested_path",
			readObjectKey: "response/data/items",
			apiResponse: map[string]interface{}{
				"response": map[string]interface{}{
					"data": map[string]interface{}{
						"items": map[string]interface{}{
							"id":   "deep-999",
							"type": "nested",
						},
					},
				},
			},
			expectedData: map[string]interface{}{
				"id":   "deep-999",
				"type": "nested",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.apiResponse)
			}))
			defer server.Close()

			// Create API client
			client, err := NewAPIClient(&APIClientOpt{
				URI:     server.URL,
				Timeout: 2,
			})
			if err != nil {
				t.Fatalf("Failed to create API client: %v", err)
			}

			// Create API object with read_object_key
			obj, err := NewAPIObject(client, &APIObjectOpts{
				Path:          "/api/objects",
				ReadObjectKey: tt.readObjectKey,
				ID:            "test-id",
				Data:          `{"id": "test-id"}`,
			})
			if err != nil {
				t.Fatalf("Failed to create API object: %v", err)
			}

			// Execute ReadObject
			err = obj.ReadObject(context.Background())

			// Verify error expectations
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error to contain '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			// Verify no error
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify extracted data matches expected
			actualData := obj.GetApiData()
			if tt.expectedData != nil {
				for k, expectedValue := range tt.expectedData {
					actualValue, exists := actualData[k]
					if !exists {
						t.Errorf("Expected key '%s' not found in api_data", k)
					} else if actualValue != expectedValue {
						t.Errorf("Key '%s': expected '%v', got '%v'", k, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

// TestReadObject_ReadObjectKeyWithXSSIPrefix tests that both xssi_prefix and read_object_key work together
func TestReadObject_ReadObjectKeyWithXSSIPrefix(t *testing.T) {
	// Create test server that returns XSSI-prefixed wrapped response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// XSSI prefix + wrapped response
		w.Write([]byte(")]}'\n{\"data\": {\"id\": \"xssi-123\", \"value\": \"protected\"}}"))
	}))
	defer server.Close()

	// Create API client with XSSI prefix configured
	client, err := NewAPIClient(&APIClientOpt{
		URI:        server.URL,
		Timeout:    2,
		XSSIPrefix: ")]}'\n",
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	// Create API object with read_object_key
	obj, err := NewAPIObject(client, &APIObjectOpts{
		Path:          "/api/objects",
		ReadObjectKey: "data", // Should extract after XSSI strip
		ID:            "xssi-123",
		Data:          `{"id": "xssi-123"}`,
	})
	if err != nil {
		t.Fatalf("Failed to create API object: %v", err)
	}

	// Execute ReadObject
	err = obj.ReadObject(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify extracted data (not wrapped, XSSI stripped)
	apiData := obj.GetApiData()
	if apiData["id"] != "xssi-123" {
		t.Errorf("Expected id 'xssi-123', got: %v", apiData["id"])
	}
	if apiData["value"] != "protected" {
		t.Errorf("Expected value 'protected', got: %v", apiData["value"])
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
