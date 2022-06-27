package restapi

import (
	"os"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRestApiObject_importBasic(t *testing.T) {
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8082, apiServerObjects, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8082")

	opt := &apiClientOpt{
		uri:                 "http://127.0.0.1:8082/",
		insecure:            false,
		username:            "",
		password:            "",
		headers:             make(map[string]string),
		timeout:             2,
		idAttribute:         "id",
		copyKeys:            make([]string, 0),
		writeReturnsObject:  false,
		createReturnsObject: false,
		debug:               debug,
	}
	client, err := NewAPIClient(opt)
	if err != nil {
		t.Fatal(err)
	}
	client.sendRequest("POST", "/api/objects", `{ "id": "1234", "first": "Foo", "last": "Bar" }`)

	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		PreCheck:  func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: generateTestResource(
					"Foo",
					`{ "id": "1234", "first": "Foo", "last": "Bar" }`,
					make(map[string]interface{}),
				),
			},
			{
				ResourceName:        "restapi_object.Foo",
				ImportState:         true,
				ImportStateId:       "1234",
				ImportStateIdPrefix: "/api/objects/",
				ImportStateVerify:   true,
				/* create_response isn't populated during import (we don't know the API response from creation) */
				ImportStateVerifyIgnore: []string{"debug", "data", "create_response"},
			},
		},
	})

	svr.Shutdown()
}
