package restapi

/*
  See:
  https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
  https://github.com/terraform-providers/terraform-provider-local/blob/master/local/resource_local_file_test.go
  https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_db_security_group_test.go
*/

import (
  "os"
  "fmt"
  "testing"
  "github.com/hashicorp/terraform/helper/resource"
  "github.com/Mastercard/terraform-provider-restapi/fakeserver"
)

func TestAccRestapiobject_Basic(t *testing.T) {
  debug := false
  api_server_objects := make(map[string]map[string]interface{})

  svr := fakeserver.NewFakeServer(8082, api_server_objects, true, debug)
  os.Setenv("REST_API_URI", "http://127.0.0.1:8082")

  client, err := NewAPIClient("http://127.0.0.1:8082/", false, "", "", make(map[string]string, 0), 2, "id", make([]string, 0), false, false, debug)
  if err != nil { t.Fatal(err) }

  /* Send a simple object */
  client.send_request("POST", "/api/objects", `
    {
      "id": "1234",
      "first": "Foo",
      "last": "Bar"
    }
  `)
  client.send_request("POST", "/api/objects", `
    {
      "id": "4321",
      "first": "Foo",
      "last": "Baz"
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
    Providers:    testAccProviders,
    PreCheck:     func() { svr.StartInBackground() },
    Steps: []resource.TestStep{
      {
        Config: fmt.Sprintf(`
            data "restapi_object" "Foo" {
               path = "/api/objects"
               search_key = "first"
               search_value = "Foo"
               debug = %t
            }
          `, debug),
        Check: resource.ComposeTestCheckFunc(
          testAccCheckRestapiObjectExists("data.restapi_object.Foo", "1234", client),
          resource.TestCheckResourceAttr("data.restapi_object.Foo", "id", "1234"),
          resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_data.first", "Foo"),
          resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_data.last", "Bar"),
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
