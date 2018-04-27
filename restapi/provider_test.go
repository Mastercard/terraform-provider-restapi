package restapi

import (
  "testing"
  "github.com/hashicorp/terraform/helper/schema"
  "github.com/hashicorp/terraform/terraform"
)

var testAccProvider terraform.ResourceProvider

func init() {
  testAccProvider = Provider().(terraform.ResourceProvider)
}

func TestProvider(t *testing.T) {
  if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
    t.Fatalf("err: %s", err)
  }
}

func TestProvider_impl(t *testing.T) {
  var _ terraform.ResourceProvider = Provider()
}
