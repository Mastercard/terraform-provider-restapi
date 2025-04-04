package restapi

import (
	"testing"
)

// TestMidpointMode tests that update_method="PATCH" is correctly detected
// and changes the behavior of updateObject()
func TestMidpointMode(t *testing.T) {
	debug := false
	
	// Initialize test data
	testApiObjects := make(map[string]map[string]interface{})
	testObj := map[string]interface{}{
		"Id":         "midpoint1",
		"name":       "testuser",
		"attribute1": "value1",
	}
	testApiObjects["midpoint1"] = testObj
	
	// Start a test server
	svr := NewMidpointFakeServer(8085, testApiObjects, debug)
	defer svr.Shutdown()
	
	// Create a standard client with PUT updates
	putClient, err := NewAPIClient(&apiClientOpt{
		uri:                "http://127.0.0.1:8085/",
		insecure:           false,
		timeout:            5,
		idAttribute:        "Id",
		writeReturnsObject: true,
		updateMethod:       "PUT", // Default behavior
		debug:              debug,
	})
	
	if err != nil {
		t.Fatalf("midpoint_mode_test.go: Failed to create PUT API client: %s", err)
	}
	
	// Create a Midpoint client with PATCH updates
	patchClient, err := NewAPIClient(&apiClientOpt{
		uri:                "http://127.0.0.1:8085/",
		insecure:           false,
		timeout:            5,
		idAttribute:        "Id",
		writeReturnsObject: true,
		updateMethod:       "PATCH", // Midpoint mode
		debug:              debug,
	})
	
	if err != nil {
		t.Fatalf("midpoint_mode_test.go: Failed to create PATCH API client: %s", err)
	}
	
	// Test that PUT is used when update_method="PUT"
	t.Run("Standard_PUT_Mode", func(t *testing.T) {
		objectOpts := &apiObjectOpts{
			path:  "/api/objects",
			id:    "midpoint1",
			debug: debug,
		}
		
		obj, err := NewAPIObject(putClient, objectOpts)
		if err != nil {
			t.Fatalf("midpoint_mode_test.go: Failed to create API object: %s", err)
		}
		
		// Read current state
		err = obj.readObject()
		if err != nil {
			t.Fatalf("midpoint_mode_test.go: Failed to read object: %s", err)
		}
		
		// Set a new attribute in the desired state
		obj.data = map[string]interface{}{
			"Id":         "midpoint1",
			"name":       "testuser",
			"attribute1": "value1",
			"newAttr":    "new value",
		}
		
		// Update the object
		err = obj.updateObject()
		if err != nil {
			t.Fatalf("midpoint_mode_test.go: Failed to update object: %s", err)
		}
		
		// Verify the method used was PUT
		if svr.lastRequest.Method != "PUT" {
			t.Fatalf("midpoint_mode_test.go: Expected PUT request, got %s", svr.lastRequest.Method)
		}
	})
	
	// Test that PATCH with ObjectModificationType is used when update_method="PATCH"
	t.Run("Midpoint_PATCH_Mode", func(t *testing.T) {
		objectOpts := &apiObjectOpts{
			path:  "/api/objects",
			id:    "midpoint1",
			debug: debug,
		}
		
		obj, err := NewAPIObject(patchClient, objectOpts)
		if err != nil {
			t.Fatalf("midpoint_mode_test.go: Failed to create API object: %s", err)
		}
		
		// Read current state
		err = obj.readObject()
		if err != nil {
			t.Fatalf("midpoint_mode_test.go: Failed to read object: %s", err)
		}
		
		// Set a new attribute in the desired state
		obj.data = map[string]interface{}{
			"Id":         "midpoint1",
			"name":       "testuser",
			"attribute1": "updated value", // Changed
			"newAttr2":   "another value", // Added
		}
		
		// Update the object
		err = obj.updateObject()
		if err != nil {
			t.Fatalf("midpoint_mode_test.go: Failed to update object: %s", err)
		}
		
		// Verify the method used was PATCH
		if svr.lastRequest.Method != "PATCH" {
			t.Fatalf("midpoint_mode_test.go: Expected PATCH request, got %s", svr.lastRequest.Method)
		}
		
		// Make sure the object was updated properly
		updatedObj := svr.objects["midpoint1"]
		if updatedObj["attribute1"] != "updated value" {
			t.Fatalf("midpoint_mode_test.go: Expected attribute1='updated value', got '%v'", 
				updatedObj["attribute1"])
		}
		
		if updatedObj["newAttr2"] != "another value" {
			t.Fatalf("midpoint_mode_test.go: Expected newAttr2='another value', got '%v'", 
				updatedObj["newAttr2"])
		}
	})
}