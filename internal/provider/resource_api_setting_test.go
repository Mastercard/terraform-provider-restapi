package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRestApiSetting_BasicLifecycle(t *testing.T) {
	apiServerObjects := map[string]map[string]interface{}{
		"resource": {
			"first": "Foo",
			"last":  "Bar",
		},
	}

	svr := fakeserver.NewFakeServer(8200, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: generateTestSettingResource(
					"basic",
					`{ "first": "Updated", "last": "Value" }`,
					nil,
					false,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_setting.basic", "api_data.first", "Updated"),
					resource.TestCheckResourceAttr("restapi_setting.basic", "api_data.last", "Value"),
					resource.TestCheckResourceAttr("restapi_setting.basic", "api_response", "{\"first\":\"Updated\",\"last\":\"Value\"}"),
					resource.TestCheckResourceAttr("restapi_setting.basic", "initial_response", "{\"first\":\"Foo\",\"last\":\"Bar\"}"),
				),
			},
			{
				Config: generateTestSettingResource(
					"basic",
					`{ "first": "Second", "last": "Update" }`,
					nil,
					false,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_setting.basic", "api_data.first", "Second"),
					resource.TestCheckResourceAttr("restapi_setting.basic", "api_data.last", "Update"),
					resource.TestCheckResourceAttr("restapi_setting.basic", "initial_response", "{\"first\":\"Foo\",\"last\":\"Bar\"}"),
				),
			},
		},
	})
}

func TestAccRestApiSetting_FailedUpdate(t *testing.T) {
	host := "127.0.0.1:8210"
	returnCodes := map[string]int{}
	responses := map[string]string{
		"GET /api/objects/resource": `{ "first": "Initial", "last": "Value" }`,
	}
	srv := mockServerSetting(host, returnCodes, responses)
	defer srv.Close()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: generateTestSettingResourceWithURI(
					"failed_update",
					`{ "first": "Initial", "last": "Value" }`,
					nil,
					false,
					"http://"+host,
				),
			},
			{
				PreConfig: func() {
					returnCodes["PUT /api/objects/resource"] = http.StatusBadRequest
				},
				Config: generateTestSettingResourceWithURI(
					"failed_update",
					`{ "first": "Changed", "last": "Value" }`,
					nil,
					false,
					"http://"+host,
				),
				ExpectError: regexp.MustCompile("unexpected response code '400'"),
			},
			{
				PreConfig: func() {
					delete(returnCodes, "PUT /api/objects/resource")
				},
				Config: generateTestSettingResourceWithURI(
					"failed_update",
					`{ "first": "Changed", "last": "Value" }`,
					nil,
					false,
					"http://"+host,
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccRestApiSetting_PlanAfterApply_IgnoreServerAdditions(t *testing.T) {
	apiServerObjects := map[string]map[string]interface{}{
		"ign-add-1": {
			"id":   "ign-add-1",
			"name": "Test Object",
			"parameters": map[string]interface{}{
				"enable_cspm": true,
			},
			"created_at": "2024-01-01T00:00:00Z",
			"metadata":   map[string]interface{}{"region": "us-south"},
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
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8124"
}

resource "restapi_setting" "test" {
  read_path = "/api/objects/ign-add-1"
  update_path = "/api/objects/ign-add-1"
  data = jsonencode({
    name = "Test Object"
    parameters = {
      enable_cspm = true
    }
  })
  ignore_server_additions = true
}
`,
			},
			{
				Config: `
provider "restapi" {
  uri = "http://127.0.0.1:8124"
}

resource "restapi_setting" "test" {
  read_path = "/api/objects/ign-add-1"
  update_path = "/api/objects/ign-add-1"
  data = jsonencode({
    name = "Test Object"
    parameters = {
      enable_cspm = true
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

func TestAccRestApiSetting_DeleteRestoresInitialManagedKeys(t *testing.T) {
	apiServerObjects := map[string]map[string]interface{}{
		"resource": {
			"first":     "Initial",
			"last":      "State",
			"unmanaged": "keep-me",
		},
	}

	svr := fakeserver.NewFakeServer(8230, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	checkDestroy := func(_ *terraform.State) error {
		obj, ok := apiServerObjects["resource"]
		if !ok {
			return fmt.Errorf("expected setting object to still exist after destroy restore")
		}
		if obj["first"] != "Initial" || obj["last"] != "State" {
			return fmt.Errorf("expected managed keys to be restored to initial values, got: %#v", obj)
		}
		if _, exists := obj["unmanaged"]; exists {
			return fmt.Errorf("expected unmanaged key to remain untouched by restore payload filtering")
		}
		return nil
	}

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		CheckDestroy:             checkDestroy,
		Steps: []resource.TestStep{
			{
				Config: generateTestSettingResourceWithURI(
					"restore",
					`{ "first": "Managed", "last": "Changed" }`,
					nil,
					false,
					"http://127.0.0.1:8230",
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_setting.restore", "initial_response", "{\"first\":\"Initial\",\"last\":\"State\",\"unmanaged\":\"keep-me\"}"),
				),
			},
			{
				Config: generateTestSettingResourceWithURI(
					"restore",
					`{ "first": "Managed", "last": "Changed" }`,
					nil,
					false,
					"http://127.0.0.1:8230",
				),
				PlanOnly: true,
			},
		},
	})
}

// This function generates a terraform configuration from
// a name, JSON data and a list of params to set.
func generateTestSettingResource(name string, data string, params map[string]interface{}, debug bool) string {
	return generateTestSettingResourceWithURI(name, data, params, debug, "http://127.0.0.1:8200")
}

func generateTestSettingResourceWithURI(name string, data string, params map[string]interface{}, debug bool, uri string) string {
	strData, _ := json.Marshal(data)
	config := []string{
		`read_path = "/api/objects/resource"`,
		`update_path = "/api/objects/resource"`,
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
		strConfig += v + "\n"
	}

	return fmt.Sprintf(`
provider "restapi" {
  uri = "%s"
}

resource "restapi_setting" "%s" {
%s
}
`, uri, name, strConfig)
}

func mockServerSetting(host string, returnCodes map[string]int, responses map[string]string) *http.Server {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/api/", func(w http.ResponseWriter, req *http.Request) {
		key := fmt.Sprintf("%s %s", req.Method, req.RequestURI)
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
