package restapi

/*
  See:
  https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
  https://github.com/terraform-providers/terraform-provider-local/blob/master/local/resource_local_file_test.go
  https://github.com/terraform-providers/terraform-provider-aws/blob/master/aws/resource_aws_db_security_group_test.go
*/

/*
  "log"
  "github.com/hashicorp/terraform/config"
  "fmt"
*/
import (
  "os"
  "testing"
  "encoding/json"
  "github.com/hashicorp/terraform/helper/resource"
  "github.com/Mastercard/terraform-provider-restapi/fakeserver"
)

// example.Widget represents a concrete Go type that represents an API resource
func TestAccRestApiObject_Basic(t *testing.T) {
  debug := false
  api_server_objects := make(map[string]map[string]interface{})

  svr := fakeserver.NewFakeServer(8082, api_server_objects, true, debug)
  os.Setenv("REST_API_URI", "http://127.0.0.1:8082")

  client, err := NewAPIClient("http://127.0.0.1:8082/", false, "", "", make(map[string]string, 0), 2, "id", make([]string, 0), false, false, debug)
  if err != nil { t.Fatal(err) }

  resource.UnitTest(t, resource.TestCase{
    Providers:    testAccProviders,
    PreCheck:     func() { svr.StartInBackground() },
    Steps: []resource.TestStep{
      {
        Config: generate_test_resource(
          "Foo",
          `{ "id": "1234", "first": "Foo", "last": "Bar" }`,
          make(map[string]interface{}),
        ),
        Check: resource.ComposeTestCheckFunc(
          testAccCheckRestapiObjectExists("restapi_object.Foo", "1234", client),
          resource.TestCheckResourceAttr("restapi_object.Foo", "id", "1234"),
          resource.TestCheckResourceAttr("restapi_object.Foo", "api_data.first", "Foo"),
          resource.TestCheckResourceAttr("restapi_object.Foo", "api_data.last", "Bar"),
        ),
      },
      /* Make a complex object with id_attribute as a child of another key
         Note that we have to pass "id" just so fakeserver won't get angry at us
       */
      {
        Config: generate_test_resource(
          "Bar",
          `{ "id": "4321", "attributes": { "id": "4321" }, "config": { "first": "Bar", "last": "Baz" } }`,
          map[string]interface{}{
            "debug": debug,
            "id_attribute": "attributes/id",
          },
        ),
        Check: resource.ComposeTestCheckFunc(
          testAccCheckRestapiObjectExists("restapi_object.Bar", "4321", client),
          resource.TestCheckResourceAttr("restapi_object.Bar", "id", "4321"),
          resource.TestCheckResourceAttrSet("restapi_object.Bar", "api_data.config"),
        ),
      },
    },
  })

  svr.Shutdown()
}

/* This function generates a terraform JSON configuration from
   a name, JSON data and a list of params to set by coaxing it
   all to maps and then serializing to JSON */
func generate_test_resource(name string, data string, params map[string]interface{}) string {
  config := map[string]interface{}{
    "path": "/api/objects",
    "data": data,
  }

  for k, v := range params {
    config[k] = v
  }

  //What a mess...
  generated := map[string]interface{}{
    "resource": map[string]interface{}{
      "restapi_object": map[string]interface{}{
        name: config,
      },
    },
  }

  res, _ := json.Marshal(generated)
  return string(res)
}
