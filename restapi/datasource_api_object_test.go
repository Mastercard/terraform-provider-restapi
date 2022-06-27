package restapi

/*
  See:
  https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
  https://github.com/terraform-providers/terraform-provider-local/blob/master/local/resource_local_file_test.go
  https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_db_security_group_test.go
*/

import (
	"fmt"
	"os"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRestapiobject_Basic(t *testing.T) {
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8082, apiServerObjects, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8082")

	opt := &apiClientOpt{
		uri:                 "http://127.0.0.1:8082/",
		insecure:            false,
		username:            "",
		password:            "",
		headers:             make(map[string]string),
		timeout:             2,
		idAttribute:         "id",
		copyKeys:            make([]string, 0),
		writeReturnsObject:  false,
		createReturnsObject: false,
		debug:               debug,
	}
	client, err := NewAPIClient(opt)
	if err != nil {
		t.Fatal(err)
	}

	/* Send a simple object */
	client.sendRequest("POST", "/api/objects", `
    {
      "id": "1234",
      "first": "Foo",
      "last": "Bar",
      "data": {
        "identifier": "FooBar"
      }
    }
  `)
	client.sendRequest("POST", "/api/objects", `
    {
      "id": "4321",
      "first": "Foo",
      "last": "Baz",
      "data": {
        "identifier": "FooBaz"
      }
    }
  `)
	client.sendRequest("POST", "/api/objects", `
    {
      "id": "5678",
      "first": "Nested",
      "last": "Fields",
      "data": {
        "identifier": "NestedFields"
      }
    }
  `)

	/* Send a complex object that we will pretend is the results of a search
	client.send_request("POST", "/api/objects", `
	  {
	    "id": "people",
	    "results": {
	      "number": 2,
	      "list": [
	        { "id": "1234", "first": "Foo", "last": "Bar" },
	        { "id": "4321", "first": "Foo", "last": "Baz" }
	      ]
	    }
	  }
	`)
	*/

	resource.UnitTest(t, resource.TestCase{
		Providers: testAccProviders,
		PreCheck:  func() { svr.StartInBackground() },
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
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
				/* Similar to the first, but also with a query string */
				Config: fmt.Sprintf(`
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
				/* Perform a test that mimicks a search (this will exercise search_path and results_key */
				Config: fmt.Sprintf(`
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

	svr.Shutdown()
}
