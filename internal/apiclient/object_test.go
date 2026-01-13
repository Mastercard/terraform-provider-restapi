package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDebug = false
var httpServerDebug = false
var apiObjectDebug = false
var apiClientDebug = false

type testAPIObject struct {
	TestCase string            `json:"Test_case"`
	ID       string            `json:"Id"`
	Revision int               `json:"Revision,omitempty"`
	Thing    string            `json:"Thing,omitempty"`
	IsCat    bool              `json:"Is_cat,omitempty"`
	Colors   []string          `json:"Colors,omitempty"`
	Attrs    map[string]string `json:"Attrs,omitempty"`
}

var testingDataObjects = []string{
	`{
	  "Test_case": "normal",
	  "Id": "1",
	  "Revision": 1,
	  "Thing": "potato",
	  "Is_cat": false,
	  "Colors": [
		"orange",
		"white"
	  ],
	  "Attrs": {
		"size": "6 in",
		"weight": "10 oz"
	  }
	}`,
	`{
      "Test_case": "minimal",
      "Id": "2",
      "Thing": "fork"
    }`,
	`{
      "Test_case": "no Colors",
      "Id": "3",
      "Thing": "paper",
      "Is_cat": false,
      "Attrs": {
        "height": "8.5 in",
        "width": "11 in"
      }
    }`,
	`{
      "Test_case": "no Attrs",
      "Id": "4",
      "Thing": "nothing",
      "Is_cat": false,
      "Colors": [
        "none"
      ]
    }`,
	`{
      "Test_case": "pet",
      "Id": "5",
      "Thing": "cat",
      "Is_cat": true,
      "Colors": [
        "orange",
        "white"
      ],
      "Attrs": {
        "size": "1.5 ft",
        "weight": "15 lb"
      }
	}`,
}

var clientOpts = APIClientOpt{
	URI:                 "http://127.0.0.1:8081/",
	Insecure:            false,
	Username:            "",
	Password:            "",
	Headers:             make(map[string]string),
	Timeout:             5,
	IDAttribute:         "Id",
	CopyKeys:            []string{"Thing"},
	WriteReturnsObject:  true,
	CreateReturnsObject: false,
	Debug:               apiClientDebug,
}
var client, _ = NewAPIClient(&clientOpts)

func generateTestObjects(dataObjects []string, t *testing.T) (typed map[string]testAPIObject, untyped map[string]map[string]interface{}) {
	// Messy... fakeserver wants "generic" objects, but it is much easier
	// to write our test cases with typed (test_api_object) objects. Make
	// maps of both
	typed = make(map[string]testAPIObject)
	untyped = make(map[string]map[string]interface{})

	for _, dataObject := range dataObjects {
		testObj := testAPIObject{}
		apiServerObj := make(map[string]interface{})
		if err := json.Unmarshal([]byte(dataObject), &testObj); err != nil {
			t.Fatalf("api_object_test.go: Failed to unmarshall JSON (to test_api_object) from '%s'", dataObject)
		}
		if err := json.Unmarshal([]byte(dataObject), &apiServerObj); err != nil {
			t.Fatalf("api_object_test.go: Failed to unmarshall JSON (to api_server_object) from '%s'", dataObject)
		}

		id := testObj.ID
		typed[id] = testObj
		untyped[id] = apiServerObj
	}

	return typed, untyped
}

