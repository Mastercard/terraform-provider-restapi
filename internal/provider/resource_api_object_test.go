package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRestApiObject_Basic(t *testing.T) {
	debug := false

	if debug {
		os.Setenv("TF_LOG", "DEBUG")
	}
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8119, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8119/",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IDAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
		Debug:               debug,
	}
	client, err := apiclient.NewAPIClient(opt)
	if err != nil {
		t.Fatal(err)
	}

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: generateTestResource(
					"Foo",
					`{ "id": "1234", "first": "Foo", "last": "Bar" }`,
					make(map[string]interface{}),
					debug,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Foo", "1234", client),
					resource.TestCheckResourceAttr("restapi_object.Foo", "id", "1234"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "api_data.first", "Foo"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "api_data.last", "Bar"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "api_response", "{\"first\":\"Foo\",\"id\":\"1234\",\"last\":\"Bar\"}"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "create_response", "{\"first\":\"Foo\",\"id\":\"1234\",\"last\":\"Bar\"}"),
				),
			},
			// Try updating the object and check create_response is unmodified
			{
				Config: generateTestResource(
					"Foo",
					`{ "id": "1234", "first": "Updated", "last": "Value" }`,
					make(map[string]interface{}),
					debug,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Foo", "1234", client),
					resource.TestCheckResourceAttr("restapi_object.Foo", "id", "1234"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "api_data.first", "Updated"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "api_data.last", "Value"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "api_response", "{\"first\":\"Updated\",\"id\":\"1234\",\"last\":\"Value\"}"),
					resource.TestCheckResourceAttr("restapi_object.Foo", "create_response", "{\"first\":\"Foo\",\"id\":\"1234\",\"last\":\"Bar\"}"),
				),
			},
			// Make a complex object with id_attribute as a child of another key
			// Note that we have to pass "id" just so fakeserver won't get angry at us
			{
				Config: generateTestResource(
					"Bar",
					`{ "id": "4321", "attributes": { "id": "4321" }, "config": { "first": "Bar", "last": "Baz" } }`,
					map[string]interface{}{
						"id_attribute": "attributes/id",
					},
					debug,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Bar", "4321", client),
					resource.TestCheckResourceAttr("restapi_object.Bar", "id", "4321"),
					resource.TestCheckResourceAttrSet("restapi_object.Bar", "api_data.config"),
				),
			},
		},
	})

	svr.Shutdown()
}

func testAccCheckRestapiObjectExists(n string, id string, client *apiclient.APIClient) resource.TestCheckFunc {
	ctx := context.Background()
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			keys := make([]string, 0, len(s.RootModule().Resources))
			for k := range s.RootModule().Resources {
				keys = append(keys, k)
			}
			return fmt.Errorf("RestAPI object not found in terraform state: %s. Found: %s", n, strings.Join(keys, ", "))
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("RestAPI object id not set in terraform")
		}

		// Make a throw-away API object to read from the API
		path := "/api/objects"
		opts := &apiclient.APIObjectOpts{
			Path:        path,
			ID:          id,
			IDAttribute: "id",
			Data:        "{}",
			Debug:       false,
		}
		obj, err := apiclient.NewAPIObject(client, opts)
		if err != nil {
			return err
		}

		err = obj.ReadObject(ctx)
		if err != nil {
			return err
		}

		return nil
	}
}

// This function generates a terraform JSON configuration from
// a name, JSON data and a list of params to set by coaxing it
// all to maps and then serializing to JSON
func generateTestResource(name string, data string, params map[string]interface{}, debug bool) string {
	return generateTestResourceWithURI(name, data, params, debug, "http://127.0.0.1:8119")
}

func generateTestResourceWithURI(name string, data string, params map[string]interface{}, debug bool, uri string) string {
	strData, _ := json.Marshal(data)
	config := []string{
		`path = "/api/objects"`,
		fmt.Sprintf(`debug = %t`, debug),
		fmt.Sprintf("data = %s", strData),
	}
	for k, v := range params {
		switch val := v.(type) {
		case string, bool, int:
			v = fmt.Sprintf(`"%v"`, val)
		default:
			marshaled, _ := json.Marshal(val)
			v = string(marshaled)
		}
		entry := fmt.Sprintf(`%s = %s`, k, v)
		config = append(config, entry)
	}
	strConfig := ""
	for _, v := range config {
		strConfig = strConfig + v + "\n"
	}

	return fmt.Sprintf(`
provider "restapi" {
  uri = "%s"
}

resource "restapi_object" "%s" {
%s
}
`, uri, name, strConfig)
}

