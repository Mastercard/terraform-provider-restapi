package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/Mastercard/terraform-provider-restapi/internal/apiclient"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestapiobject_Basic(t *testing.T) {
	ctx := context.Background()
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8116, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8116/",
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

	// Send a simple object
	client.SendRequest(ctx, "POST", "/api/objects", `
    {
      "id": "1234",
      "first": "Foo",
      "last": "Bar",
      "data": {
        "identifier": "FooBar"
      }
    }
  `, debug)
	client.SendRequest(ctx, "POST", "/api/objects", `
    {
      "id": "4321",
      "first": "Foo",
      "last": "Baz",
      "data": {
        "identifier": "FooBaz"
      }
    }
  `, debug)
	client.SendRequest(ctx, "POST", "/api/objects", `
    {
      "id": "5678",
      "first": "Nested",
      "last": "Fields",
      "data": {
        "identifier": "NestedFields"
      }
    }
  `, debug)

	// Send a complex object that we will pretend is the results of a search
	// client.send_request("POST", "/api/objects", `
	//   {
	//     "id": "people",
	//     "results": {
	//       "number": 2,
	//       "list": [
	//         { "id": "1234", "first": "Foo", "last": "Bar" },
	//         { "id": "4321", "first": "Foo", "last": "Baz" }
	//       ]
	//     }
	//   }
	// `)

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8116"
}

data "restapi_object" "Foo" {
  path = "/api/objects"
  search_key = "last"
  search_value = "Bar"
  debug = %t
}
          `, debug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("data.restapi_object.Foo", "1234", client),
					resource.TestCheckResourceAttr("data.restapi_object.Foo", "id", "1234"),
					resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_data.first", "Foo"),
					resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_data.last", "Bar"),
					resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_response", "{\"data\":{\"identifier\":\"FooBar\"},\"first\":\"Foo\",\"id\":\"1234\",\"last\":\"Bar\"}"),
				),
				// PreventDiskCleanup: true,
			},
			{
				Config: fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8116"
}

data "restapi_object" "Nested" {
  path = "/api/objects"
  search_key = "data/identifier"
  search_value = "NestedFields"
  debug = %t
}
          `, debug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("data.restapi_object.Nested", "5678", client),
					resource.TestCheckResourceAttr("data.restapi_object.Nested", "id", "5678"),
					resource.TestCheckResourceAttr("data.restapi_object.Nested", "api_data.first", "Nested"),
					resource.TestCheckResourceAttr("data.restapi_object.Nested", "api_data.last", "Fields"),
				),
			},
			{
				// Similar to the first, but also with a query string
				Config: fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8116"
}

data "restapi_object" "Baz" {
  path = "/api/objects"
  query_string = "someArg=foo&anotherArg=bar"
  search_key = "last"
  search_value = "Baz"
  debug = %t
}
          `, debug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("data.restapi_object.Baz", "4321", client),
					resource.TestCheckResourceAttr("data.restapi_object.Baz", "id", "4321"),
					resource.TestCheckResourceAttr("data.restapi_object.Baz", "api_data.first", "Foo"),
					resource.TestCheckResourceAttr("data.restapi_object.Baz", "api_data.last", "Baz"),
				),
			},
			{
				// Perform a test that mimicks a search (this will exercise search_path and results_key
				Config: fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8116"
}

data "restapi_object" "Baz" {
  path = "/api/objects"
  search_path = "/api/object_list"
  search_key = "last"
  search_value = "Baz"
  results_key = "list"
  debug = %t
}
		       `, debug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("data.restapi_object.Baz", "4321", client),
					resource.TestCheckResourceAttr("data.restapi_object.Baz", "id", "4321"),
					resource.TestCheckResourceAttr("data.restapi_object.Baz", "api_data.first", "Foo"),
					resource.TestCheckResourceAttr("data.restapi_object.Baz", "api_data.last", "Baz"),
				),
			},
		},
	})
}

// TestAccRestapiObjectDataSource_ResultsContainsObject tests the results_contains_object parameter
// which allows using search results directly without a second GET request
func TestAccRestapiObjectDataSource_ResultsContainsObject(t *testing.T) {
	debug := false

	// Pre-populate the objects before creating the server
	apiServerObjects := map[string]map[string]interface{}{
		"user1": {
			"id":        "user1",
			"username":  "john_doe",
			"email":     "john@example.com",
			"full_name": "John Doe",
		},
		"user2": {
			"id":        "user2",
			"username":  "jane_smith",
			"email":     "jane@example.com",
			"full_name": "Jane Smith",
		},
	}

	svr := fakeserver.NewFakeServer(8117, apiServerObjects, map[string]string{}, true, debug, "")
	defer svr.Shutdown()

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8117",
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

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		PreCheck:                 func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8117"
}

data "restapi_object" "user_search" {
  path = "/api/objects"
  search_key = "username"
  search_value = "john_doe"
  results_contains_object = true
  debug = %t
}
          `, debug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("data.restapi_object.user_search", "user1", client),
					resource.TestCheckResourceAttr("data.restapi_object.user_search", "id", "user1"),
					resource.TestCheckResourceAttr("data.restapi_object.user_search", "api_data.username", "john_doe"),
					resource.TestCheckResourceAttr("data.restapi_object.user_search", "api_data.email", "john@example.com"),
					resource.TestCheckResourceAttr("data.restapi_object.user_search", "api_data.full_name", "John Doe"),
				),
			},
			{
				// Test with results_key AND results_contains_object
				Config: fmt.Sprintf(`
provider "restapi" {
  uri = "http://127.0.0.1:8117"
}

data "restapi_object" "user_with_results_key" {
  path = "/api/objects"
  search_path = "/api/object_list"
  search_key = "username"
  search_value = "jane_smith"
  results_key = "list"
  results_contains_object = true
  debug = %t
}
          `, debug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRestapiObjectExists("data.restapi_object.user_with_results_key", "user2", client),
					resource.TestCheckResourceAttr("data.restapi_object.user_with_results_key", "id", "user2"),
					resource.TestCheckResourceAttr("data.restapi_object.user_with_results_key", "api_data.username", "jane_smith"),
					resource.TestCheckResourceAttr("data.restapi_object.user_with_results_key", "api_data.full_name", "Jane Smith"),
				),
			},
		},
	})
}
