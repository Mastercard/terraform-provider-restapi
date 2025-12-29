package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSourceObject_valid(t *testing.T) {
	var tests = map[string]string{
		"simple": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_key = "name"
				search_value = "test"
			}`,

		"with_search_path": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_path = "/api/objects/search"
				search_key = "name"
				search_value = "test"
			}`,

		"with_query_strings": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_key = "name"
				search_value = "test"
				query_string = "?include=metadata"
				read_query_string = "?detailed=true"
			}`,

		"with_search_data": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_key = "name"
				search_value = "test"
				search_data = jsonencode({
					filter = "active"
					limit = 10
				})
			}`,

		"with_results_key": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_key = "name"
				search_value = "test"
				results_key = "data/items"
			}`,

		"with_overrides": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_key = "name"
				search_value = "test"
				id_attribute = "custom_id"
				debug = true
			}`,

		"complex_search": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_path = "/api/objects/search"
				search_key = "metadata/name"
				search_value = "test-object"
				results_key = "response/data/items"
				query_string = "?active=true"
				read_query_string = "?include=all"
				search_data = jsonencode({
					filters = {
						type = "production"
						status = "active"
					}
				})
			}`,

		"nested_search_key": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			data "restapi_object" "test" {
				path = "/api/objects"
				search_key = "attributes/identifier/id"
				search_value = "12345"
			}`,
	}

	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
					},
				},
			})
		})
	}
}

func TestDataSourceObject_invalid(t *testing.T) {
	type testCase struct {
		config      string
		expectError string
	}
	var tests = map[string]testCase{
		"missing_path": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					search_key = "name"
					search_value = "test"
				}
			`,
			expectError: `The argument "path" is required`,
		},

		"missing_search_key": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_value = "test"
				}
			`,
			expectError: `The argument "search_key" is required`,
		},

		"missing_search_value": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_key = "name"
				}
			`,
			expectError: `The argument "search_value" is required`,
		},

		"invalid_search_data_json": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_key = "name"
					search_value = "test"
					search_data = "not valid json"
				}
			`,
			expectError: `not valid JSON`,
		},

		"bad_type_debug": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_key = "name"
					search_value = "test"
					debug = "not-a-boolean"
				}
			`,
			expectError: `Inappropriate value for attribute "debug"`,
		},

		"unknown_attribute": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_key = "name"
					search_value = "test"
					unknown_field = "value"
				}
			`,
			expectError: `An argument named "unknown_field" is not expected here`,
		},

		"computed_attribute_set": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_key = "name"
					search_value = "test"
					api_data = {}
				}
			`,
			expectError: `(?i)(read-only|computed)`,
		},

		"api_response_set": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				data "restapi_object" "test" {
					path = "/api/objects"
					search_key = "name"
					search_value = "test"
					api_response = "some value"
				}
			`,
			expectError: `(?i)(read-only|computed)`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: regexp.MustCompile(tc.expectError),
					},
				},
			})
		})
	}
}
