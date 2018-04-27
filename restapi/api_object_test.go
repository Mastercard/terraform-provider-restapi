package restapi

import (
  "log"
  "testing"
  "net/http"
  "time"
  "encoding/json"
  "fmt"
  "io/ioutil"
  "strings"
)

var api_object_server *http.Server

var test_debug = false
var http_server_debug = false
var api_object_debug = false
var api_client_debug = false

/* Populated with generate_test_api_objects */
var test_api_objects map[string]test_api_object

type test_api_object struct {
  Test_case string             `json:"Test_case"`
  Id        string             `json:"Id"`
  Revision  int                `json:"Revision,omitempty"`
  Thing     string             `json:"Thing,omitempty"`
  Is_cat    bool               `json:"Is_cat,omitempty"`
  Colors    []string           `json:"Colors,omitempty"`
  Attrs     map[string]string  `json:"Attrs,omitempty"`
}

func TestAPIObject(t *testing.T) {
  var testing_objects map[string]*api_object
  var err error

  if test_debug { log.Println("api_object_test.go: Creating test API objects") }
  generate_test_api_objects(t, test_debug)

  if test_debug { log.Println("api_object_test.go: Starting HTTP server") }
  setup_api_object_server(test_debug)

  client := NewAPIClient (
    "http://127.0.0.1:8081/",  /* URL */
    false,                     /* insecure */
    "",                        /* username */
    "",                        /* password */
    "",                        /* Authorization header */
    5,                         /* HTTP Timeout in seconds */
    "Id",                      /* Attribute from server that serves as ID */
    []string{ "Thing" },       /* keys to copy from api_data to data */
    true,                      /* Write returns object */
    false,                     /* Create returns object */
    api_client_debug,          /* Debug logging */
    )

  /* Construct a local map of test case objects with only the ID populated */
  if test_debug { log.Println("api_object_test.go: Building test objects...") }
  testing_objects = make(map[string]*api_object)

  for id, api_obj := range test_api_objects {
    if test_debug { log.Printf("api_object_test.go:   '%s'\n", id) }

    o, err := NewAPIObject(
      client,                            /* The HTTP client created above */
      "/api",                            /* path to the "object" in the test server (note: id will automatically be appended) */
      "",                                /* Do not set an ID to force the constructor to verify id_attribute works */
      fmt.Sprintf(`{ "Id": "%s" }`, id), /* Start with only an empty JSON object ID as our "data" */
      api_object_debug,                  /* Whether the object's debug is enabled */
    )
    if err != nil {
      t.Fatalf("api_object_test.go: Failed to create new api_object for id '%s'", id)
    } else {
      test_case := api_obj.Test_case
      testing_objects[test_case] = o
    }
  }

  /* Loop through all of the objects and GET their data from the server */
  log.Printf("api_object_test.go: Testing read_object()")
  for Test_case, _ := range testing_objects {
    if test_debug { log.Printf("api_object_test.go: Getting data for '%s' test case from server\n", Test_case) }
    err := testing_objects[Test_case].read_object()
    if err != nil {
      t.Fatalf("api_object_test.go: Failed to read data for test case '%s': %s", Test_case, err)
    }
  }

  /* Verify our copy_keys is happy by seeing if Thing made it into the data hash */
  log.Printf("api_object_test.go: Testing copy_keys()")
  if testing_objects["normal"].data["Thing"].(string) == "" {
    t.Fatalf("api_object_test.go: copy_keys for 'normal' object failed. Expected 'Thing' to be non-empty, but got '%+v'\n", testing_objects["normal"].data["Thing"])
  }

  /* Go ahead and update one of our objects */
  log.Printf("api_object_test.go: Testing update_object()")
  testing_objects["minimal"].data["Thing"] = "spoon"
  testing_objects["minimal"].update_object()
  if err != nil {
    t.Fatalf("api_object_test.go: Failed in update_object() test: %s", err)
  } else if testing_objects["minimal"].api_data["Thing"] != "spoon" {
    t.Fatalf("api_object_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
      "spoon", testing_objects["minimal"].api_data["Thing"], testing_objects["minimal"])
  }

  /* Delete one and make sure a 404 follows */
  log.Printf("api_object_test.go: Testing delete_object()")
  testing_objects["pet"].delete_object()
  err = testing_objects["pet"].read_object()
  if err == nil {
    t.Fatalf("api_object_test.go: 'pet' object deleted, but 404 not returned when getting it.\n")
  }

  /* Recreate the one we just got rid of */
  log.Printf("api_object_test.go: Testing create_object()")
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

  if test_debug { log.Println("api_object_test.go: Stopping HTTP server") }
  shutdown_api_object_server()
  if test_debug { log.Println("api_object_test.go: Done") }
}



/* HTTP handler that will read an object, store it in our list
   of tested objects and print the object back to the caller */
