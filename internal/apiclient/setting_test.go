package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var APISettingDebug = false

type testAPISetting struct {
	TestCase string            `json:"Test_case"`
	Revision int               `json:"Revision,omitempty"`
	Thing    string            `json:"Thing,omitempty"`
	IsCat    bool              `json:"Is_cat,omitempty"`
	Colors   []string          `json:"Colors,omitempty"`
	Attrs    map[string]string `json:"Attrs,omitempty"`
}

var testingDataSettings = []string{
	`{
	  "Test_case": "normal",
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
	  "Thing": "carrot"
	}`,
}

var clientSettingsOpts = APIClientOpt{
	URI:                 "http://127.0.0.1:8201/",
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
var clientSettings, _ = NewAPIClient(&clientSettingsOpts)

func generateTestSettings(dataObjects []string, t *testing.T) (typed map[string]testAPISetting, untyped map[string]map[string]interface{}) {
	typed = make(map[string]testAPISetting)
	untyped = make(map[string]map[string]interface{})

	for index, dataObject := range dataObjects {
		testObj := testAPISetting{}
		apiServerObj := make(map[string]interface{})
		if err := json.Unmarshal([]byte(dataObject), &testObj); err != nil {
			t.Fatalf("api_setting_test.go: Failed to unmarshall JSON (to test_api_object) from '%s'", dataObject)
		}
		if err := json.Unmarshal([]byte(dataObject), &apiServerObj); err != nil {
			t.Fatalf("api_setting_test.go: Failed to unmarshall JSON (to api_server_object) from '%s'", dataObject)
		}

		typed[fmt.Sprintf("test-%d", index)] = testObj
		untyped[fmt.Sprintf("test-%d", index)] = apiServerObj
	}
	return typed, untyped
}

func TestAPISetting(t *testing.T) {
	if apiClientDebug {
		os.Setenv("TF_LOG", "DEBUG")
	}

	ctx := context.Background()
	generatedObjects, apiServerObjects := generateTestSettings(testingDataSettings, t)

	// Will be populated later
	requiredHeaders := map[string]string{}

	// Construct a local map of test case objects with only the ID populated
	if testDebug {
		fmt.Println("api_setting_test.go: Building test objects...")
	}

	// Holds the full list of api_object items that we are testing
	// indexed by the name of the test case
	var testingObjects = make(map[string]*APISetting)

	for id, testObj := range generatedObjects {
		if testDebug {
			fmt.Printf("  '%s'\n", id)
		}

		objectOpts := &APISettingOpts{
			ReadPath:   "/api/objects/" + id,
			UpdatePath: "/api/objects/" + id,
			Data:       "{}",            // Start with only an empty JSON object ID as our "data"
			Debug:      APISettingDebug, // Whether the object's debug is enabled
		}
		o, err := NewAPISetting(clientSettings, objectOpts)
		if err != nil {
			t.Fatalf("api_setting_test.go: Failed to create new api_object for id '%s'", id)
		}

		testCase := testObj.TestCase
		testingObjects[testCase] = o
	}

	if testDebug {
		fmt.Println("api_setting_test.go: Starting HTTP server")
	}
	svr := fakeserver.NewFakeServer(8201, apiServerObjects, requiredHeaders, true, httpServerDebug, "")

	t.Run("read_object_with_read_data", func(t *testing.T) {
		for testCase := range testingObjects {
			t.Run(testCase, func(t *testing.T) {
				if testDebug {
					fmt.Printf("Getting data for '%s' test case from server\n", testCase)
				}
				if testingObjects[testCase].readData == nil {
					testingObjects[testCase].readData = make(map[string]interface{})
				}
				err := testingObjects[testCase].ReadSetting(ctx)
				if err != nil {
					t.Fatalf("api_setting_test.go: Failed to read data for test case '%s': %s", testCase, err)
				}
			})
		}
	})

	// Go ahead and update one of our objects
	t.Run("update_object", func(t *testing.T) {
		testingObjects["minimal"].data["Thing"] = "spoon"
		err := testingObjects["minimal"].UpdateSetting(ctx)
		if err != nil {
			t.Fatalf("api_setting_test.go: Failed in update_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "spoon" {
			t.Fatalf("api_setting_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"spoon", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	// Update once more with update_data
	t.Run("update_object_with_update_data", func(t *testing.T) {
		if testingObjects["minimal"].updateData == nil {
			testingObjects["minimal"].updateData = make(map[string]interface{})
		}
		testingObjects["minimal"].updateData["Thing"] = "knife"
		err := testingObjects["minimal"].UpdateSetting(ctx)
		if err != nil {
			t.Fatalf("api_setting_test.go: Failed in update_object() test: %s", err)
		} else if testingObjects["minimal"].apiData["Thing"] != "knife" {
			t.Fatalf("api_setting_test.go: Failed to update 'Thing' field of 'minimal' object. Expected it to be '%s' but it is '%s'\nFull obj: %+v\n",
				"knife", testingObjects["minimal"].apiData["Thing"], testingObjects["minimal"])
		}
	})

	t.Run("read_object_basic_auth", func(t *testing.T) {
		optsCopy := clientSettingsOpts
		optsCopy.Username = "testuser"
		optsCopy.Password = "testpass"
		basicAuthClient, _ := NewAPIClient(&optsCopy)

		tmpObject, _ := NewAPISetting(basicAuthClient, &APISettingOpts{
			ReadPath:   "/api/objects/test-0",
			UpdatePath: "/api/objects/test-0",
			ID:         "1",
		})

		// Fakeserver will expect a header that exactly matches this, simulating basic auth
		requiredHeaders["Authorization"] = "Basic dGVzdHVzZXI6dGVzdHBhc3M=" // base64(testuser:testpass)

		err := testingObjects["normal"].ReadSetting(ctx)
		if err == nil {
			t.Fatalf("api_setting_test.go: Expected error reading an object without basic auth configured, but none was found")
		}

		err = tmpObject.ReadSetting(ctx)
		if err != nil {
			t.Fatalf("api_setting_test.go: Failed to read data for test case 'normal' with basic auth: %s", err)
		}

		// Clean up shared data between test cases
		delete(requiredHeaders, "Authorization")
	})

	if testDebug {
		fmt.Println("api_setting_test.go: Stopping HTTP server")
	}
	svr.Shutdown()
	if testDebug {
		fmt.Println("api_setting_test.go: Done")
	}
}

func TestGetSettingApiDataAndResponse(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"test1": {
			"id":    "test1",
			"name":  "Test Object",
			"value": 123,
		},
	}
	svr := fakeserver.NewFakeServer(8201, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8201",
		Timeout: 2,
	})

	obj, _ := NewAPISetting(client, &APISettingOpts{
		ReadPath:   "/api/objects/test1",
		UpdatePath: "/api/objects/test1",
	})

	err := obj.ReadSetting(ctx)
	require.NoError(t, err, "ReadSetting should not return an error")

	apiData := obj.GetApiData()
	assert.NotNil(t, apiData, "GetApiData should return a map")
	assert.Equal(t, "Test Object", apiData["name"], "API data should contain correct name")
	assert.Equal(t, "123", apiData["value"], "API data should contain correct value")

	apiResponse := obj.GetApiResponse()
	assert.NotEmpty(t, apiResponse, "GetApiResponse should return non-empty string")
	assert.Contains(t, apiResponse, "test1", "API response should contain the object ID")
}

func TestSettingFullLifecycle(t *testing.T) {
	ctx := context.Background()

	testObjects := map[string]map[string]interface{}{
		"test1": {
			"name":  "Test Object",
			"value": 123,
		},
	}
	svr := fakeserver.NewFakeServer(8201, testObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	client, _ := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:8201",
		Timeout: 2,
	})

	obj, _ := NewAPISetting(client, &APISettingOpts{
		ReadPath:   "/api/objects/test1",
		UpdatePath: "/api/objects/test1",
		Data:       "{ \"name\": \"Test Object Changed\", \"value\": 456 }",
	})

	err := obj.CreateSetting(ctx)
	require.NoError(t, err, "Create should not return an error")

	apiData := obj.GetApiData()
	assert.NotNil(t, apiData, "GetApiData should return a map")
	assert.Equal(t, "Test Object Changed", apiData["name"], "API data should contain correct name")
	assert.Equal(t, "456", apiData["value"], "API data should contain correct value")

	err = obj.DeleteSetting(ctx)
	require.NoError(t, err, "Update should not return an error")

	err = obj.ReadSetting(ctx)
	require.NoError(t, err, "Read after delete should not return an error")
	apiData = obj.GetApiData()
	assert.NotNil(t, apiData, "GetApiData should return a map")
	assert.Equal(t, "Test Object", apiData["name"], "API data should contain correct name")
	assert.Equal(t, "123", apiData["value"], "API data should contain correct value")
}

func TestSettingInitialStateRehydration(t *testing.T) {
	client, err := NewAPIClient(&APIClientOpt{
		URI:     "http://127.0.0.1:1",
		Timeout: 1,
	})
	require.NoError(t, err)

	obj, err := NewAPISetting(client, &APISettingOpts{
		ReadPath:     "/api/objects/resource",
		UpdatePath:   "/api/objects/resource",
		Data:         `{"first":"Managed"}`,
		InitialState: `{"first":"Initial","last":"State","extra":"keep"}`,
		UpdateMethod: "PUT",
		ReadMethod:   "GET",
		QueryString:  "",
		UpdateData:   "",
		ReadData:     "",
		ID:           "",
		IDAttribute:  "",
		Headers:      map[string]string{},
	})
	require.NoError(t, err)

	assert.Equal(t, `{"extra":"keep","first":"Initial","last":"State"}`, obj.GetInitialStateResponse())
}

func TestFilterInitialStateByData(t *testing.T) {
	initial := map[string]interface{}{
		"first": "Initial",
		"last":  "State",
		"meta": map[string]interface{}{
			"owner": "server",
			"tags":  []interface{}{"a", "b"},
		},
		"extra": "server-only",
	}

	data := map[string]interface{}{
		"first": "Managed",
		"meta": map[string]interface{}{
			"tags": []interface{}{"override"},
		},
	}

	filtered := filterInitialStateByData(initial, data)

	assert.Equal(t, "Initial", filtered["first"])
	assert.NotContains(t, filtered, "last")
	assert.NotContains(t, filtered, "extra")

	nested, ok := filtered["meta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"a", "b"}, nested["tags"])
	assert.NotContains(t, nested, "owner")
}
