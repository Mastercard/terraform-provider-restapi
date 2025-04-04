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

// Modified version of the fakeserver to handle Midpoint-specific PATCH requests
type midpointFakeServer struct {
	server      *http.Server
	objects     map[string]map[string]interface{}
	debug       bool
	running     bool
	lastRequest *http.Request
	lastBody    []byte
}

func NewMidpointFakeServer(port int, objects map[string]map[string]interface{}, debug bool) *midpointFakeServer {
	serverMux := http.NewServeMux()

	svr := &midpointFakeServer{
		debug:   debug,
		objects: objects,
		running: false,
	}

	serverMux.HandleFunc("/api/", svr.handleAPIObject)

	apiObjectServer := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: serverMux,
	}

	svr.server = apiObjectServer

	if debug {
		log.Printf("midpoint_patch_test.go: Starting fake Midpoint server on port %d", port)
	}

	go apiObjectServer.ListenAndServe()
	/* Sleep a bit to let the server start up */
	time.Sleep(100 * time.Millisecond)
	svr.running = true

	return svr
}

func (svr *midpointFakeServer) Shutdown() {
	if svr.running {
		svr.server.Close()
		svr.running = false
		// Give some time for the server to shut down
		time.Sleep(100 * time.Millisecond) 
	}
}

