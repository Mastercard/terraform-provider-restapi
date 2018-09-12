package restapi

import (
  "fmt"
  "strings"
  "github.com/hashicorp/terraform/helper/resource"
  "github.com/hashicorp/terraform/terraform"
)

func testAccCheckRestapiObjectExists(n string, id string, client *api_client) resource.TestCheckFunc {
  return func(s *terraform.State) error {
    rs, ok := s.RootModule().Resources[n]
    if !ok {
      keys := make([]string, 0, len(s.RootModule().Resources))
      for k := range s.RootModule().Resources {
        keys = append(keys, k)
      }
      return fmt.Errorf("RestAPI object not found in terraform state: %s. Found: %s", n, strings.Join(keys, ", "))
    }

    if rs.Primary.ID == "" {
      return fmt.Errorf("RestAPI object id not set in terraform")
    }

    /* Make a throw-away API object to read from the API */
    path := "/api/objects"
    obj, err := NewAPIObject (
      client,
      path + "/{id}",
      path,
      path + "/{id}",
      path + "/{id}",
      id,
      "{}",
      true,
    )
    if err != nil { return err }

    err = obj.read_object()
    if err != nil { return err }

    return nil
  }
}
