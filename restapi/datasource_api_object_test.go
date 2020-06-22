package restapi

/*
  See:
  https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
  https://github.com/terraform-providers/terraform-provider-local/blob/master/local/resource_local_file_test.go
  https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_db_security_group_test.go
*/

import (
	"fmt"
	"github.com/Mastercard/terraform-provider-restapi/fakeserver"
	"github.com/hashicorp/terraform/helper/resource"
	"os"
	"testing"
)

func TestAccRestapiobject_Basic(t *testing.T) {
	debug := false
	api_server_objects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8082, api_server_objects, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8082")

	opt := &apiClientOpt{
		uri:                   "http://127.0.0.1:8082/",
		insecure:              false,
		username:              "",
		password:              "",
		headers:               make(map[string]string, 0),
		timeout:               2,
		id_attribute:          "id",
		copy_keys:             make([]string, 0),
		write_returns_object:  false,
		create_returns_object: false,
		debug:                 debug,
	}
	client, err := NewAPIClient(opt)
	if err != nil {
		t.Fatal(err)
	}

	/* Send a simple object */
	client.send_request("POST", "/api/objects", `
    {
      "id": "1234",
      "first": "Foo",
      "last": "Bar",
      "data": {
        "identifier": "FooBar"
      }
    }
  `)
	client.send_request("POST", "/api/objects", `
    {
      "id": "4321",
      "first": "Foo",
      "last": "Baz",
      "data": {
        "identifier": "FooBaz"
      }
    }
  `)
	client.send_request("POST", "/api/objects", `
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
			/* TODO: Fails with fakeserver because a request for /api/objects/people/4321 is unexpected (400 error)
			      Find a way to test this effectively
			   {
			     Config: fmt.Sprintf(`
			         data "restapi_object" "Baz" {
			            path = "/api/objects/people"
			            search_key = "last"
			            search_value = "Baz"
			            results_key = "results/list"
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
			*/
		},
	})

	svr.Shutdown()
}