func mockServer(host string, returnCodes map[string]int, responses map[string]string) *http.Server {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/api/", func(w http.ResponseWriter, req *http.Request) {
		key := fmt.Sprintf("%s %s", req.Method, req.RequestURI) // e.g. "PUT /api/objects/1234"
		returnCode, ok := returnCodes[key]
		if !ok {
			returnCode = http.StatusOK
		}
		w.WriteHeader(returnCode)
		responseBody, ok := responses[key]
		if !ok {
			responseBody = ""
		}
		w.Write([]byte(responseBody))
	})
	srv := &http.Server{
		Addr:    host,
		Handler: serverMux,
	}
	go srv.ListenAndServe()
	return srv
}

func TestAccRestApiObject_FailedUpdate(t *testing.T) {
	debug := false

	host := "127.0.0.1:8119"
	returnCodes := map[string]int{
		"PUT /api/objects/1234": http.StatusBadRequest,
	}
	responses := map[string]string{
		"GET /api/objects/1234": `{ "id": "1234", "foo": "Bar" }`,
	}
	srv := mockServer(host, returnCodes, responses)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create the resource
				Config: generateTestResourceWithURI(
					"Foo",
					`{ "id": "1234", "foo": "Bar" }`,
					make(map[string]interface{}),
					debug,
					"http://"+host,
				),
				Check: resource.TestCheckResourceAttr("restapi_object.Foo", "data", `{ "id": "1234", "foo": "Bar" }`),
			},
			{
				// Try update. It will fail because we return 400 for PUT operations from mock server
				Config: generateTestResourceWithURI(
					"Foo",
					`{ "id": "1234", "foo": "Updated" }`,
					make(map[string]interface{}),
					debug,
					"http://"+host,
				),
				ExpectError: regexp.MustCompile("unexpected response code '400'"),
			},
			{
				// Expecting plan to be non-empty because the failed apply above shouldn't update terraform state
				Config: generateTestResourceWithURI(
					"Foo",
					`{ "id": "1234", "foo": "Updated" }`,
					make(map[string]interface{}),
					debug,
					"http://"+host,
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccRestApiObject_PlanAfterApply tests that running plan after apply
// does not produce errors. This reproduces the issue from GitHub issue #344
// where the data attribute was read as empty during plan.
func TestAccRestApiObject_PlanAfterApply(t *testing.T) {
	debug := false

	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8120, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Create the resource
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8120"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id    = "test-1"
    name  = "Test Object"
    value = 42
  })
}
`,
			},
			{
				// Run plan only with same config - should succeed without errors
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8120"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id    = "test-1"
    name  = "Test Object"
    value = 42
  })
}
`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRestApiObject_PlanAfterApply_IgnoreAllServerChanges tests issue #344
// with ignore_all_server_changes enabled. This feature copies state.Data to plan.Data,
// which could cause issues if state.Data is null or empty.
func TestAccRestApiObject_PlanAfterApply_IgnoreAllServerChanges(t *testing.T) {
	debug := false

	// Pre-populate the server with an object that has extra fields
	apiServerObjects := map[string]map[string]interface{}{
		"test-ign-1": {
			"id":          "test-ign-1",
			"name":        "Test Object",
			"value":       42,
			"extra_field": "server-added",
		},
	}

	svr := fakeserver.NewFakeServer(8121, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Create the resource with ignore_all_server_changes
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8121"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id    = "test-ign-1"
    name  = "Test Object"
    value = 42
  })
  ignore_all_server_changes = true
}
`,
			},
			{
				// Run plan only - should succeed without "Invalid JSON String Value" error
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8121"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id    = "test-ign-1"
    name  = "Test Object"
    value = 42
  })
  ignore_all_server_changes = true
}
`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRestApiObject_PlanAfterApply_ComplexJSON tests issue #344 with complex
// nested JSON that includes conditionals and variables (similar to SCC WP module).
// This tests the scenario where JSON might have different structure during plan.
func TestAccRestApiObject_PlanAfterApply_ComplexJSON(t *testing.T) {
	debug := false

	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8122, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Create resource with complex nested JSON structure
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8122"
}

