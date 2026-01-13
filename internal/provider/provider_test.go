package provider

import (
	"regexp"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"restapi": providerserver.NewProtocol6WithError(New("test")()),
}

func TestProvider_valid(t *testing.T) {
	var tests = map[string]string{
		"simple": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					id = "55555"
					first = "Foo"
					last = "Bar"
				})
			}`,

		"oath_all": `
			provider "restapi" {
               	uri = "http://localhost:8080/"

				oauth_client_credentials {
					oauth_client_id     = "myclientid"
					oauth_client_secret = "myclientsecret"
					oauth_token_endpoint = "http://localhost:8080/oauth/token"
					oauth_scopes        = ["scope1", "scope2"]
				}
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					id = "55555"
					first = "Foo"
					last = "Bar"
				})
			}`,

		"with_retries": `
			provider "restapi" {
               	uri = "http://localhost:8080/"

				retries {
					max_retries = 5
					min_wait    = 2
					max_wait    = 60
				}
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					id = "55555"
					first = "Foo"
					last = "Bar"
				})
			}`,

		"oauth_with_endpoint_params": `
			provider "restapi" {
               	uri = "http://localhost:8080/"

				oauth_client_credentials {
					oauth_client_id      = "myclientid"
					oauth_client_secret  = "myclientsecret"
					oauth_token_endpoint = "http://localhost:8080/oauth/token"
					oauth_scopes         = ["scope1", "scope2"]
					endpoint_params = {
						audience = "https://api.example.com"
						resource = "https://resource.example.com"
					}
				}
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					id = "55555"
					first = "Foo"
					last = "Bar"
				})
			}`,
	}

	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						PlanOnly:           true,
						Config:             config,
						ExpectNonEmptyPlan: true,
					},
				},
			})
		})
	}
}

func TestProvider_invalid(t *testing.T) {
	var tests = map[string]string{
		"missing_uri": `
			provider "restapi" {
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"invalid_uri": `
			provider "restapi" {
				uri = "::::invalid-uri"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"bad_types": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				insecure = "not-a-boolean"
				username = 12345
				password = true
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"oauth_missing_fields": `
			provider "restapi" {
			   	uri = "http://localhost:8080/"

				oauth_client_credentials {
					oauth_client_id     = "myclientid"
					# Missing client secret and token endpoint
				}
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"oauth_wrong_type": `
			provider "restapi" {
			   	uri = "http://localhost:8080/"

				oauth_client_credentials = "this-should-be-a-block-not-a-string"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"timeout_negative": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				timeout = -5
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"rate_limit_zero": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				rate_limit = 0
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"rate_limit_negative": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				rate_limit = -1.5
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"conflicting_auth_oauth_and_basic": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				username = "user"
				password = "pass"
				oauth_client_credentials {
					oauth_client_id      = "myclientid"
					oauth_client_secret  = "myclientsecret"
					oauth_token_endpoint = "http://localhost:8080/oauth/token"
				}
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"conflicting_auth_bearer_and_basic": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				username = "user"
				password = "pass"
				bearer_token = "mytoken"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"conflicting_cert_file_and_string": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				cert_file = "/path/to/cert.pem"
				cert_string = "-----BEGIN CERTIFICATE-----"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"conflicting_key_file_and_string": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				key_file = "/path/to/key.pem"
				key_string = "-----BEGIN PRIVATE KEY-----"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"conflicting_root_ca_file_and_string": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				root_ca_file = "/path/to/ca.pem"
				root_ca_string = "-----BEGIN CERTIFICATE-----"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"cert_without_key": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				cert_file = "/path/to/cert.pem"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"key_without_cert": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				key_file = "/path/to/key.pem"
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"retry_max_negative": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				retries {
					max_retries = -1
				}
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"retry_min_wait_negative": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				retries {
					max_retries = 3
					min_wait    = -5
					max_wait    = 30
				}
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"retry_max_wait_negative": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				retries {
					max_retries = 3
					min_wait    = 1
					max_wait    = -10
				}
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,

		"retry_min_greater_than_max": `
			provider "restapi" {
				uri = "http://localhost:8080/"
				retries {
					max_retries = 3
					min_wait    = 60
					max_wait    = 30
				}
			}
			data "restapi_object" "test" {
				path = "/api/test"
			}
		`,
	}

	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(".+"),
					},
				},
			})
		})
	}
}

// TestAccProvider_DependentURI tests issue #291: provider URI depending on resource outputs
// This simulates scenarios like azurerm_databricks_workspace where the URI is unknown during plan
func TestAccProvider_DependentURI(t *testing.T) {
	// Start fake server
	apiServerObjects := make(map[string]map[string]interface{})
	svr := fakeserver.NewFakeServer(8082, apiServerObjects, map[string]string{}, true, false, "")
	defer svr.Shutdown()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderDependentURIConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.dependent", "id", "dep1"),
					resource.TestCheckResourceAttr("restapi_object.dependent", "api_data.name", "Dependent Object"),
				),
			},
		},
	})
}

func testAccProviderDependentURIConfig() string {
	return `

# Bootstrap provider with known URI (used to create the api_endpoint resource)
provider "restapi" {
  alias = "bootstrap"
  uri = "http://127.0.0.1:8082"
  write_returns_object = true
}

# Simulate a resource that provides the API endpoint (like azurerm_databricks_workspace)
# In real usage, this would be a resource whose output is unknown during plan
resource "restapi_object" "api_endpoint" {
  path = "/api/objects"
  data = jsonencode({
    id = "config1"
    endpoint_host = "127.0.0.1:8082"
  })
  provider = restapi.bootstrap
}


# Main provider whose URI depends on the api_endpoint resource
# During plan phase, this URI is unknown, but during apply it becomes known
provider "restapi" {
  uri = "http://${restapi_object.api_endpoint.api_data["endpoint_host"]}"
  write_returns_object = true
}

# This resource uses the provider with dependent URI
# It should be plannable even though the URI is unknown during initial plan
resource "restapi_object" "dependent" {
  path = "/api/objects"
  data = jsonencode({
    id = "dep1"
    name = "Dependent Object"
    value = "Created with dynamically determined URI"
  })
}
`
}
