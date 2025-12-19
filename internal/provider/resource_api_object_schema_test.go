package restapi

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestResourceObject_valid(t *testing.T) {
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

		"complex": `
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
				create_method = "PUT"
				read_method = "GET"
				destroy_data = jsonencode({
					soft_delete = true
				})
				force_new = [
					"first",
					"last"
				]
			}`,

		"all_paths": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				create_path = "/api/objects"
				read_path = "/api/objects/{id}"
				update_path = "/api/objects/{id}"
				destroy_path = "/api/objects/{id}"
				data = jsonencode({
					name = "test"
				})
			}`,

		"all_methods": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				create_method = "POST"
				read_method = "GET"
				update_method = "PATCH"
				destroy_method = "DELETE"
				data = jsonencode({
					name = "test"
				})
			}`,

		"with_all_data_fields": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					name = "test"
				})
				read_data = jsonencode({
					include_metadata = true
				})
				update_data = jsonencode({
					update_metadata = true
				})
				destroy_data = jsonencode({
					soft_delete = true
				})
			}`,

		"with_ignore_options": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					name = "test"
				})
				ignore_changes_to = [
					"last_modified",
					"metadata.timestamp"
				]
				ignore_all_server_changes = false
			}`,

		"with_search_options": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					name = "test"
				})
				query_string = "?include=metadata"
				read_search = "custom_search_value"
			}`,

		"with_overrides": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					name = "test"
				})
				id_attribute = "custom_id"
				object_id = "explicit-id-123"
				debug = true
			}`,

		"empty_json_object": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({})
			}`,

		"nested_json": `
			provider "restapi" {
               	uri = "http://localhost:8080/"
			}
			resource "restapi_object" "test" {
				path = "/api/objects"
				data = jsonencode({
					name = "test"
					metadata = {
						created_by = "terraform"
						tags = ["test", "example"]
					}
					config = {
						settings = {
							option1 = true
							option2 = "value"
						}
					}
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

func TestResourceObject_invalid(t *testing.T) {
	type testCase struct {
		config      string
		expectError string
	}
	var tests = map[string]testCase{
		"missing_data": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
				}
			`,
			expectError: `The argument "data" is required`,
		},

		"missing_path": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					data = jsonencode({
						id = "55555"
						first = "Foo"
						last = "Bar"
					})
				}
			`,
			expectError: `The argument "path" is required`,
		},

		"bad_data": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = "Not a JSON object"
				}
			`,
			expectError: `not valid JSON`,
		},

		"invalid_read_data_json": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					read_data = "{invalid json}"
				}
			`,
			expectError: `not valid JSON`,
		},

		"invalid_update_data_json": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					update_data = "not json at all"
				}
			`,
			expectError: `not valid JSON`,
		},

		"invalid_destroy_data_json": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					destroy_data = "{broken: json}"
				}
			`,
			expectError: `not valid JSON`,
		},

		"bad_type_debug": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					debug = "not-a-boolean"
				}
			`,
			expectError: `Inappropriate value for attribute "debug"`,
		},

		"bad_type_force_new": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					force_new = "should-be-a-list"
				}
			`,
			expectError: `Inappropriate value for attribute "force_new"`,
		},

		"bad_type_ignore_changes": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					ignore_changes_to = "should-be-a-list"
				}
			`,
			expectError: `Inappropriate value for attribute "ignore_changes_to"`,
		},

		"unknown_attribute": {
			config: `
				provider "restapi" {
					uri = "http://localhost:8080/"
				}
				resource "restapi_object" "test" {
					path = "/api/objects"
					data = jsonencode({id = "123"})
					unknown_field = "value"
				}
			`,
			expectError: `An argument named "unknown_field" is not expected here`,
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