locals {
  enabled = true
  account_id = "acc-12345"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id = "complex-1"
    parameters = {
      enable_feature = local.enabled
      target_accounts = local.enabled ? [
        {
          account_id = local.account_id
          account_type = "standard"
        }
      ] : []
    }
  })
}
`,
			},
			{
				// Run plan only with same config - should succeed without errors
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8122"
}

locals {
  enabled = true
  account_id = "acc-12345"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id = "complex-1"
    parameters = {
      enable_feature = local.enabled
      target_accounts = local.enabled ? [
        {
          account_id = local.account_id
          account_type = "standard"
        }
      ] : []
    }
  })
}
`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRestApiObject_PatchMethod tests issue #344 with PATCH method configuration
// similar to the SCC workload protection module scenario.
func TestAccRestApiObject_PatchMethod(t *testing.T) {
	debug := false

	// Pre-populate with an existing resource
	apiServerObjects := map[string]map[string]interface{}{
		"patch-1": {
			"id":         "patch-1",
			"parameters": map[string]interface{}{"enabled": true},
		},
	}

	svr := fakeserver.NewFakeServer(8123, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Create using PATCH method (like SCC WP module)
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8123"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id = "patch-1"
    parameters = {
      enabled = true
    }
  })
  create_method  = "PATCH"
  update_method  = "PATCH"
  destroy_method = "PATCH"
}
`,
			},
			{
				// Run plan only - should succeed without "Invalid JSON String Value" error
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8123"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id = "patch-1"
    parameters = {
      enabled = true
    }
  })
  create_method  = "PATCH"
  update_method  = "PATCH"
  destroy_method = "PATCH"
}
`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRestApiObject_PlanAfterApply_IgnoreServerAdditions tests issue #344 with
// ignore_server_additions enabled. This feature was being tested when the bug was discovered.
// The issue is that data attribute is read as empty during plan phase.
func TestAccRestApiObject_PlanAfterApply_IgnoreServerAdditions(t *testing.T) {
	// Pre-populate the server with an object that has many extra fields
	// (similar to what IBM Cloud SCC WP API returns)
	apiServerObjects := map[string]map[string]interface{}{
		"ign-add-1": {
			"id":   "ign-add-1",
			"name": "Test Object",
			"parameters": map[string]interface{}{
				"enable_cspm": true,
				"target_accounts": []interface{}{
					map[string]interface{}{
						"account_id":   "acc-12345",
						"account_type": "standard",
					},
				},
			},
			// Extra fields added by server that should be ignored
			"created_at":       "2024-01-01T00:00:00Z",
			"updated_at":       "2024-01-02T00:00:00Z",
			"created_by":       "system",
			"resource_version": "v1.0.0",
			"metadata": map[string]interface{}{
				"region":     "us-south",
				"crn":        "crn:v1:...",
				"account_id": "acc-12345",
			},
		},
	}

	svr := fakeserver.NewFakeServer(8124, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Create the resource with ignore_server_additions
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8124"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "ign-add-1"
    name = "Test Object"
    parameters = {
      enable_cspm = true
      target_accounts = [
        {
          account_id   = "acc-12345"
          account_type = "standard"
        }
      ]
    }
  })
  ignore_server_additions = true
}
`,
			},
			{
				// Run plan only - this is where the "Invalid JSON String Value" error occurred
				// The bug was that state.Data was empty during plan phase
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8124"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "ign-add-1"
    name = "Test Object"
    parameters = {
      enable_cspm = true
      target_accounts = [
        {
          account_id   = "acc-12345"
          account_type = "standard"
        }
      ]
    }
  })
  ignore_server_additions = true
}
`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRestApiObject_IgnoreServerAdditions_WithChangesTo tests the combination
// of ignore_server_additions with ignore_changes_to, as this is how the feature
// is designed to work (ignore_server_additions modifies the behavior of delta detection).
func TestAccRestApiObject_IgnoreServerAdditions_WithChangesTo(t *testing.T) {
	apiServerObjects := map[string]map[string]interface{}{
		"ign-combo-1": {
			"id":   "ign-combo-1",
			"name": "Test Object",
			"parameters": map[string]interface{}{
				"enabled": true,
			},
			// Server-added fields
			"last_modified": "2024-01-01T00:00:00Z",
			"version":       "2",
		},
	}

	svr := fakeserver.NewFakeServer(8125, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8125"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "ign-combo-1"
    name = "Test Object"
    parameters = {
      enabled = true
    }
  })
  ignore_server_additions = true
  ignore_changes_to       = ["last_modified", "version"]
}
`,
			},
			{
				// Run plan only after apply
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8125"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "ign-combo-1"
    name = "Test Object"
    parameters = {
      enabled = true
    }
  })
  ignore_server_additions = true
  ignore_changes_to       = ["last_modified", "version"]
}
`,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRestApiObject_UnknownDataInPlan tests issue #344 specifically - when plan.Data
// contains unknown values (from computed fields or data sources), the ModifyPlan function
// should not error with "Error Parsing Plan Data: unexpected end of JSON input".
// This test uses timestamp() to create a dependency that makes the data unknown during plan.
func TestAccRestApiObject_UnknownDataInPlan(t *testing.T) {
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8126, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// First create a basic resource
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8126"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "unknown-test-1"
    name = "Initial"
  })
}
`,
			},
			{
				// Now change the config to depend on a computed value.
				// The timestamp() function returns a value that's unknown during plan.
				// This should NOT cause "Error Parsing Plan Data" error after our fix.
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8126"
}

locals {
  # Use a value that changes each time to force plan to have unknown data
  dynamic_name = "Updated-${timestamp()}"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "unknown-test-1"
    name = local.dynamic_name
  })
}
`,
				// We expect this to plan successfully (not error) even though data contains unknown
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccRestApiObject_IgnoreServerAdditions_DetectsServerModifications verifies that
// api_data reflects server-side modifications even with ignore_server_additions=true.
func TestAccRestApiObject_IgnoreServerAdditions_DetectsServerModifications(t *testing.T) {
	// Start with server data matching user config
	apiServerObjects := map[string]map[string]interface{}{
		"detect-mod-1": {
			"id":   "detect-mod-1",
			"name": "Original",
			// Server-added fields (should be ignored)
			"created_at": "2024-01-01T00:00:00Z",
			"version":    "1",
		},
	}

	svr := fakeserver.NewFakeServer(8128, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Step 1: Create resource with ignore_server_additions
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8128"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "detect-mod-1"
    name = "Original"
  })
  ignore_server_additions = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.name", "Original"),
				),
			},
			{
				// Step 2: Simulate server modifying a user-configured field
				// by updating the server data before running plan
				PreConfig: func() {
					// Modify the server's data to simulate the server changing
					// a field that the user explicitly configured
					apiServerObjects["detect-mod-1"]["name"] = "Server Modified"
					apiServerObjects["detect-mod-1"]["version"] = "2"
				},
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8128"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "detect-mod-1"
    name = "Original"
  })
  ignore_server_additions = true
}
`,
				// With ignore_server_additions, user-configured fields that the server
				// modifies SHOULD be detected as drift (unlike ignore_all_server_changes).
				// The api_data should reflect what the server has.
				Check: resource.ComposeTestCheckFunc(
					// api_data should show what the server returned
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.name", "Server Modified"),
				),
			},
		},
	})
}

// TestAccRestApiObject_IgnoreServerAdditions_UserConfigChanges verifies that user
// config changes are detected and applied even with ignore_server_additions=true.
func TestAccRestApiObject_IgnoreServerAdditions_UserConfigChanges(t *testing.T) {
	apiServerObjects := map[string]map[string]interface{}{
		"user-change-1": {
			"id":   "user-change-1",
			"name": "Original",
			// Server-added fields
			"created_at": "2024-01-01T00:00:00Z",
			"metadata":   map[string]interface{}{"region": "us-south"},
		},
	}

	svr := fakeserver.NewFakeServer(8129, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Step 1: Create resource
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8129"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "user-change-1"
    name = "Original"
  })
  ignore_server_additions = true
}
`,
			},
			{
				// Step 2: User changes their config - verify plan shows changes (PlanOnly)
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8129"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "user-change-1"
    name = "User Updated"
  })
  ignore_server_additions = true
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// Step 3: Apply the user's change and verify it took effect
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8129"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "user-change-1"
    name = "User Updated"
  })
  ignore_server_additions = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.name", "User Updated"),
				),
			},
		},
	})
}

