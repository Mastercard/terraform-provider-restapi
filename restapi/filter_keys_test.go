package restapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Custom test server that returns the content from service_from_midpoint.json for GET
// and captures/validates PATCH requests
type filterKeysFakeServer struct {
	server      *http.Server
	objects     map[string]map[string]interface{}
	debug       bool
	running     bool
	lastRequest *http.Request
	lastBody    []byte
	patchHistory []map[string]interface{}
}

func NewFilterKeysFakeServer(port int, objects map[string]map[string]interface{}, debug bool) *filterKeysFakeServer {
	serverMux := http.NewServeMux()

	svr := &filterKeysFakeServer{
		debug:        debug,
		objects:      objects,
		running:      false,
		patchHistory: make([]map[string]interface{}, 0),
	}

	serverMux.HandleFunc("/api/", svr.handleAPIObject)

	apiObjectServer := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: serverMux,
	}

	svr.server = apiObjectServer

	if debug {
		log.Printf("filter_keys_test.go: Starting fake server on port %d", port)
	}

	go apiObjectServer.ListenAndServe()
	/* Sleep a bit to let the server start up */
	time.Sleep(100 * time.Millisecond)
	svr.running = true

	return svr
}

func (svr *filterKeysFakeServer) Shutdown() {
	if svr.running {
		svr.server.Close()
		svr.running = false
		// Give some time for the server to shut down
		time.Sleep(100 * time.Millisecond)
	}
}

func (svr *filterKeysFakeServer) handleAPIObject(w http.ResponseWriter, r *http.Request) {
	var obj map[string]interface{}
	var id string
	var ok bool

	// Save the last request for test verification
	svr.lastRequest = r

	// Read the request body
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("filter_keys_test.go: ERROR - Failed to read request body: %s\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	svr.lastBody = b

	if svr.debug {
		log.Printf("filter_keys_test.go: Received %s request to %s\n", r.Method, r.URL.Path)
		log.Printf("filter_keys_test.go: Request body: %s\n", string(b))
	}

	path := r.URL.EscapedPath()
	parts := strings.Split(path, "/")

	if len(parts) == 4 {
		id = parts[3]
		obj, ok = svr.objects[id]

		if !ok && r.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
	} else if path != "/api/objects" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if r.Method == "DELETE" {
		delete(svr.objects, id)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == "PATCH" && len(b) > 0 {
		// Handle Midpoint PATCH request with ObjectModificationType
		var patchRequest map[string]interface{}
		err := json.Unmarshal(b, &patchRequest)
		if err != nil {
			log.Printf("filter_keys_test.go: Failed to unmarshal PATCH data: %s\n", err)
			http.Error(w, "Invalid JSON in patch request", http.StatusBadRequest)
			return
		}

		// Save the patch request for later verification
		svr.patchHistory = append(svr.patchHistory, patchRequest)

		// Navigate to objectModification.itemDelta in the request
		objectMod, ok := patchRequest["objectModification"].(map[string]interface{})
		if !ok {
			http.Error(w, "Missing objectModification in request", http.StatusBadRequest)
			return
		}

		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			http.Error(w, "Missing itemDelta in request", http.StatusBadRequest)
			return
		}

		// Process the patch according to modificationType
		modType, ok := itemDelta["modificationType"].(string)
		if !ok {
			http.Error(w, "Missing modificationType in request", http.StatusBadRequest)
			return
		}

		path, ok := itemDelta["path"].(string)
		if !ok {
			http.Error(w, "Missing path in request", http.StatusBadRequest)
			return
		}

		// Handle nested paths in the patch request
		if strings.Contains(path, ".") {
			parts := strings.Split(path, ".")
			currentObj := obj
			
			// Navigate to the parent object where the change needs to be made
			for i := 0; i < len(parts)-1; i++ {
				part := parts[i]
				
				// If the path component doesn't exist and we're adding, create it
				if nestedObj, exists := currentObj[part]; !exists {
					if modType == "add" {
						currentObj[part] = make(map[string]interface{})
					} else {
						http.Error(w, fmt.Sprintf("Path component '%s' not found", part), http.StatusBadRequest)
						return
					}
					currentObj = currentObj[part].(map[string]interface{})
				} else {
					if nestedMap, ok := nestedObj.(map[string]interface{}); ok {
						currentObj = nestedMap
					} else {
						http.Error(w, fmt.Sprintf("Path component '%s' is not an object", part), http.StatusBadRequest)
						return
					}
				}
			}
			
			// Get the final key name
			finalKey := parts[len(parts)-1]
			
			// Perform the operation on the final key
			switch modType {
			case "add":
				value, exists := itemDelta["value"]
				if !exists {
					http.Error(w, "Missing value for add operation", http.StatusBadRequest)
					return
				}
				currentObj[finalKey] = value
				
			case "delete":
				delete(currentObj, finalKey)
				
			case "replace":
				value, exists := itemDelta["value"]
				if !exists {
					http.Error(w, "Missing value for replace operation", http.StatusBadRequest)
					return
				}
				currentObj[finalKey] = value
				
			default:
				http.Error(w, fmt.Sprintf("Unsupported modificationType: %s", modType), http.StatusBadRequest)
				return
			}
		} else {
			// Handle top-level paths
			switch modType {
			case "add":
				value, exists := itemDelta["value"]
				if !exists {
					http.Error(w, "Missing value for add operation", http.StatusBadRequest)
					return
				}
				obj[path] = value
				
			case "delete":
				delete(obj, path)
				
			case "replace":
				value, exists := itemDelta["value"]
				if !exists {
					http.Error(w, "Missing value for replace operation", http.StatusBadRequest)
					return
				}
				obj[path] = value
				
			default:
				http.Error(w, fmt.Sprintf("Unsupported modificationType: %s", modType), http.StatusBadRequest)
				return
			}
		}

		// Save changes
		svr.objects[id] = obj

		// Return the updated object
		respBody, _ := json.Marshal(obj)
		w.Write(respBody)
		return
	}

	// Just return the object for GET
	respBody, _ := json.Marshal(obj)
	w.Write(respBody)
}

// Helper function to load JSON from a file
func loadJSONFromFile(filePath string) (map[string]interface{}, error) {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	
	// Parse the JSON
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}
	
	return result, nil
}

