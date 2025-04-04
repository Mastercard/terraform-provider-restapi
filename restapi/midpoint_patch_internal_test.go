package restapi

import (
	"encoding/json"
	"testing"
)

func TestSendMidpointPatch(t *testing.T) {
	// Create test objects
	testObjects := make(map[string]map[string]interface{})
	testObject := map[string]interface{}{
		"Id":          "test1",
		"name":        "testname",
		"description": "test description",
	}
	testObjects["test1"] = testObject

	// Start the mock server
	svr := NewMidpointFakeServer(8083, testObjects, false)
	defer svr.Shutdown()

	// Create client and API object
	client, err := NewAPIClient(&apiClientOpt{
		uri:                "http://127.0.0.1:8083/",
		insecure:           false,
		timeout:            5,
		idAttribute:        "Id",
		writeReturnsObject: true,
		updateMethod:       "PATCH", // This is key for Midpoint integration
		debug:              false,
	})

	if err != nil {
		t.Fatalf("midpoint_patch_internal_test.go: Failed to create API client: %s", err)
	}

	objectOpts := &apiObjectOpts{
		path:  "/api/objects",
		id:    "test1",
		debug: false,
	}

	obj, err := NewAPIObject(client, objectOpts)
	if err != nil {
		t.Fatalf("midpoint_patch_internal_test.go: Failed to create API object: %s", err)
	}

	// Read current state
	err = obj.readObject()
	if err != nil {
		t.Fatalf("midpoint_patch_internal_test.go: Failed to read object: %s", err)
	}

	// Test sendMidpointPatch for add operation
	t.Run("Send_Add_Patch", func(t *testing.T) {
		err := obj.sendMidpointPatch("add", "newField", "new value")
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: sendMidpointPatch failed: %s", err)
		}

		// Verify the request
		var patchReq map[string]interface{}
		err = json.Unmarshal(svr.lastBody, &patchReq)
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: Failed to unmarshal PATCH request body: %s", err)
		}

		// Check structure
		objectMod, ok := patchReq["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_internal_test.go: Missing objectModification in request")
		}

		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_internal_test.go: Missing itemDelta in request")
		}

		// Check values
		if itemDelta["modificationType"] != "add" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected modificationType='add', got '%v'", 
				itemDelta["modificationType"])
		}

		if itemDelta["path"] != "newField" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected path='newField', got '%v'", 
				itemDelta["path"])
		}

		if itemDelta["value"] != "new value" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected value='new value', got '%v'", 
				itemDelta["value"])
		}
	})

	// Test sendMidpointPatch for delete operation
	t.Run("Send_Delete_Patch", func(t *testing.T) {
		err := obj.sendMidpointPatch("delete", "description", nil)
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: sendMidpointPatch failed: %s", err)
		}

		// Verify the request
		var patchReq map[string]interface{}
		err = json.Unmarshal(svr.lastBody, &patchReq)
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: Failed to unmarshal PATCH request body: %s", err)
		}

		// Check structure
		objectMod, ok := patchReq["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_internal_test.go: Missing objectModification in request")
		}

		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_internal_test.go: Missing itemDelta in request")
		}

		// Check values
		if itemDelta["modificationType"] != "delete" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected modificationType='delete', got '%v'", 
				itemDelta["modificationType"])
		}

		if itemDelta["path"] != "description" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected path='description', got '%v'", 
				itemDelta["path"])
		}

		// For delete operations, there should be no value
		if _, exists := itemDelta["value"]; exists {
			t.Fatalf("midpoint_patch_internal_test.go: Expected no value for delete operation, but found one")
		}
	})

	// Test sendMidpointPatch for replace operation
	t.Run("Send_Replace_Patch", func(t *testing.T) {
		err := obj.sendMidpointPatch("replace", "name", "updated name")
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: sendMidpointPatch failed: %s", err)
		}

		// Verify the request
		var patchReq map[string]interface{}
		err = json.Unmarshal(svr.lastBody, &patchReq)
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: Failed to unmarshal PATCH request body: %s", err)
		}

		// Check structure
		objectMod, ok := patchReq["objectModification"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_internal_test.go: Missing objectModification in request")
		}

		itemDelta, ok := objectMod["itemDelta"].(map[string]interface{})
		if !ok {
			t.Fatalf("midpoint_patch_internal_test.go: Missing itemDelta in request")
		}

		// Check values
		if itemDelta["modificationType"] != "replace" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected modificationType='replace', got '%v'", 
				itemDelta["modificationType"])
		}

		if itemDelta["path"] != "name" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected path='name', got '%v'", 
				itemDelta["path"])
		}

		if itemDelta["value"] != "updated name" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected value='updated name', got '%v'", 
				itemDelta["value"])
		}
	})
}

func TestPatchMidpointObject(t *testing.T) {
	// Create test objects
	testObjects := make(map[string]map[string]interface{})
	testObject := map[string]interface{}{
		"Id":          "test2",
		"name":        "testname",
		"description": "test description",
		"attribute1":  "value1",
	}
	testObjects["test2"] = testObject

	// Start the mock server
	svr := NewMidpointFakeServer(8084, testObjects, false)
	defer svr.Shutdown()

	// Create client and API object
	client, err := NewAPIClient(&apiClientOpt{
		uri:                "http://127.0.0.1:8084/",
		insecure:           false,
		timeout:            5,
		idAttribute:        "Id",
		writeReturnsObject: true,
		updateMethod:       "PATCH", // This is key for Midpoint integration
		debug:              false,
	})

	if err != nil {
		t.Fatalf("midpoint_patch_internal_test.go: Failed to create API client: %s", err)
	}

	objectOpts := &apiObjectOpts{
		path:  "/api/objects",
		id:    "test2",
		debug: false,
	}

	obj, err := NewAPIObject(client, objectOpts)
	if err != nil {
		t.Fatalf("midpoint_patch_internal_test.go: Failed to create API object: %s", err)
	}

	// Read current state
	err = obj.readObject()
	if err != nil {
		t.Fatalf("midpoint_patch_internal_test.go: Failed to read object: %s", err)
	}

	// Test patchMidpointObject with various changes
	t.Run("Patch_Multiple_Changes", func(t *testing.T) {
		// Set desired state with multiple differences from current state
		obj.data = map[string]interface{}{
			"Id":          "test2",           // same
			"name":        "updated name",    // changed
			"newField":    "new value",       // added
			// "description" removed
			"attribute1":  "value1",          // same
		}

		// Perform the patch
		err := obj.patchMidpointObject()
		if err != nil {
			t.Fatalf("midpoint_patch_internal_test.go: patchMidpointObject failed: %s", err)
		}

		// Verify the object in the server is updated correctly
		updatedObj := svr.objects["test2"]

		// Check the replaced field
		if updatedObj["name"] != "updated name" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected name='updated name', got '%v'", 
				updatedObj["name"])
		}

		// Check the new field
		if updatedObj["newField"] != "new value" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected newField='new value', got '%v'", 
				updatedObj["newField"])
		}

		// Check the deleted field
		if _, exists := updatedObj["description"]; exists {
			t.Fatalf("midpoint_patch_internal_test.go: Expected description to be deleted, but it still exists")
		}

		// Check the unchanged field
		if updatedObj["attribute1"] != "value1" {
			t.Fatalf("midpoint_patch_internal_test.go: Expected attribute1='value1', got '%v'", 
				updatedObj["attribute1"])
		}
	})
}