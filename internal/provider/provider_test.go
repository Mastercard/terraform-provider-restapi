package restapi

import (
	"regexp"
	"testing"

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

/*

func TestResourceProvider_Oauth(t *testing.T) {
	rp := Provider()
	raw := map[string]interface{}{
		"uri": "http://foo.bar/baz",
		"oauth_client_credentials": map[string]interface{}{
			"oauth_client_id": "test",
			"oauth_client_credentials": map[string]interface{}{
				"audience": "coolAPI",
			},
		},
	}


	//   XXX: This is expected to work even though we are not
	//        explicitly declaring the required url parameter since
	//        the test suite is run with the ENV entry set.

	err := rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
	if err != nil {
		t.Fatalf("Provider failed with error: %v", err)
	}
}

func TestResourceProvider_RequireTestPath(t *testing.T) {
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8085, apiServerObjects, true, debug, "")
	svr.StartInBackground()

	rp := Provider()
	raw := map[string]interface{}{
		"uri":       "http://127.0.0.1:8085/",
		"test_path": "/api/objects",
	}

	err := rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
	if err != nil {
		t.Fatalf("Provider config failed when visiting %v at %v but it did not!", raw["test_path"], raw["uri"])
	}

	rp = Provider()
	raw = map[string]interface{}{
		"uri":       "http://127.0.0.1:8085/",
		"test_path": "/api/apaththatdoesnotexist",
	}

	err = rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
	if err == nil {
		t.Fatalf("Provider was expected to fail when visiting %v at %v but it did not!", raw["test_path"], raw["uri"])
	}

	svr.Shutdown()
}
*/
