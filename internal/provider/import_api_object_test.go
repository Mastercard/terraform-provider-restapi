package provider

import (
	"context"
	"os"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestApiObject_importBasic(t *testing.T) {
	ctx := context.Background()
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8082, apiServerObjects, map[string]string{}, true, debug, "")
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
	client.SendRequest(ctx, "POST", "/api/objects", `{ "id": "1234", "first": "Foo", "last": "Bar" }`, debug)

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
			},
			{
				ResourceName:        "restapi_object.Foo",
			ImportState:         true,
			ImportStateId:       "1234",
			ImportStateIdPrefix: "/api/objects/",
			ImportStateVerify:   true,
			// create_response isn't populated during import (we don't know the API response from creation)
			// object_id is set during import but not during normal resource creation
			ImportStateVerifyIgnore: []string{"debug", "data", "create_response", "ignore_all_server_changes", "ignore_server_additions", "object_id"},
		},
		},
	})

	svr.Shutdown()
}
