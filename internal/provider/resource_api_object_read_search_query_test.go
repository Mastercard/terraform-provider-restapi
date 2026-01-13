package provider

import (
	"fmt"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRestApiObject_ReadSearchQueryString tests that query_string within read_search
// is properly used when performing searches
func TestAccRestApiObject_ReadSearchQueryString(t *testing.T) {
	debug := false

	// Set up initial objects
	apiServerObjects := map[string]map[string]interface{}{
		"1234": {
			"id":     "1234",
			"name":   "test-object",
			"status": "active",
		},
	}

	svr := fakeserver.NewFakeServer(8114, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8114/",
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
				Config: testAccRestApiObjectConfig_ReadSearchQueryString("1234", "test-object", "active"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.test", "1234", client),
					resource.TestCheckResourceAttr("restapi_object.test", "id", "1234"),
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.name", "test-object"),
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.status", "active"),
				),
			},
		},
	})
}

// TestAccRestApiObject_ReadSearchQueryStringWithObjectLevel tests that when both
// read_search.query_string and object-level query_string are set, they merge correctly
func TestAccRestApiObject_ReadSearchQueryStringWithObjectLevel(t *testing.T) {
	debug := false

	// Set up initial objects
	apiServerObjects := map[string]map[string]interface{}{
		"1234": {
			"id":     "1234",
			"name":   "test-object",
			"status": "active",
		},
	}

	svr := fakeserver.NewFakeServer(8115, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8115/",
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
				Config: testAccRestApiObjectConfig_ReadSearchQueryStringBoth("1234", "test-object", "active"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("restapi_object.test", "1234", client),
					resource.TestCheckResourceAttr("restapi_object.test", "id", "1234"),
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.name", "test-object"),
					resource.TestCheckResourceAttr("restapi_object.test", "api_data.status", "active"),
				),
			},
		},
	})
}

func testAccRestApiObjectConfig_ReadSearchQueryString(id, name, status string) string {
	return fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8114/"
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({
    id     = "%s"
    name   = "%s"
    status = "%s"
  })
  id_attribute = "id"
  
  read_search = {
    search_key   = "name"
    search_value = "%s"
    query_string = "include_metadata=true&status=%s"
  }
}
`, id, name, status, name, status)
}

func testAccRestApiObjectConfig_ReadSearchQueryStringBoth(id, name, status string) string {
	return fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8115/"
}

resource "restapi_object" "test" {
  path         = "/api/objects"
  query_string = "version=v1"
  
  data = jsonencode({
    id     = "%s"
    name   = "%s"
    status = "%s"
  })
  id_attribute = "id"
  
  read_search = {
    search_key   = "name"
    search_value = "%s"
    query_string = "include_metadata=true&status=%s"
  }
}
`, id, name, status, name, status)
}