func TestFilterKeys(t *testing.T) {
	// Load test data from files
	serviceMidpointJSON, err := loadJSONFromFile("/Users/jplana/src/terraform-provider-midpoint-restapi/service_from_midpoint.json")
	if err != nil {
		t.Fatalf("filter_keys_test.go: Failed to load service_from_midpoint.json: %s", err)
	}

	// Extract service ID
	serviceID := ""
	if service, ok := serviceMidpointJSON["service"].(map[string]interface{}); ok {
		if oid, ok := service["oid"].(string); ok {
			serviceID = oid
		}
	}
	
	if serviceID == "" {
		t.Fatalf("filter_keys_test.go: Could not extract service ID from service_from_midpoint.json")
	}

	// Create the test object
	testObjects := make(map[string]map[string]interface{})
	testObjects[serviceID] = serviceMidpointJSON

	debug := true

	// Start the test server
	svr := NewFilterKeysFakeServer(8083, testObjects, debug)
	defer svr.Shutdown()

	// Define keys to filter (these are fields present in service_from_midpoint.json but not in service.json)
	keysToFilter := []string{
		"@metadata",
		"version",
		"operationExecution",
		"iteration",
		"iterationToken",
		"linkRef",
		"activation",
	}

	// Create a client configured for MidPoint with filter keys
	client, err := NewAPIClient(&apiClientOpt{
		uri:                "http://127.0.0.1:8083/",
		insecure:           false,
		timeout:            5,
		idAttribute:        "oid",
		writeReturnsObject: true,
		updateMethod:       "PATCH", // Using PATCH for MidPoint integration
		debug:              debug,
	})

	if err != nil {
		t.Fatalf("filter_keys_test.go: Failed to create API client: %s", err)
	}

	// Create API object with filter keys
	objectOpts := &apiObjectOpts{
		path:       "/api/objects",
		id:         serviceID,
		debug:      debug,
		filterKeys: keysToFilter,
	}

	obj, err := NewAPIObject(client, objectOpts)
	if err != nil {
		t.Fatalf("filter_keys_test.go: Failed to create API object: %s", err)
	}

	// Test 1: Read the object and verify filtered keys are not in the state
	t.Run("FilterKeys_OnRead", func(t *testing.T) {
		// Read the object
		err = obj.readObject()
		if err != nil {
			t.Fatalf("filter_keys_test.go: Failed to read object: %s", err)
		}

		// Verify that filtered keys are not in the state
		for _, key := range keysToFilter {
			// Check if the key exists in the root level
			if _, exists := obj.apiData[key]; exists {
				t.Errorf("filter_keys_test.go: Filtered key '%s' should not be in apiData", key)
			}

			// For nested objects, check inside service if it exists
			if service, ok := obj.apiData["service"].(map[string]interface{}); ok {
				if _, exists := service[key]; exists {
					t.Errorf("filter_keys_test.go: Filtered key '%s' should not be in apiData['service']", key)
				}
			}
		}

		// Verify that essential data is still there
		if service, ok := obj.apiData["service"].(map[string]interface{}); ok {
			if service["oid"] != serviceID {
				t.Errorf("filter_keys_test.go: Expected apiData['service']['oid'] to be '%s', got '%v'",
					serviceID, service["oid"])
			}
			
			// Check for presence of keys that should not be filtered
			if service["name"] == nil {
				t.Errorf("filter_keys_test.go: Expected apiData['service']['name'] to exist, but it doesn't")
			}
			
			if service["description"] == nil {
				t.Errorf("filter_keys_test.go: Expected apiData['service']['description'] to exist, but it doesn't")
			}
		} else {
			t.Errorf("filter_keys_test.go: Expected apiData to contain 'service' map")
		}
	})

	// Test 2: Update the object and verify only necessary changes are sent
	t.Run("FilterKeys_OnUpdate", func(t *testing.T) {
		// Make a change to the object directly using the sendMidpointPatch method
		err := obj.sendMidpointPatch("replace", "service.description", "Updated service description")
		if err != nil {
			t.Fatalf("filter_keys_test.go: Failed to send patch: %s", err)
		}

		// Verify the PATCH request was made
		if svr.lastRequest.Method != "PATCH" {
			t.Fatalf("filter_keys_test.go: Expected PATCH request, got %s", svr.lastRequest.Method)
		}

		// Verify the last patch contains only the changed field
		if len(svr.patchHistory) == 0 {
			t.Fatalf("filter_keys_test.go: No PATCH requests were recorded")
		}

		// Get the last patch
		lastPatch := svr.patchHistory[len(svr.patchHistory)-1]
		
		// Unmarshal the patch request
		objectMod, ok := lastPatch["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Missing objectModification in PATCH request")
		}

		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Missing itemDelta in PATCH request")
		}

		// Verify we're only replacing the description field
		if itemDelta["modificationType"] != "replace" {
			t.Fatalf("filter_keys_test.go: Expected modificationType='replace', got '%v'",
				itemDelta["modificationType"])
		}

		expectedPath := "service.description"
		if itemDelta["path"] != expectedPath {
			t.Fatalf("filter_keys_test.go: Expected path='%s', got '%v'",
				expectedPath, itemDelta["path"])
		}

		if itemDelta["value"] != "Updated service description" {
			t.Fatalf("filter_keys_test.go: Expected value='Updated service description', got '%v'",
				itemDelta["value"])
		}

		// Verify that no filtered keys were included in the patch
		for _, patch := range svr.patchHistory {
			objectMod, ok := patch["objectModification"].(map[string]interface{})
			if !ok {
				continue
			}

			itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
			if !ok {
				continue
			}

			path, ok := itemDelta["path"].(string)
			if !ok {
				continue
			}

			// Check if the path matches any of the filtered keys
			for _, filteredKey := range keysToFilter {
				if path == filteredKey || strings.HasPrefix(path, filteredKey+".") {
					t.Fatalf("filter_keys_test.go: PATCH request includes filtered key '%s' at path '%s'",
						filteredKey, path)
				}
			}
		}
	})
	
	// Test 3: Complex nested structure filtering
	t.Run("FilterKeys_NestedStructures", func(t *testing.T) {
		// Load test data
		serviceData, ok := serviceMidpointJSON["service"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Failed to extract service data")
		}
		
		// Add a nested structure with metadata to filter out
		nestedData := map[string]interface{}{
			"@metadata": map[string]interface{}{
				"lastModified": "2025-04-08T14:38:34.676Z",
			},
			"value": "test value",
		}
		serviceData["testNested"] = nestedData
		
		// Update the server's object
		svr.objects[serviceID]["service"] = serviceData
		
		// Re-read the object
		err = obj.readObject()
		if err != nil {
			t.Fatalf("filter_keys_test.go: Failed to read object with nested structure: %s", err)
		}
		
		// Verify the nested @metadata was filtered out
		service, ok := obj.apiData["service"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Failed to get service from apiData after re-read")
		}
		
		nestedResult, ok := service["testNested"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Failed to get nested structure from apiData")
		}
		
		// The @metadata key should be filtered out
		if _, exists := nestedResult["@metadata"]; exists {
			t.Errorf("filter_keys_test.go: Nested filtered key '@metadata' should not be in apiData['service']['testNested']")
		}
		
		// The value key should still be there
		if nestedResult["value"] != "test value" {
			t.Errorf("filter_keys_test.go: Expected apiData['service']['testNested']['value'] to be 'test value', got '%v'",
				nestedResult["value"])
		}
	})
	
	// Test 4: Array filtering
	t.Run("FilterKeys_Arrays", func(t *testing.T) {
		// Load test data
		serviceData, ok := serviceMidpointJSON["service"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Failed to extract service data")
		}
		
		// Add an array with items containing metadata to filter out
		arrayData := []interface{}{
			map[string]interface{}{
				"@metadata": map[string]interface{}{
					"lastModified": "2025-04-08T14:38:34.676Z",
				},
				"id": "item1",
				"value": "array item 1",
			},
			map[string]interface{}{
				"@metadata": map[string]interface{}{
					"lastModified": "2025-04-08T14:38:34.676Z",
				},
				"id": "item2",
				"value": "array item 2",
			},
		}
		serviceData["testArray"] = arrayData
		
		// Update the server's object
		svr.objects[serviceID]["service"] = serviceData
		
		// Re-read the object
		err = obj.readObject()
		if err != nil {
			t.Fatalf("filter_keys_test.go: Failed to read object with array: %s", err)
		}
		
		// Verify the @metadata was filtered out from array items
		service, ok := obj.apiData["service"].(map[string]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Failed to get service from apiData after re-read")
		}
		
		arrayResult, ok := service["testArray"].([]interface{})
		if !ok {
			t.Fatalf("filter_keys_test.go: Failed to get array from apiData")
		}
		
		// Check each array item
		for i, item := range arrayResult {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("filter_keys_test.go: Array item %d is not a map", i)
			}
			
			// The @metadata key should be filtered out
			if _, exists := itemMap["@metadata"]; exists {
				t.Errorf("filter_keys_test.go: Filtered key '@metadata' should not be in array item %d", i)
			}
			
			// The other keys should still be there
			if itemMap["id"] == nil {
				t.Errorf("filter_keys_test.go: Expected array item %d to have 'id', but it doesn't", i)
			}
			
			if itemMap["value"] == nil {
				t.Errorf("filter_keys_test.go: Expected array item %d to have 'value', but it doesn't", i)
			}
		}
	})
}