package restapi

import (
	"encoding/json"
	"fmt"
	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"log"
	"testing"
)

var test_debug = false
var http_server_debug = false
var api_object_debug = false
var api_client_debug = false

type test_api_object struct {
	Test_case string            `json:"Test_case"`
	Id        string            `json:"Id"`
	Revision  int               `json:"Revision,omitempty"`
	Thing     string            `json:"Thing,omitempty"`
	Is_cat    bool              `json:"Is_cat,omitempty"`
	Colors    []string          `json:"Colors,omitempty"`
	Attrs     map[string]string `json:"Attrs,omitempty"`
}

var testing_data_objects = []string{
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

var client, err = NewAPIClient(&apiClientOpt{
	uri:                   "http://127.0.0.1:8081/",
	insecure:              false,
	username:              "",
	password:              "",
	headers:               make(map[string]string, 0),
	timeout:               5,
	id_attribute:          "Id",
	copy_keys:             []string{"Thing"},
	write_returns_object:  true,
	create_returns_object: false,
	debug:                 api_client_debug,
})

func GenerateTestObjects(data_objects []string, t *testing.T, test_debug bool) (typed map[string]test_api_object, untyped map[string]map[string]interface{}) {
	/* Messy... fakeserver wants "generic" objects, but it is much easier
	   to write our test cases with typed (test_api_object) objects. Make
	   maps of both */
	typed = make(map[string]test_api_object)
	untyped = make(map[string]map[string]interface{})

	for _, data_object := range data_objects {
		test_obj, api_server_obj := add_test_api_object(data_object, t, test_debug)

		id := test_obj.Id
		test_case := test_obj.Test_case

		if test_debug {
			log.Printf("api_object_test.go: Adding test object for case '%s' as id '%s'\n", test_case, id)
		}
		typed[id] = test_obj

		if test_debug {
			log.Printf("api_object_test.go: Adding API server test object for case '%s' as id '%s'\n", test_case, id)
		}
		untyped[id] = api_server_obj
	}

	return typed, untyped
}

func add_test_api_object(input string, t *testing.T, test_debug bool) (test_obj test_api_object, api_server_obj map[string]interface{}) {
	if err := json.Unmarshal([]byte(input), &test_obj); err != nil {
		t.Fatalf("api_object_test.go: Failed to unmarshall JSON (to test_api_object) from '%s'", input)
	}

	if err := json.Unmarshal([]byte(input), &api_server_obj); err != nil {
		t.Fatalf("api_object_test.go: Failed to unmarshall JSON (to api_server_object) from '%s'", input)
	}

	return test_obj, api_server_obj
}

func TestAPIObject(t *testing.T) {
	generated_objects, api_server_objects := GenerateTestObjects(testing_data_objects, t, test_debug)

	/* Construct a local map of test case objects with only the ID populated */
	if test_debug {
		log.Println("api_object_test.go: Building test objects...")
	}

	/* Holds the full list of api_object items that we are testing
	   indexed by the name of the test case */
	var testing_objects = make(map[string]*api_object)

	for id, test_obj := range generated_objects {
		if test_debug {
			log.Printf("api_object_test.go:   '%s'\n", id)
		}

		object_opts := &apiObjectOpts{
			path:  "/api/objects",
			data:  fmt.Sprintf(`{ "Id": "%s" }`, id), /* Start with only an empty JSON object ID as our "data" */
			debug: api_object_debug,                  /* Whether the object's debug is enabled */
		}
		o, err := NewAPIObject(client, object_opts)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to create new api_object for id '%s'", id)
		}

		test_case := test_obj.Test_case
		testing_objects[test_case] = o
	}

	if test_debug {
		log.Println("api_object_test.go: Starting HTTP server")
	}
	svr := fakeserver.NewFakeServer(8081, api_server_objects, true, http_server_debug, "")

	/* Loop through all of the objects and GET their data from the server */
	t.Run("read_object", func(t *testing.T) {
		if test_debug {
			log.Printf("api_object_test.go: Testing read_object()")
		}
		for Test_case, _ := range testing_objects {
			t.Run(Test_case, func(t *testing.T) {
				if test_debug {
					log.Printf("api_object_test.go: Getting data for '%s' test case from server\n", Test_case)
				}
				err := testing_objects[Test_case].read_object()
				if err != nil {
					t.Fatalf("api_object_test.go: Failed to read data for test case '%s': %s", Test_case, err)
				}
			})
		}
	})

	/* Verify our copy_keys is happy by seeing if Thing made it into the data hash */
	t.Run("copy_keys", func(t *testing.T) {
		if test_debug {
			log.Printf("api_object_test.go: Testing copy_keys()")
		}
		if testing_objects["normal"].data["Thing"].(string) == "" {
			t.Fatalf("api_object_test.go: copy_keys for 'normal' object failed. Expected 'Thing' to be non-empty, but got '%+v'\n", testing_objects["normal"].data["Thing"])
		}
	})

	/* Go ahead and update one of our objects */
	t.Run("update_object", func(t *testing.T) {
		if test_debug {
			log.Printf("api_object_test.go: Testing update_object()")
		}
		testing_objects["minimal"].data["Thing"] = "spoon"
		testing_objects["minimal"].update_object()
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in update_object() test: %s", err)
		} else if testing_objects["minimal"].api_data["Thing"] != "spoon" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"spoon", testing_objects["minimal"].api_data["Thing"], testing_objects["minimal"])
		}
	})

	/* Delete one and make sure a 404 follows */
	t.Run("delete_object", func(t *testing.T) {
		if test_debug {
			log.Printf("api_object_test.go: Testing delete_object()")
		}
		testing_objects["pet"].delete_object()
		err = testing_objects["pet"].read_object()
		if err != nil {
			t.Fatalf("api_object_test.go: 'pet' object deleted, but an error was returned when reading the object (expected the provider to cope with this!\n")
		}
	})

	/* Recreate the one we just got rid of */
	t.Run("create_object", func(t *testing.T) {
		if test_debug {
			log.Printf("api_object_test.go: Testing create_object()")
		}
		testing_objects["pet"].data["Thing"] = "dog"
		err = testing_objects["pet"].create_object()
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in create_object() test: %s", err)
		} else if testing_objects["minimal"].api_data["Thing"] != "spoon" {
			t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"spoon", testing_objects["minimal"].api_data["Thing"], testing_objects["minimal"])
		}

		/* verify it's there */
		err = testing_objects["pet"].read_object()
		if err != nil {
			t.Fatalf("api_object_test.go: Failed in read_object() test: %s", err)
		} else if testing_objects["pet"].api_data["Thing"] != "dog" {
			t.Fatalf("api_object_test.go: Failed in create_object() test. Object created is xpected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"dog", testing_objects["minimal"].api_data["Thing"], testing_objects["minimal"])
		}
	})

	t.Run("find_object", func(t *testing.T) {
		object_opts := &apiObjectOpts{
			path:  "/api/objects",
			debug: api_object_debug,
		}
		object, err := NewAPIObject(client, object_opts)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to create new api_object to find")
		}

		query_string := ""
		search_key := "Thing"
		search_value := "dog"
		results_key := ""
		tmp_obj, err := object.find_object(query_string, search_key, search_value, results_key)
		if err != nil {
			t.Fatalf("api_object_test.go: Failed to find api_object: %s", search_value)
		}

		if object.id != "5" {
			t.Errorf("%s: expected populated object from search to be %s but got %s", search_value, "5", object.id)
		}

		if tmp_obj["Id"] != "5" {
			t.Errorf("%s: expected found object from search to be %s but got %s from %v", search_value, "5", tmp_obj["Id"], tmp_obj)
		}
	})

	if test_debug {
		log.Println("api_object_test.go: Stopping HTTP server")
	}
	svr.Shutdown()
	if test_debug {
		log.Println("api_object_test.go: Done")
	}
}
