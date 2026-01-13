package provider

import (
	"regexp"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccProvider_TestPath tests the test_path configuration feature.
// This test verifies that the provider validates connectivity to the API
// by making a test request to the specified path during configuration.
// This test requires a running fake server, so it cannot be a simple unit test.
func TestAccProvider_TestPath(t *testing.T) {
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	// Start fake server on port 8121
	svr := fakeserver.NewFakeServer(8121, apiServerObjects, map[string]string{}, true, debug, "")
	svr.StartInBackground()
	defer svr.Shutdown()

	t.Run("success_path_exists", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: `
						provider "restapi" {
							uri       = "http://127.0.0.1:8121/"
							test_path = "/api/objects"
						}
						
						resource "restapi_object" "test" {
							path = "/api/objects"
							data = jsonencode({
								id = "99998"
								name = "test"
							})
						}
					`,
					// The test_path should succeed since /api/objects exists
					// The resource should be created successfully
				},
			},
		})
	})

	t.Run("failure_path_not_exists", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: `
						provider "restapi" {
							uri       = "http://127.0.0.1:8121/"
							test_path = "/api/nonexistent"
						}
						
						resource "restapi_object" "test" {
							path = "/api/objects"
							data = jsonencode({
								id = "99999"
								name = "test"
							})
						}
					`,
					// The provider should fail to configure because test_path doesn't exist
					ExpectError: regexp.MustCompile("(?i)(test request failed|unexpected response)"),
				},
			},
		})
	})
}