// TestAccRestApiObject_UnknownDataWithIgnoreServerAdditions tests issue #344 with
// ignore_server_additions enabled when plan.Data contains unknown values.
// This is the exact scenario that was failing in the SCC WP module.
func TestAccRestApiObject_UnknownDataWithIgnoreServerAdditions(t *testing.T) {
	// Pre-populate with server data that has extra fields
	apiServerObjects := map[string]map[string]interface{}{
		"unknown-ign-1": {
			"id":   "unknown-ign-1",
			"name": "Test",
			"parameters": map[string]interface{}{
				"enabled": true,
			},
			// Server-added fields
			"created_at": "2024-01-01T00:00:00Z",
			"metadata":   map[string]interface{}{"version": "1"},
		},
	}

	svr := fakeserver.NewFakeServer(8127, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				// Create with ignore_server_additions
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8127"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "unknown-ign-1"
    name = "Test"
    parameters = {
      enabled = true
    }
  })
  ignore_server_additions = true
}
`,
			},
			{
				// Change to use a computed value - this should NOT error
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8127"
}

locals {
  computed_name = "Updated-${timestamp()}"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id   = "unknown-ign-1"
    name = local.computed_name
    parameters = {
      enabled = true
    }
  })
  ignore_server_additions = true
}
`,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