func TestAPIObject(t *testing.T) {
	if apiClientDebug {
		os.Setenv("TF_LOG", "DEBUG")
	}

	ctx := context.Background()
	generatedObjects, apiServerObjects := generateTestObjects(testingDataObjects, t)

	// Will be populated later
	requiredHeaders := map[string]string{}

	// Construct a local map of test case objects with only the ID populated
	if testDebug {
		fmt.Println("api_object_test.go: Building test objects...")
	}

	// Holds the full list of api_object items that we are testing
	// indexed by the name of the test case
	var testingObjects = make(map[string]*APIObject)

	for id, testObj := range generatedObjects {
		if testDebug {
			fmt.Printf("  '%s'\n", id)
		}

		objectOpts := &APIObjectOpts{
			Path:  "/api/objects",
			Data:  fmt.Sprintf(`{ "Id": "%s" }`, id), // Start with only an empty JSON object ID as our "data"
			Debug: apiObjectDebug,                    // Whether the object's debug is enabled
		}
		o, err := NewAPIObject(client, objectOpts)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to create new api_object for id '%s'", id)
		}

		testCase := testObj.TestCase
		testingObjects[testCase] = o
	}

	if testDebug {
		fmt.Println("api_object_test.go: Starting HTTP server")
	}
	svr := fakeserver.NewFakeServer(8081, apiServerObjects, requiredHeaders, true, httpServerDebug, "")

	// Loop through all of the objects and GET their data from the server
	t.Run("read_object", func(t *testing.T) {
		for testCase := range testingObjects {
			t.Run(testCase, func(t *testing.T) {
				if testDebug {
					fmt.Printf("Getting data for '%s' test case from server\n", testCase)
				}
				err := testingObjects[testCase].ReadObject(ctx)
				if err != nil {
					t.Fatalf("api_object_test.go: Failed to read data for test case '%s': %s", testCase, err)
				}
			})
		}
	})

	t.Run("read_object_with_read_data", func(t *testing.T) {
		for testCase := range testingObjects {
			t.Run(testCase, func(t *testing.T) {
				if testDebug {
					fmt.Printf("Getting data for '%s' test case from server\n", testCase)
				}
				if testingObjects[testCase].readData == nil {
					testingObjects[testCase].readData = make(map[string]interface{})
				}
				testingObjects[testCase].readData["path"] = "/" + testCase
				err := testingObjects[testCase].ReadObject(ctx)
				if err != nil {
					t.Fatalf("api_object_test.go: Failed to read data for test case '%s': %s", testCase, err)
				}
			})
		}
	})

	// Verify our copy_keys is happy by changing a thing server-side and seeing if the new Thing made it into the data after re-reading the object
	t.Run("copy_keys", func(t *testing.T) {
		apiServerObjects["1"]["Thing"] = strings.ReplaceAll(apiServerObjects["1"]["Thing"].(string), "potato", "carrot")
		err := testingObjects["normal"].ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in copy_keys() test: %s", err)
		}
		if testingObjects["normal"].data["Thing"].(string) != "carrot" {
			t.Fatalf("api_object_test.go: copy_keys for 'normal' object failed. Expected 'Thing' to be 'carrot'', but got '%+v'\n", testingObjects["normal"].data["Thing"])
		}
	})

	// Go ahead and update one of our objects
	t.Run("update_object", func(t *testing.T) {
		testingObjects["minimal"].data["Thing"] = "spoon"
		err := testingObjects["minimal"].UpdateObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in update_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "spoon" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"spoon", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	// Update once more with update_data
	t.Run("update_object_with_update_data", func(t *testing.T) {
		if testingObjects["minimal"].updateData == nil {
			testingObjects["minimal"].updateData = make(map[string]interface{})
		}
		testingObjects["minimal"].updateData["Thing"] = "knife"
		err := testingObjects["minimal"].UpdateObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in update_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "knife" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"knife", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	// Delete one and make sure a 404 follows
	t.Run("delete_object", func(t *testing.T) {
		testingObjects["pet"].DeleteObject(ctx)
		err := testingObjects["pet"].ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: 'pet' object deleted, but an error was returned when reading the object (expected the provider to cope with this!\n")
		}
	})

	// Recreate the one we just got rid of
	t.Run("create_object", func(t *testing.T) {
		testingObjects["pet"].data["Thing"] = "dog"
		err := testingObjects["pet"].CreateObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in create_object() test: %s", err)
		} else if testingObjects["pet"].apiData["Thing"] != "dog" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"dog", testingObjects["pet"].apiData["Thing"], testingObjects["pet"])
		}

		// verify it's there
		err = testingObjects["pet"].ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in read_object() test: %s", err)
		} else if testingObjects["pet"].apiData["Thing"] != "dog" {
			t.Fatalf("api_object_test.go: Failed in create_object() test. Object created is xpected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"dog", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	t.Run("find_object", func(t *testing.T) {
		objectOpts := &APIObjectOpts{
			Path:  "/api/objects",
			Debug: apiObjectDebug,
		}
		object, err := NewAPIObject(client, objectOpts)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to create new api_object to find")
		}

		queryString := ""
		searchKey := "Thing"
		searchValue := "dog"
		resultsKey := ""
		searchData := ""
		tmpObj, err := object.FindObject(ctx, queryString, searchKey, searchValue, resultsKey, searchData)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to find api_object %s - %s", searchValue, err)
		}

		if object.ID != "5" {
			t.Errorf("%s: expected populated object from search to be %s but got %s", searchValue, "5", object.ID)
		}

		if tmpObj["Id"] != "5" {
			t.Errorf("%s: expected found object from search to be %s but got %s from %v", searchValue, "5", tmpObj["Id"], tmpObj)
		}
	})

	// Delete it again with destroy_data and make sure a 404 follows
	t.Run("delete_object_with_destroy_data", func(t *testing.T) {
		if testingObjects["pet"].destroyData == nil {
			testingObjects["pet"].destroyData = make(map[string]interface{})
		}
		testingObjects["pet"].destroyData["destroy"] = "true"
		testingObjects["pet"].DeleteObject(ctx)
		err := testingObjects["pet"].ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: 'pet' object deleted, but an error was returned when reading the object (expected the provider to cope with this!\n")
		}
	})

	t.Run("read_object_basic_auth", func(t *testing.T) {
		optsCopy := clientOpts
		optsCopy.Username = "testuser"
		optsCopy.Password = "testpass"
		basicAuthClient, _ := NewAPIClient(&optsCopy)

		tmpObject, _ := NewAPIObject(basicAuthClient, &APIObjectOpts{ID: "1"})

		// Fakeserver will expect a header that exactly matches this, simulating basic auth
		requiredHeaders["Authorization"] = "Basic dGVzdHVzZXI6dGVzdHBhc3M=" // base64(testuser:testpass)

		err := testingObjects["normal"].ReadObject(ctx)
		if err == nil {
			t.Fatalf("api_object_test.go: Expected error reading an object without basic auth configured, but none was found")
		}

		err = tmpObject.ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to read data for test case 'normal' with basic auth: %s", err)
		}

		// Clean up shared data between test cases
		delete(requiredHeaders, "Authorization")
	})

	if testDebug {
		fmt.Println("api_object_test.go: Stopping HTTP server")
	}
	svr.Shutdown()
	if testDebug {
		fmt.Println("api_object_test.go: Done")
	}
}

