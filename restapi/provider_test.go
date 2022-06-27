package restapi

import (
	"context"
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccProvider *schema.Provider
var testAccProviders map[string]*schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"restapi": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func TestResourceProvider_RequireBasic(t *testing.T) {
	rp := Provider()
	raw := map[string]interface{}{}

	/*
	   XXX: This is expected to work even though we are not
	        explicitly declaring the required url parameter since
	        the test suite is run with the ENV entry set.
	*/
	err := rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
	if err != nil {
		t.Fatalf("Provider failed with error: %v", err)
	}
}

func TestResourceProvider_Oauth(t *testing.T) {
	rp := Provider()
	raw := map[string]interface{}{
		"uri": "http://foo.bar/baz",
		"oauth_client_credentials": map[string]interface{}{
			"oauth_client_id": "test",
			/*
				Commented out 2022-06-27. Although terraform allows the provider to define this as
				array of strings, it panics during unmarshal on the terraform provider SDK
						"oauth_client_credentials": map[string]interface{}{
							"test": []string{
								"value1",
								"value2",
							},
						},
			*/
		},
	}

	/*
	   XXX: This is expected to work even though we are not
	        explicitly declaring the required url parameter since
	        the test suite is run with the ENV entry set.
	*/
	err := rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
	if err != nil {
		t.Fatalf("Provider failed with error: %v", err)
	}
}

func TestResourceProvider_RequireTestPath(t *testing.T) {
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8085, apiServerObjects, true, debug, "")
	svr.StartInBackground()

	rp := Provider()
	raw := map[string]interface{}{
		"uri":       "http://127.0.0.1:8085/",
		"test_path": "/api/objects",
	}

	err := rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))

	/* Now test the inverse */
	rp = Provider()
	raw = map[string]interface{}{
		"uri":       "http://127.0.0.1:8085/",
		"test_path": "/api/apaththatdoesnotexist",
	}

	err = rp.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
	if err == nil {
		t.Fatalf("Provider was expected to fail when visiting %v at %v but it did not!", raw["test_path"], raw["uri"])
	}

	svr.Shutdown()
}