func handle_api_object (w http.ResponseWriter, r *http.Request) {
  var obj test_api_object
  var id string
  var ok bool

  /* Assume this will never fail */
  b, _ := ioutil.ReadAll(r.Body)

  if http_server_debug {
    log.Printf("api_object_test.go http_server: Recieved request: %+v\n", r)
    log.Printf("api_object_test.go http_server: BODY: %s\n", string(b))
    log.Printf("api_object_test.go http_server: Test cases and IDs:\n")
    for id, obj := range test_api_objects {
      log.Printf("  %s: %s\n", id, obj.Test_case)
    }
  }

  parts := strings.Split(r.RequestURI, "/")

  /* If it was a valid request, there will be three parts
     and the ID will exist */
  if len(parts) == 3 {
    id = parts[2]
    obj, ok = test_api_objects[id];
    if http_server_debug { log.Printf("api_object_test.go http_server: Detected ID %s (exists: %t, method: %s)", id, ok, r.Method) }
    /* Make sure the object requested exists unless it's being created */
    if !ok {
      http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
      return
    }
  } else if r.RequestURI != "/api" {
    /* How did something get to this handler with the wrong number of args??? */
    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
    return
  }

  if r.Method == "DELETE" {
    /* Get rid of this one */
    delete(test_api_objects, id)
    if http_server_debug { log.Printf("api_object_test.go http_server: Object deleted.\n") }
    return
  }
  /* if data was sent, parse the data */
  if string(b) != "" {
    err := json.Unmarshal(b, &obj)

    if err != nil {
      /* Failure goes back to the user as a 500. Log data here for
         debugging (which shouldn't ever fail!) */
      log.Fatalf("api_object_test.go http_server: Unmarshal of request failed: %s\n", err);
      log.Fatalf("\nBEGIN passed data:\n%s\nEND passed data.", string(b));
      http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
      return
    } else {
      /* In the case of POST above, id is not yet known - set it here */
      if id == "" { id = obj.Id }

      /* Overwrite our stored test object */
      if http_server_debug {
        log.Printf("api_object_test.go http_server: Overwriting %s with new data:%+v\n", id, obj)
      }
      test_api_objects[id] = obj

      /* Coax the data we were sent back to JSON and send it to the user */
      b, _ := json.Marshal(obj)
      w.Write(b)
      return
    }
  } else {
    /* No data was sent... must be just a retrieval */
    if http_server_debug { log.Printf("api_object_test.go http_server: Returning object.\n") }
    b, _ := json.Marshal(obj)
    w.Write(b)
    return
  }

  /* All cases by now should have already returned... something wasn't handled */
  http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
  return
}


/* Bind an HTTP server to localhost and populate "objects" with
   /api/{id} from our map of test objects */
func setup_api_object_server (test_debug bool) {
  serverMux := http.NewServeMux()

  for id, obj := range test_api_objects {
    if test_debug { log.Printf("api_object_test.go:   Adding handler for '/api/%s' => '%s' test case\n", id, obj.Test_case) }
    serverMux.HandleFunc(fmt.Sprintf("/api/%s", id), handle_api_object)
  }

  /* Add one for POST to just /api, too */
  serverMux.HandleFunc("/api", handle_api_object)

  api_object_server = &http.Server{
    Addr: "127.0.0.1:8081",
    Handler: serverMux,
  }

  go api_object_server.ListenAndServe()

  /* Let the server start */
  time.Sleep(1 * time.Second)
}

func shutdown_api_object_server () {
  api_object_server.Close()
}

func generate_test_api_objects (t *testing.T, test_debug bool) {
  test_api_objects = make(map[string]test_api_object)

  add_test_api_object(
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
    }`, t, test_debug)

  add_test_api_object(
    `{
      "Test_case": "minimal",
      "Id": "2",
      "Thing": "fork"
    }`, t, test_debug)

  add_test_api_object(
    `{
      "Test_case": "no Colors",
      "Id": "3",
      "Thing": "paper",
      "Is_cat": false,
      "Attrs": {
        "height": "8.5 in",
        "width": "11 in"
      }
    }`, t, test_debug)
  add_test_api_object(
    `{
      "Test_case": "no Attrs",
      "Id": "4",
      "Thing": "nothing",
      "Is_cat": false,
      "Colors": [
        "none"
      ]
    }`, t, test_debug)

  add_test_api_object(
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
    }`, t, test_debug)
}

func add_test_api_object (input string, t *testing.T, test_debug bool) {
  var obj test_api_object
  err := json.Unmarshal([]byte(input), &obj)

  if err != nil {
    t.Fatalf("api_object_test.go: Failed to unmarshall JSON from '%s'", input)
  } else {
    id := obj.Id
    Test_case := obj.Test_case
    if test_debug { log.Printf("api_object_test.go: Adding test object for case '%s' as id '%s'\n", Test_case, id) }
    test_api_objects[id] = obj
  }
}
