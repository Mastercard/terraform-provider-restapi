package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
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

var client, err = NewAPIClient(&APIClientOpt{
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
})

func generateTestObjects(dataObjects []string, t *testing.T, testDebug bool) (typed map[string]testAPIObject, untyped map[string]map[string]interface{}) {
	// Messy... fakeserver wants "generic" objects, but it is much easier
	// to write our test cases with typed (test_api_object) objects. Make
	// maps of both
	typed = make(map[string]testAPIObject)
	untyped = make(map[string]map[string]interface{})

	for _, dataObject := range dataObjects {
		testObj, apiServerObj := addTestAPIObject(dataObject, t, testDebug)

		id := testObj.ID
		testCase := testObj.TestCase

		if testDebug {
			fmt.Printf("Adding test object for case '%s' as id '%s'\n", testCase, id)
		}
		typed[id] = testObj

		if testDebug {
			fmt.Printf("Adding API server test object for case '%s' as id '%s'\n", testCase, id)
		}
		untyped[id] = apiServerObj
	}

	return typed, untyped
}

func addTestAPIObject(input string, t *testing.T, testDebug bool) (testObj testAPIObject, apiServerObj map[string]interface{}) {
	if err := json.Unmarshal([]byte(input), &testObj); err != nil {
		t.Fatalf("api_object_test.go: Failed to unmarshall JSON (to test_api_object) from '%s'", input)
	}

	if err := json.Unmarshal([]byte(input), &apiServerObj); err != nil {
		t.Fatalf("api_object_test.go: Failed to unmarshall JSON (to api_server_object) from '%s'", input)
	}

	return testObj, apiServerObj
}

func TestAPIObject(t *testing.T) {
	ctx := context.Background()
	generatedObjects, apiServerObjects := generateTestObjects(testingDataObjects, t, testDebug)

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
	svr := fakeserver.NewFakeServer(8081, apiServerObjects, true, httpServerDebug, "")

	// Loop through all of the objects and GET their data from the server
	t.Run("read_object", func(t *testing.T) {
		if testDebug {
			fmt.Printf("Testing read_object()")
		}
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
		if testDebug {
			fmt.Printf("Testing read_object() with read_data")
		}
		for testCase := range testingObjects {
			t.Run(testCase, func(t *testing.T) {
				if testDebug {
					fmt.Printf("Getting data for '%s' test case from server\n", testCase)
				}
				testingObjects[testCase].readData["path"] = "/" + testCase
				err := testingObjects[testCase].ReadObject(ctx)
				if err != nil {
					t.Fatalf("api_object_test.go: Failed to read data for test case '%s': %s", testCase, err)
				}
			})
		}
	})

	// Verify our copy_keys is happy by seeing if Thing made it into the data hash
	t.Run("copy_keys", func(t *testing.T) {
		if testDebug {
			fmt.Printf("Testing copy_keys()")
		}
		if testingObjects["normal"].data["Thing"].(string) == "" {
			t.Fatalf("api_object_test.go: copy_keys for 'normal' object failed. Expected 'Thing' to be non-empty, but got '%+v'\n", testingObjects["normal"].data["Thing"])
		}
	})

	// Go ahead and update one of our objects
	t.Run("update_object", func(t *testing.T) {
		if testDebug {
			fmt.Printf("Testing update_object()")
		}
		testingObjects["minimal"].data["Thing"] = "spoon"
		testingObjects["minimal"].UpdateObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in update_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "spoon" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"spoon", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	// Update once more with update_data
	t.Run("update_object_with_update_data", func(t *testing.T) {
		if testDebug {
			fmt.Printf("Testing update_object() with update_data")
		}
		testingObjects["minimal"].updateData["Thing"] = "knife"
		testingObjects["minimal"].UpdateObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in update_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "knife" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"knife", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	// Delete one and make sure a 404 follows
	t.Run("delete_object", func(t *testing.T) {
		if testDebug {
			fmt.Printf("Testing delete_object()")
		}
		testingObjects["pet"].DeleteObject(ctx)
		err = testingObjects["pet"].ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: 'pet' object deleted, but an error was returned when reading the object (expected the provider to cope with this!\n")
		}
	})

	// Recreate the one we just got rid of
	t.Run("create_object", func(t *testing.T) {
		if testDebug {
			fmt.Printf("Testing create_object()")
		}
		testingObjects["pet"].data["Thing"] = "dog"
		err = testingObjects["pet"].CreateObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in create_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "knife" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"knife", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
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
		if testDebug {
			fmt.Printf("Testing delete_object() with destroy_data")
		}
		testingObjects["pet"].destroyData["destroy"] = "true"
		testingObjects["pet"].DeleteObject(ctx)
		err = testingObjects["pet"].ReadObject(ctx)
		if err != nil {
			t.Fatalf("api_object_test.go: 'pet' object deleted, but an error was returned when reading the object (expected the provider to cope with this!\n")
		}
	})

	if testDebug {
		fmt.Println("api_object_test.go: Stopping HTTP server")
	}
	svr.Shutdown()
	if testDebug {
		fmt.Println("api_object_test.go: Done")
	}
}