func TestGetApiDataAndResponse(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"test1": {
			"id":    "test1",
			"name":  "Test Object",
			"value": 123,
		},
	}
	svr := fakeserver.NewFakeServer(8081, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8081",
		Timeout: 2,
	})

	obj, _ := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		ID:   "test1",
	})

	err := obj.ReadObject(ctx)
	require.NoError(t, err, "ReadObject should not return an error")

	apiData := obj.GetApiData()
	assert.NotNil(t, apiData, "GetApiData should return a map")
	assert.Equal(t, "test1", apiData["id"], "API data should contain correct id")
	assert.Equal(t, "Test Object", apiData["name"], "API data should contain correct name")
	assert.Equal(t, "123", apiData["value"], "API data should contain correct value")

	apiResponse := obj.GetApiResponse()
	assert.NotEmpty(t, apiResponse, "GetApiResponse should return non-empty string")
	assert.Contains(t, apiResponse, "test1", "API response should contain the object ID")
}

func TestReadObject404Handling(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{}
	svr := fakeserver.NewFakeServer(8081, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8081",
		Timeout: 2,
	})

	obj, _ := NewAPIObject(client, &APIObjectOpts{
		Path: "/api/objects",
		ID:   "nonexistent",
	})

	err := obj.ReadObject(ctx)

	assert.NoError(t, err, "ReadObject should not return error on 404")
	assert.Equal(t, "", obj.ID, "ID should be cleared after 404")
}

func TestReadObjectWithReadSearch(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"obj1": {
			"id":     "obj1",
			"name":   "First Object",
			"status": "active",
		},
		"obj2": {
			"id":     "obj2",
			"name":   "Second Object",
			"status": "inactive",
		},
		"obj3": {
			"id":     "obj3",
			"name":   "Third Object",
			"status": "active",
		},
	}
	svr := fakeserver.NewFakeServer(8081, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8081",
		Timeout: 2,
	})

	obj, _ := NewAPIObject(client, &APIObjectOpts{
		Path:       "/api/objects",
		SearchPath: "/api/objects",
		ID:         "obj2",
		ReadSearch: map[string]string{
			"search_key":   "name",
			"search_value": "Second Object",
		},
	})

	err := obj.ReadObject(ctx)
	require.NoError(t, err, "ReadObject with search should not return an error")

	assert.Equal(t, "obj2", obj.ID, "Should find correct object by search")

	apiData := obj.GetApiData()
	assert.Equal(t, "Second Object", apiData["name"], "Should have correct object data from search")
}

func TestReadObjectSearchNotFound(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"obj1": {
			"id":   "obj1",
			"name": "Existing Object",
		},
	}
	svr := fakeserver.NewFakeServer(8081, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8081",
		Timeout: 2,
	})

	obj, _ := NewAPIObject(client, &APIObjectOpts{
		Path:       "/api/objects",
		SearchPath: "/api/objects",
		ID:         "obj999",
		ReadSearch: map[string]string{
			"search_key":   "name",
			"search_value": "NonExistent Object",
		},
	})

	err := obj.ReadObject(ctx)
	assert.NoError(t, err, "ReadObject with no search results should not return error")
	assert.Equal(t, "", obj.ID, "ID should be cleared when search finds nothing")
}

func TestFindObjectEdgeCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		objects       map[string]map[string]interface{}
		searchKey     string
		searchValue   string
		resultsKey    string
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty_results_array",
			objects:       map[string]map[string]interface{}{},
			searchKey:     "name",
			searchValue:   "test",
			expectError:   true,
			errorContains: "failed to find an object",
		},
		{
			name: "missing_search_key_in_results",
			objects: map[string]map[string]interface{}{
				"obj1": {
					"id":          "obj1",
					"other_field": "value",
				},
			},
			searchKey:     "name",
			searchValue:   "test",
			expectError:   true,
			errorContains: "failed to get the value of 'name'",
		},
		{
			name: "missing_id_attribute",
			objects: map[string]map[string]interface{}{
				"obj1": {
					"name": "test",
				},
			},
			searchKey:     "name",
			searchValue:   "test",
			expectError:   true,
			errorContains: "failed to find id_attribute",
		},
		{
			name: "successful_search",
			objects: map[string]map[string]interface{}{
				"obj1": {
					"id":   "obj1",
					"name": "test",
				},
			},
			searchKey:   "name",
			searchValue: "test",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svr := fakeserver.NewFakeServer(8081, tt.objects, map[string]string{}, true, false, "")
			defer svr.Shutdown()

			client, _ := NewAPIClient(&APIClientOpt{
				URI:     "http://127.0.0.1:8081",
				Timeout: 2,
			})

			obj, _ := NewAPIObject(client, &APIObjectOpts{
				Path:       "/api/objects",
				SearchPath: "/api/objects",
			})

			result, err := obj.FindObject(ctx, "", tt.searchKey, tt.searchValue, tt.resultsKey, "")

			if tt.expectError {
				assert.Error(t, err, "FindObject should return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "FindObject should not return an error")
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

func TestFindObjectWithResultsKey(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"obj1": {
			"id":   "obj1",
			"name": "Test Object",
		},
	}
	svr := fakeserver.NewFakeServer(8081, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8081",
		Timeout: 2,
	})

	obj, _ := NewAPIObject(client, &APIObjectOpts{
		Path:       "/api/objects",
		SearchPath: "/api/object_list", // This endpoint returns {"list": [...], "results": true, ...}
	})

	result, err := obj.FindObject(ctx, "", "name", "Test Object", "list", "")
	assert.NoError(t, err, "FindObject with results_key should not return an error")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Equal(t, "obj1", obj.ID, "Should find the correct object ID")
}

func TestSetDataFromFound(t *testing.T) {
	client, _ := NewAPIClient(&APIClientOpt{
		URI:         "http://127.0.0.1:8081",
		Timeout:     2,
		IDAttribute: "id",
	})

	obj, _ := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		IDAttribute: "id",
	})

	// Simulate data returned from FindObject
	foundData := map[string]interface{}{
		"id":    "test123",
		"name":  "Test User",
		"email": "test@example.com",
		"nested": map[string]interface{}{
			"field": "value",
		},
	}

	// Set the data using SetDataFromMap
	err := obj.SetDataFromMap(foundData)
	assert.NoError(t, err, "SetDataFromMap should not return an error")

	// Verify the object ID was set
	assert.Equal(t, "test123", obj.ID, "Object ID should be set from found data")

	// Verify api_data is populated correctly
	apiData := obj.GetApiData()
	assert.Equal(t, "test123", apiData["id"], "api_data should contain id")
	assert.Equal(t, "Test User", apiData["name"], "api_data should contain name")
	assert.Equal(t, "test@example.com", apiData["email"], "api_data should contain email")

	// Verify api_response contains the JSON
	apiResponse := obj.GetApiResponse()
	assert.Contains(t, apiResponse, "test123", "api_response should contain the data as JSON")
	assert.Contains(t, apiResponse, "Test User", "api_response should contain the data as JSON")

	// Test with invalid data (empty map should fail to set ID)
	obj2, _ := NewAPIObject(client, &APIObjectOpts{
		Path:        "/api/objects",
		IDAttribute: "id",
	})

	emptyData := map[string]interface{}{}
	err = obj2.SetDataFromMap(emptyData)
	assert.Error(t, err, "SetDataFromMap should return error when ID cannot be extracted")
}
