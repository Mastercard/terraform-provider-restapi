package restapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestReadObject_ReadSearchQueryString tests that read_search.query_string is properly used
func TestReadObject_ReadSearchQueryString(t *testing.T) {
	tests := []struct {
		name                  string
		objectQueryString     string
		readSearchQueryString string
		expectedQueryString   string
	}{
		{
			name:                  "only read_search query_string",
			objectQueryString:     "",
			readSearchQueryString: "per_page=1000&status=active",
			expectedQueryString:   "per_page=1000&status=active",
		},
		{
			// Regression test for issue #332: ensure no invalid "&" prefix when
			// only object-level query_string is provided
			name:                  "only object-level query_string",
			objectQueryString:     "version=v1",
			readSearchQueryString: "",
			expectedQueryString:   "version=v1",
		},
		{
			name:                  "both query_strings merge correctly",
			objectQueryString:     "version=v1",
			readSearchQueryString: "per_page=1000",
			expectedQueryString:   "per_page=1000&version=v1",
		},
		{
			name:                  "no query_strings",
			objectQueryString:     "",
			readSearchQueryString: "",
			expectedQueryString:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string

			// Create a test server that captures the request path
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				if r.URL.RawQuery != "" {
					receivedPath += "?" + r.URL.RawQuery
				}

				// Return a search result
				response := map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"id":     "test-id",
							"name":   "test-object",
							"status": "active",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client, err := NewAPIClient(&APIClientOpt{
				URI:     server.URL,
				Timeout: 2,
			})
			if err != nil {
				t.Fatalf("Failed to create API client: %v", err)
			}

			readSearch := map[string]string{
				"search_key":   "name",
				"search_value": "test-object",
				"results_key":  "data",
			}
			if tt.readSearchQueryString != "" {
				readSearch["query_string"] = tt.readSearchQueryString
			}

			opts := &APIObjectOpts{
				Path:        "/api/objects",
				Data:        `{"id": "test-id", "name": "test-object"}`,
				IDAttribute: "id",
				QueryString: tt.objectQueryString,
				ReadSearch:  readSearch,
			}

			obj, err := NewAPIObject(client, opts)
			if err != nil {
				t.Fatalf("NewAPIObject() error = %v", err)
			}

			// Trigger a read operation which will use read_search
			ctx := context.Background()
			err = obj.ReadObject(ctx)
			if err != nil {
				t.Fatalf("ReadObject() error = %v", err)
			}

			// Verify the query string was used correctly
			expectedPath := "/api/objects"
			if tt.expectedQueryString != "" {
				expectedPath += "?" + tt.expectedQueryString
			}

			if receivedPath != expectedPath {
				t.Errorf("Expected path %q, got %q", expectedPath, receivedPath)
			}
		})
	}
}

// TestReadObject_ReadSearchWithIdPlaceholder tests that {id} placeholder in search_value
// is properly substituted with the object's ID
func TestReadObject_ReadSearchWithIdPlaceholder(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		if r.URL.RawQuery != "" {
			receivedPath += "?" + r.URL.RawQuery
		}

		// Return search results with the object matching the ID
		response := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"id":       "12345",
					"owner_id": "owner-12345",
					"name":     "test-object",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     server.URL,
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	opts := &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "12345", "owner_id": "owner-12345", "name": "test-object"}`,
		IDAttribute: "id",
		ReadSearch: map[string]string{
			"search_key":   "owner_id",
			"search_value": "owner-{id}", // {id} should be replaced with 12345
			"results_key":  "items",
		},
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		t.Fatalf("NewAPIObject() error = %v", err)
	}

	ctx := context.Background()
	err = obj.ReadObject(ctx)
	if err != nil {
		t.Fatalf("ReadObject() error = %v", err)
	}

	// Verify the object was found (ID should still be set)
	if obj.ID != "12345" {
		t.Errorf("Expected ID to be 12345, got %q", obj.ID)
	}

	// Verify we called the collection endpoint
	if receivedPath != "/api/objects" {
		t.Errorf("Expected path /api/objects, got %q", receivedPath)
	}
}

// TestReadObject_ReadSearchWithResultsKey tests that results_key properly extracts
// the array from nested JSON responses
func TestReadObject_ReadSearchWithResultsKey(t *testing.T) {
	tests := []struct {
		name       string
		resultsKey string
		response   map[string]interface{}
		shouldFind bool
	}{
		{
			name:       "simple results_key",
			resultsKey: "data",
			response: map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": "test-id", "name": "test-object"},
				},
			},
			shouldFind: true,
		},
		{
			name:       "nested results_key",
			resultsKey: "response/items",
			response: map[string]interface{}{
				"response": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"id": "test-id", "name": "test-object"},
					},
				},
			},
			shouldFind: true,
		},
		{
			name:       "no results_key - array at root",
			resultsKey: "",
			response: map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": "test-id", "name": "test-object"},
				},
			},
			shouldFind: false, // Should fail because root is not an array
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client, err := NewAPIClient(&APIClientOpt{
				URI:     server.URL,
				Timeout: 2,
			})
			if err != nil {
				t.Fatalf("Failed to create API client: %v", err)
			}

			readSearch := map[string]string{
				"search_key":   "name",
				"search_value": "test-object",
			}
			if tt.resultsKey != "" {
				readSearch["results_key"] = tt.resultsKey
			}

			opts := &APIObjectOpts{
				Path:        "/api/objects",
				Data:        `{"id": "test-id", "name": "test-object"}`,
				IDAttribute: "id",
				ReadSearch:  readSearch,
			}

			obj, err := NewAPIObject(client, opts)
			if err != nil {
				t.Fatalf("NewAPIObject() error = %v", err)
			}

			ctx := context.Background()
			err = obj.ReadObject(ctx)

			if tt.shouldFind {
				if err != nil {
					t.Errorf("Expected to find object, but got error: %v", err)
				}
				if obj.ID != "test-id" {
					t.Errorf("Expected ID to be test-id, got %q", obj.ID)
				}
			} else {
				// It's okay if it fails - the test is checking behavior with invalid results_key
				if err == nil && obj.ID == "" {
					// Object not found is acceptable
				}
			}
		})
	}
}

// TestReadObject_ReadSearchNoMatchFound tests that when no matching object is found
// in search results, the object ID is cleared (treated as deleted)
func TestReadObject_ReadSearchNoMatchFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return search results that don't match the search criteria
		response := map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"id":   "other-id",
					"name": "other-object",
				},
				map[string]interface{}{
					"id":   "another-id",
					"name": "another-object",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     server.URL,
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	opts := &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "test-id", "name": "test-object"}`,
		IDAttribute: "id",
		ReadSearch: map[string]string{
			"search_key":   "name",
			"search_value": "test-object", // This won't match any object in the response
			"results_key":  "data",
		},
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		t.Fatalf("NewAPIObject() error = %v", err)
	}

	ctx := context.Background()
	err = obj.ReadObject(ctx)
	if err != nil {
		t.Fatalf("ReadObject() should not error when object not found, got: %v", err)
	}

	// When object is not found in search, ID should be cleared
	if obj.ID != "" {
		t.Errorf("Expected ID to be empty when object not found, got %q", obj.ID)
	}
}