func (svr *midpointFakeServer) handleAPIObject(w http.ResponseWriter, r *http.Request) {
	var obj map[string]interface{}
	var id string
	var ok bool

	// Save the last request for test verification
	svr.lastRequest = r
	
	// Read the request body
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("midpoint_patch_test.go: ERROR - Failed to read request body: %s\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	svr.lastBody = b
	
	if svr.debug {
		log.Printf("midpoint_patch_test.go: Received %s request to %s\n", r.Method, r.URL.Path)
		log.Printf("midpoint_patch_test.go: Request body: %s\n", string(b))
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
	} else if path == "/api/objects" && r.Method == "GET" {
		// Return all objects for GET /api/objects
		result := make([]map[string]interface{}, 0)
		for _, hash := range svr.objects {
			result = append(result, hash)
		}
		respBody, _ := json.Marshal(result)
		w.Write(respBody)
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
			log.Printf("midpoint_patch_test.go: Failed to unmarshal PATCH data: %s\n", err)
			http.Error(w, "Invalid JSON in patch request", http.StatusBadRequest)
			return
		}
		
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
			if svr.debug {
				log.Printf("midpoint_patch_test.go: Deleted attribute '%s'", path)
				log.Printf("midpoint_patch_test.go: Object after deletion: %v", obj)
			}
			
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
		
		// Save changes
		svr.objects[id] = obj
		
		// Return the updated object
		respBody, _ := json.Marshal(obj)
		w.Write(respBody)
		return
	}
	
	// Handle POST/PUT normally
	if len(b) > 0 {
		err := json.Unmarshal(b, &obj)
		if err != nil {
			log.Printf("midpoint_patch_test.go: Failed to unmarshal request data: %s\n", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		
		// For POST, extract ID from the request
		if id == "" {
			if val, ok := obj["id"]; ok {
				id = fmt.Sprintf("%v", val)
			} else if val, ok := obj["Id"]; ok {
				id = fmt.Sprintf("%v", val)
			} else if val, ok := obj["ID"]; ok {
				id = fmt.Sprintf("%v", val)
			} else {
				http.Error(w, "POST sent with no id field in the data", http.StatusBadRequest)
				return
			}
		}
		
		// Save object
		svr.objects[id] = obj
		
		// Return the object
		respBody, _ := json.Marshal(obj)
		w.Write(respBody)
		return
	}
	
	// Just return the object for GET
	respBody, _ := json.Marshal(obj)
	w.Write(respBody)
}

func TestMidpointPatchIntegration(t *testing.T) {
	// Initialize test data
	testObject := map[string]interface{}{
		"Id":          "user1",
		"name":        "jsmith",
		"givenName":   "John",
		"familyName":  "Smith",
		"description": "Initial description",
	}
	
	testObjects := make(map[string]map[string]interface{})
	testObjects["user1"] = testObject
	
	debug := false
	
	// Start the test server
	svr := NewMidpointFakeServer(8082, testObjects, debug)
	defer svr.Shutdown()
	
	// Create a client configured for Midpoint PATCH
	client, err := NewAPIClient(&apiClientOpt{
		uri:                 "http://127.0.0.1:8082/",
		insecure:            false,
		timeout:             5,
		idAttribute:         "Id",
		writeReturnsObject:  true,
		updateMethod:        "PATCH", // This is the key setting for Midpoint integration
		debug:               debug,
	})
	
	if err != nil {
		t.Fatalf("midpoint_patch_test.go: Failed to create API client: %s", err)
	}
	
	// Create API object
	objectOpts := &apiObjectOpts{
		path:  "/api/objects",
		id:    "user1",
		debug: debug,
	}
	
	obj, err := NewAPIObject(client, objectOpts)
	if err != nil {
		t.Fatalf("midpoint_patch_test.go: Failed to create API object: %s", err)
	}
	
	// Read current state
	err = obj.readObject()
	if err != nil {
		t.Fatalf("midpoint_patch_test.go: Failed to read object: %s", err)
	}
	
	// Verify initial state
	if obj.apiData["name"] != "jsmith" {
		t.Fatalf("midpoint_patch_test.go: Initial state incorrect, expected name='jsmith', got '%v'", obj.apiData["name"])
	}
	
	// Test 1: Add a new attribute
	t.Run("Add_Attribute", func(t *testing.T) {
		// Set a new attribute in the desired state
		obj.data = make(map[string]interface{})
		for k, v := range obj.apiData {
			obj.data[k] = v
		}
		obj.data["emailAddress"] = "john.smith@example.com"
		
		// Update the object (should use PATCH)
		err = obj.updateObject()
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to update object: %s", err)
		}
		
		// Verify PATCH request was made with the right data
		if svr.lastRequest.Method != "PATCH" {
			t.Fatalf("midpoint_patch_test.go: Expected PATCH request, got %s", svr.lastRequest.Method)
		}
		
		var patchReq map[string]interface{}
		err = json.Unmarshal(svr.lastBody, &patchReq)
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to unmarshal PATCH request body: %s", err)
		}
		
		// Verify the ObjectModificationType structure
		objectMod, ok := patchReq["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_test.go: Missing objectModification in PATCH request")
		}
		
		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_test.go: Missing itemDelta in PATCH request")
		}
		
		if itemDelta["modificationType"] != "add" {
			t.Fatalf("midpoint_patch_test.go: Expected modificationType='add', got '%v'", 
				itemDelta["modificationType"])
		}
		
		if itemDelta["path"] != "emailAddress" {
			t.Fatalf("midpoint_patch_test.go: Expected path='emailAddress', got '%v'", 
				itemDelta["path"])
		}
		
		if itemDelta["value"] != "john.smith@example.com" {
			t.Fatalf("midpoint_patch_test.go: Expected value='john.smith@example.com', got '%v'", 
				itemDelta["value"])
		}
		
		// Verify the state was updated correctly
		if obj.apiData["emailAddress"] != "john.smith@example.com" {
			t.Fatalf("midpoint_patch_test.go: State not updated correctly, expected emailAddress='john.smith@example.com', got '%v'", 
				obj.apiData["emailAddress"])
		}
	})
	
	// Test 2: Modify an existing attribute
	t.Run("Modify_Attribute", func(t *testing.T) {
		// Change an existing attribute in the desired state
		obj.data["description"] = "Updated description"
		
		// Update the object (should use PATCH)
		err = obj.updateObject()
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to update object: %s", err)
		}
		
		// Verify PATCH request was made with the right data
		if svr.lastRequest.Method != "PATCH" {
			t.Fatalf("midpoint_patch_test.go: Expected PATCH request, got %s", svr.lastRequest.Method)
		}
		
		var patchReq map[string]interface{}
		err = json.Unmarshal(svr.lastBody, &patchReq)
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to unmarshal PATCH request body: %s", err)
		}
		
		// Verify the ObjectModificationType structure
		objectMod, ok := patchReq["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_test.go: Missing objectModification in PATCH request")
		}
		
		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_test.go: Missing itemDelta in PATCH request")
		}
		
		if itemDelta["modificationType"] != "replace" {
			t.Fatalf("midpoint_patch_test.go: Expected modificationType='replace', got '%v'", 
				itemDelta["modificationType"])
		}
		
		if itemDelta["path"] != "description" {
			t.Fatalf("midpoint_patch_test.go: Expected path='description', got '%v'", 
				itemDelta["path"])
		}
		
		if itemDelta["value"] != "Updated description" {
			t.Fatalf("midpoint_patch_test.go: Expected value='Updated description', got '%v'", 
				itemDelta["value"])
		}
		
		// Verify the state was updated correctly
		if obj.apiData["description"] != "Updated description" {
			t.Fatalf("midpoint_patch_test.go: State not updated correctly, expected description='Updated description', got '%v'", 
				obj.apiData["description"])
		}
	})
	
	// Test 3: Delete an attribute using direct method
	t.Run("Delete_Attribute", func(t *testing.T) {
		// Directly call sendMidpointPatch instead of relying on update logic
		err := obj.sendMidpointPatch("delete", "description", nil)
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to send patch: %s", err)
		}
		
		// Verify PATCH request was made with the right data
		if svr.lastRequest.Method != "PATCH" {
			t.Fatalf("midpoint_patch_test.go: Expected PATCH request, got %s", svr.lastRequest.Method)
		}
		
		var patchReq map[string]interface{}
		err = json.Unmarshal(svr.lastBody, &patchReq)
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to unmarshal PATCH request body: %s", err)
		}
		
		// Verify the ObjectModificationType structure
		objectMod, ok := patchReq["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_test.go: Missing objectModification in PATCH request")
		}
		
		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_test.go: Missing itemDelta in PATCH request")
		}
		
		if itemDelta["modificationType"] != "delete" {
			t.Fatalf("midpoint_patch_test.go: Expected modificationType='delete', got '%v'", 
				itemDelta["modificationType"])
		}
		
		if itemDelta["path"] != "description" {
			t.Fatalf("midpoint_patch_test.go: Expected path='description', got '%v'", 
				itemDelta["path"])
		}
		
		// For delete operations, there should be no value
		if _, exists := itemDelta["value"]; exists {
			t.Fatalf("midpoint_patch_test.go: Expected no value for delete operation, but found one")
		}
		
		// Re-read the object to ensure we have the latest state
		err = obj.readObject()
		if err != nil {
			t.Fatalf("midpoint_patch_test.go: Failed to re-read object: %s", err)
		}
		
		// Check the updated state in the server directly
		if _, exists := svr.objects["user1"]["description"]; exists {
			t.Fatalf("midpoint_patch_test.go: Attribute not deleted in server state")
		}
	})
}