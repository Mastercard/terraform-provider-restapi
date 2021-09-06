package restapi

import (
	"testing"

	"github.com/Mastercard/terraform-provider-restapi/fakeserver"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProvider terraform.ResourceProvider
var testAccProviders map[string]terraform.ResourceProvider

func init() {
	testAccProvider = Provider().(terraform.ResourceProvider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"restapi": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func TestResourceProvider_RequireBasic(t *testing.T) {
	rp := Provider()

	raw := map[string]interface{}{}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	/*
	   XXX: This is expected to work even though we are not
	        explicitly declaring the required url parameter since
	        the test suite is run with the ENV entry set.
	*/
	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("Provider failed with error: %s", err)
	}
}

func TestResourceProvider_Oauth(t *testing.T) {
	rp := Provider()

	raw := map[string]interface{}{
		"uri": "http://foo.bar/baz",
		"oauth_client_credentials": map[string]interface{}{
			"oauth_client_id": "test",
			"oauth_client_credentials": map[string]interface{}{
				"test": []string{
					"value1",
					"value2",
				},
			},
		},
	}

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	/*
	   XXX: This is expected to work even though we are not
	        explicitly declaring the required url parameter since
	        the test suite is run with the ENV entry set.
	*/
	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("Provider failed with error: %s", err)
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

	rawConfig, err := config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err != nil {
		t.Fatalf("Explicit provider configuration failed with error: %s", err)
	}

	/* Now test the inverse */
	rp = Provider()
	raw = map[string]interface{}{
		"uri":       "http://127.0.0.1:8085/",
		"test_path": "/api/apaththatdoesnotexist",
	}

	rawConfig, err = config.NewRawConfig(raw)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = rp.Configure(terraform.NewResourceConfig(rawConfig))
	if err == nil {
		t.Fatalf("Provider was expected to fail when visiting %v at %v but it did not!", raw["test_path"], raw["uri"])
	}

	svr.Shutdown()
}
