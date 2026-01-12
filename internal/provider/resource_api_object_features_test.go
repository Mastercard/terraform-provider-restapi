package provider

import (
	"os"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	apiclient "github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRestApiObject_IgnoreChangesTo tests the ignore_changes_to feature
func TestAccRestApiObject_IgnoreChangesTo(t *testing.T) {
	debug := false

	apiServerObjects := map[string]map[string]interface{}{
		"ignore1": {
			"id":        "ignore1",
			"name":      "Test",
			"timestamp": "2026-01-12T10:00:00Z",
		},
	}

	svr := fakeserver.NewFakeServer(8103, apiServerObjects, map[string]string{}, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8103")

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: `
resource "restapi_object" "Test" {
  path = "/api/objects"
  data = "{ \"id\": \"ignore1\", \"name\": \"Test\", \"timestamp\": \"2026-01-01T00:00:00Z\" }"
  ignore_changes_to = ["timestamp"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.Test", "id", "ignore1"),
					resource.TestCheckResourceAttr("restapi_object.Test", "api_data.name", "Test"),
				),
			},
		},
	})
}

// TestAccRestApiObject_IgnoreAllServerChanges tests the ignore_all_server_changes feature
func TestAccRestApiObject_IgnoreAllServerChanges(t *testing.T) {
	debug := false

	apiServerObjects := map[string]map[string]interface{}{
		"ignore2": {
			"id":   "ignore2",
			"name": "ServerModified",
		},
	}

	svr := fakeserver.NewFakeServer(8104, apiServerObjects, map[string]string{}, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8104")

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: `
resource "restapi_object" "Test" {
  path = "/api/objects"
  data = "{ \"id\": \"ignore2\", \"name\": \"Original\" }"
  ignore_all_server_changes = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.Test", "id", "ignore2"),
					// With ignore_all_server_changes, state should show our data
					resource.TestCheckResourceAttr("restapi_object.Test", "api_data.name", "Original"),
				),
			},
		},
	})
}

// TestAccRestApiObject_ForceNew tests that changing non-force_new fields doesn't trigger replacement
func TestAccRestApiObject_ForceNew(t *testing.T) {
	debug := false

	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8105, apiServerObjects, map[string]string{}, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8105")

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8105/",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IDAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  true,
		CreateReturnsObject: true,
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
				Config: `
resource "restapi_object" "Test" {
  path = "/api/objects"
  data = "{ \"id\": \"forcenew1\", \"type\": \"A\", \"name\": \"Test\" }"
  force_new = ["type"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Test", "forcenew1", client),
					resource.TestCheckResourceAttr("restapi_object.Test", "id", "forcenew1"),
					resource.TestCheckResourceAttr("restapi_object.Test", "api_data.type", "A"),
				),
			},
			{
				// Changing name (not in force_new) should update in-place
				Config: `
resource "restapi_object" "Test" {
  path = "/api/objects"
  data = "{ \"id\": \"forcenew1\", \"type\": \"A\", \"name\": \"Updated\" }"
  force_new = ["type"]
}
`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Test", "forcenew1", client),
					resource.TestCheckResourceAttr("restapi_object.Test", "id", "forcenew1"),
					resource.TestCheckResourceAttr("restapi_object.Test", "api_data.name", "Updated"),
				),
			},
		},
	})
}

// TestAccRestApiObject_DestroyData tests the destroy_data feature
func TestAccRestApiObject_DestroyData(t *testing.T) {
	debug := false

	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8107, apiServerObjects, map[string]string{}, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8107")

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8107/",
		Timeout:             2,
		WriteReturnsObject:  true,
		CreateReturnsObject: true,
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
				Config: `
resource "restapi_object" "Test" {
  path = "/api/objects"
  data = "{ \"id\": \"destroy1\", \"name\": \"Test\" }"
  destroy_data = "{ \"reason\": \"test cleanup\" }"
}
`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Test", "destroy1", client),
				),
			},
		},
	})
}

// TestAccRestApiObject_QueryString tests query string support
func TestAccRestApiObject_QueryString(t *testing.T) {
	debug := false

	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8108, apiServerObjects, map[string]string{}, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8108")

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8108/",
		Timeout:             2,
		WriteReturnsObject:  true,
		CreateReturnsObject: true,
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
				Config: `
resource "restapi_object" "Test" {
  path = "/api/objects"
  data = "{ \"id\": \"query1\", \"name\": \"Test\" }"
  query_string = "version=2&force=true"
}
`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.Test", "query1", client),
					resource.TestCheckResourceAttr("restapi_object.Test", "id", "query1"),
				),
			},
		},
	})
}