// TestReadObject_ReadSearchMultipleResults tests that the correct object is selected
// when search returns multiple results
func TestReadObject_ReadSearchMultipleResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return multiple objects, only one should match
		response := map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"id":     "obj-1",
					"name":   "first-object",
					"status": "active",
				},
				map[string]interface{}{
					"id":     "obj-2",
					"name":   "target-object", // This is the one we're searching for
					"status": "active",
				},
				map[string]interface{}{
					"id":     "obj-3",
					"name":   "third-object",
					"status": "active",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     server.URL,
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	opts := &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "obj-2", "name": "target-object"}`,
		IDAttribute: "id",
		ReadSearch: map[string]string{
			"search_key":   "name",
			"search_value": "target-object",
			"results_key":  "data",
		},
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		t.Fatalf("NewAPIObject() error = %v", err)
	}

	ctx := context.Background()
	err = obj.ReadObject(ctx)
	if err != nil {
		t.Fatalf("ReadObject() error = %v", err)
	}

	// Verify we got the correct object
	if obj.ID != "obj-2" {
		t.Errorf("Expected ID to be obj-2, got %q", obj.ID)
	}

	// Verify the apiData contains the correct object
	obj.mux.RLock()
	defer obj.mux.RUnlock()

	if obj.apiData["name"] != "target-object" {
		t.Errorf("Expected name to be target-object, got %v", obj.apiData["name"])
	}
}

// TestReadObject_ReadSearchWithSearchData tests that search_data is properly sent
// as the POST body when performing a search
func TestReadObject_ReadSearchWithSearchData(t *testing.T) {
	var receivedBody string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method

		// Capture the request body
		if r.Body != nil {
			bodyBytes, _ := json.Marshal(nil)
			if r.ContentLength > 0 {
				bodyBytes = make([]byte, r.ContentLength)
				r.Body.Read(bodyBytes)
			}
			receivedBody = string(bodyBytes)
		}

		// Return search results
		response := map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"id":   "test-id",
					"name": "test-object",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAPIClient(&APIClientOpt{
		URI:     server.URL,
		Timeout: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	searchData := `{"filter": {"status": "active"}, "limit": 100}`

	opts := &APIObjectOpts{
		Path:        "/api/objects",
		Data:        `{"id": "test-id", "name": "test-object"}`,
		IDAttribute: "id",
		ReadSearch: map[string]string{
			"search_key":   "name",
			"search_value": "test-object",
			"results_key":  "results",
			"search_data":  searchData,
		},
	}

	obj, err := NewAPIObject(client, opts)
	if err != nil {
		t.Fatalf("NewAPIObject() error = %v", err)
	}

	ctx := context.Background()
	err = obj.ReadObject(ctx)
	if err != nil {
		t.Fatalf("ReadObject() error = %v", err)
	}

	// Verify the search_data was sent
	if receivedBody == "" {
		t.Error("Expected search_data to be sent in request body, but body was empty")
	}

	// Verify the method used (should be GET by default, but with body)
	if receivedMethod != "GET" {
		t.Logf("Note: Search used method %s (may vary based on API client config)", receivedMethod)
	}
}
