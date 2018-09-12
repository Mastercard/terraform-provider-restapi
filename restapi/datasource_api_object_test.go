package restapi

/*
  See:
  https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
  https://github.com/terraform-providers/terraform-provider-local/blob/master/local/resource_local_file_test.go
  https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_db_security_group_test.go
*/

import (
  "os"
  "testing"
  "github.com/hashicorp/terraform/helper/resource"
  "github.com/Mastercard/terraform-provider-restapi/fakeserver"
)

func TestAccRestapiobject_Basic(t *testing.T) {
  debug := false
  api_server_objects := make(map[string]map[string]interface{})

  svr := fakeserver.NewFakeServer(8082, api_server_objects, true, debug)
  if os.Getenv("REST_API_URI") != "http://127.0.0.1:8082" {
    t.Fatalf("REST_API_URI environment variable must be set to 'http://127.0.0.1:8082' but it is set to '%s'", os.Getenv("REST_API_URI"))
  }

  client, err := NewAPIClient("http://127.0.0.1:8082/", false, "", "", make(map[string]string, 0), 2, "id", make([]string, 0), false, false, debug)
  if err != nil { t.Fatal(err) }
  client.send_request("POST", "/api/objects", `{ "id": "1234", "first": "Foo", "last": "Bar" }`)

  resource.UnitTest(t, resource.TestCase{
    Providers:    testAccProviders,
    PreCheck:     func() { svr.StartInBackground() },
    Steps: []resource.TestStep{
      {
        Config: `
            data "restapi_object" "Foo" {
               path = "/api/objects"
               search_key = "first"
               search_value = "Foo"
            }
          `,
        Check: resource.ComposeTestCheckFunc(
          testAccCheckRestapiObjectExists("data.restapi_object.Foo", "1234", client),
          resource.TestCheckResourceAttr("data.restapi_object.Foo", "id", "1234"),
          resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_data.first", "Foo"),
          resource.TestCheckResourceAttr("data.restapi_object.Foo", "api_data.last", "Bar"),
        ),
      },
    },
  })

  svr.Shutdown()
}
