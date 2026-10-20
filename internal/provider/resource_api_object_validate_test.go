package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonDecodeBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// newValidateTestServer creates an httptest.Server that handles both the object CRUD path
// (/api/objects) and a validation path (/api/objects/validate).
// It calls validateFn for requests to the validate path.
func newValidateTestServer(t *testing.T, validateFn http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Minimal object store for CRUD.
	objects := map[string]map[string]interface{}{}

	// Validate endpoint — delegates to the provided handler.
	mux.HandleFunc("/api/objects/validate", validateFn)

	// Object CRUD endpoint.
	mux.HandleFunc("/api/objects/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			id := r.URL.Path[len("/api/objects/"):]
			if obj, ok := objects[id]; ok {
				writeJSON(w, obj)
			} else {
				http.NotFound(w, r)
			}
		case http.MethodDelete:
			id := r.URL.Path[len("/api/objects/"):]
			delete(objects, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/objects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var obj map[string]interface{}
		if err := jsonDecodeBody(r, &obj); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, _ := obj["id"].(string)
		if id == "" {
			id = "auto-id"
			obj["id"] = id
		}
		objects[id] = obj
		writeJSON(w, obj)
	})

	return httptest.NewServer(mux)
}

// TestAccRestApiObject_ValidatePath_Success tests that validate_path is called during plan
// and a 200 response allows the plan and apply to succeed.
func TestAccRestApiObject_ValidatePath_Success(t *testing.T) {
	var validateCallCount int64

	svr := newValidateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&validateCallCount, 1)
		w.WriteHeader(http.StatusOK)
	})
	defer svr.Close()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "restapi" {
  uri                  = %q
  write_returns_object = true
}

resource "restapi_object" "test" {
  path            = "/api/objects"
  data            = jsonencode({ id = "val1", name = "test-validate" })
  validate_path   = "/api/objects/validate"
  validate_method = "POST"
}
`, svr.URL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.test", "id", "val1"),
					resource.TestCheckResourceAttr("restapi_object.test", "validate_path", "/api/objects/validate"),
					resource.TestCheckResourceAttr("restapi_object.test", "validate_method", "POST"),
				),
			},
		},
	})

	if atomic.LoadInt64(&validateCallCount) == 0 {
		t.Error("expected validate_path to be called at least once during plan, but it was not called")
	}
}

// TestAccRestApiObject_ValidatePath_Failure tests that a non-2xx from validate_path
// causes the plan to fail with an appropriate error.
func TestAccRestApiObject_ValidatePath_Failure(t *testing.T) {
	svr := newValidateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errorMessage":"tableName is required"}`, http.StatusBadRequest)
	})
	defer svr.Close()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "restapi" {
  uri                  = %q
  write_returns_object = true
}

resource "restapi_object" "test" {
  path          = "/api/objects"
  data          = jsonencode({ id = "bad1", name = "bad-config" })
  validate_path = "/api/objects/validate"
}
`, svr.URL),
				ExpectError: regexp.MustCompile(`API pre-flight validation failed`),
			},
		},
	})
}

// TestAccRestApiObject_NoValidatePath tests that omitting validate_path creates the resource normally.
func TestAccRestApiObject_NoValidatePath(t *testing.T) {
	svr := newValidateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Should not be called.
		t.Error("validate endpoint was called unexpectedly")
		w.WriteHeader(http.StatusOK)
	})
	defer svr.Close()

	resource.UnitTest(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "restapi" {
  uri                  = %q
  write_returns_object = true
}

resource "restapi_object" "test" {
  path = "/api/objects"
  data = jsonencode({ id = "novld1", name = "no-validate" })
}
`, svr.URL),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.test", "id", "novld1"),
					resource.TestCheckNoResourceAttr("restapi_object.test", "validate_path"),
					resource.TestCheckNoResourceAttr("restapi_object.test", "validate_method"),
				),
			},
		},
	})
}
