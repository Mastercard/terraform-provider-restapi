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
	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRestApiObject_Basic(t *testing.T) {
	debug := false

	if debug {
		os.Setenv("TF_LOG", "DEBUG")
	}
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8082, apiServerObjects, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8082")

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8082/",
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
resource "restapi_object" "%s" {
%s
}
`, name, strConfig)
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

	host := "127.0.0.1:8082"
	returnCodes := map[string]int{
		"PUT /api/objects/1234": http.StatusBadRequest,
	}
	responses := map[string]string{
		"GET /api/objects/1234": `{ "id": "1234", "foo": "Bar" }`,
	}
	srv := mockServer(host, returnCodes, responses)
	defer srv.Close()

	os.Setenv("REST_API_URI", "http://"+host)

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create the resource
				Config: generateTestResource(
					"Foo",
					`{ "id": "1234", "foo": "Bar" }`,
					make(map[string]interface{}),
					debug,
				),
				Check: resource.TestCheckResourceAttr("restapi_object.Foo", "data", `{ "id": "1234", "foo": "Bar" }`),
			},
			{
				// Try update. It will fail becuase we return 400 for PUT operations from mock server
				Config: generateTestResource(
					"Foo",
					`{ "id": "1234", "foo": "Updated" }`,
					make(map[string]interface{}),
					debug,
				),
				ExpectError: regexp.MustCompile("unexpected response code '400'"),
			},
			{
				// Expecting plan to be non-empty because the failed apply above shouldn't update terraform state
				Config: generateTestResource(
					"Foo",
					`{ "id": "1234", "foo": "Updated" }`,
					make(map[string]interface{}),
					debug,
				),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
