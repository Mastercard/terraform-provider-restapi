package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRestApiObject_ReadSearchQueryString tests that query_string within read_search
// is properly used when performing searches
func TestAccRestApiObject_ReadSearchQueryString(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRestApiObjectConfig_ReadSearchQueryString("1234", "test-object", "active"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.test", "id", "/api/objects/1234"),
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
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRestApiObjectConfig_ReadSearchQueryStringBoth("1234", "test-object", "active"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.test", "id", "/api/objects/1234"),
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
  uri = "http://127.0.0.1:8080/"
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
    results_key  = "data"
    query_string = "include_metadata=true&status=%s"
  }
}
`, id, name, status, name, status)
}

func testAccRestApiObjectConfig_ReadSearchQueryStringBoth(id, name, status string) string {
	return fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8080/"
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
    results_key  = "data"
    query_string = "include_metadata=true&status=%s"
  }
}
`, id, name, status, name, status)
}
