package restapi

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGetStringAtKey(t *testing.T) {
	ctx := context.Background()
	testObj := make(map[string]interface{})
	err := json.Unmarshal([]byte(`
    {
      "rootFoo": "bar",
      "top": {
        "foo": "bar",
        "number": 1234567890,
        "float": 1.23456789,
        "middle": {
          "bottom": {
            "foo": "bar"
          }
        },
        "list": [
          "bar",
          "baz"
        ]
      },
	  "trueFalse": true
    }
  `), &testObj)
	if nil != err {
		t.Fatalf("Error unmarshalling JSON: %s", err)
	}

	var res string

	res, err = GetStringAtKey(ctx, testObj, "rootFoo")
	if err != nil {
		t.Fatalf("Error extracting 'rootFoo' from JSON payload: %s", err)
	} else if res != "bar" {
		t.Fatalf("Error: Expected 'bar', but got %s", res)
	}

	res, err = GetStringAtKey(ctx, testObj, "trueFalse")
	if err != nil {
		t.Fatalf("Error extracting 'trueFalse' from JSON payload: %s", err)
	} else if res != "true" {
		t.Fatalf("Error: Expected 'true', but got %s", res)
	}

	res, err = GetStringAtKey(ctx, testObj, "top/foo")
	if err != nil {
		t.Fatalf("Error extracting 'top/foo' from JSON payload: %s", err)
	} else if res != "bar" {
		t.Fatalf("Error: Expected 'bar', but got %s", res)
	}

	res, err = GetStringAtKey(ctx, testObj, "top/middle/bottom/foo")
	if err != nil {
		t.Fatalf("Error extracting top/foo from JSON payload: %s", err)
	} else if res != "bar" {
		t.Fatalf("Error: Expected 'bar', but got %s", res)
	}

	_, err = GetStringAtKey(ctx, testObj, "top/middle/junk")
	if err == nil {
		t.Fatalf("Error expected when trying to extract 'top/middle/junk' from payload")
	}

	res, err = GetStringAtKey(ctx, testObj, "top/number")
	if err != nil {
		t.Fatalf("Error extracting 'top/number' from JSON payload: %s", err)
	} else if res != "1234567890" {
		t.Fatalf("Error: Expected '1234567890', but got %s", res)
	}

	res, err = GetStringAtKey(ctx, testObj, "top/float")
	if err != nil {
		t.Fatalf("Error extracting 'top/float' from JSON payload: %s", err)
	} else if res != "1.23456789" {
		t.Fatalf("Error: Expected '1.23456789', but got %s", res)
	}
}

func TestGetListStringAtKey(t *testing.T) {
	ctx := context.Background()
	testObj := make(map[string]interface{})
	err := json.Unmarshal([]byte(`
    {
      "rootFoo": "bar",
      "items": [
        {
          "foo": "bar",
          "number": 1,
          "list_numbers": [1, 2, 3],
          "test": [{"id": "3333"}, {"id": "1337"}],
          "resource": {
            "id": "123"
          }
        }
      ]
    }
  `), &testObj)
	if nil != err {
		t.Fatalf("Error unmarshalling JSON: %s", err)
	}

	var res string

	res, err = GetStringAtKey(ctx, testObj, "items/0/resource/id")
	if err != nil {
		t.Fatalf("Error extracting 'resource' from JSON payload: %s", err)
	} else if res != "123" {
		t.Fatalf("Error: Expected '123', but got %s", res)
	}

	res, err = GetStringAtKey(ctx, testObj, "items/0/test/1/id")
	if err != nil {
		t.Fatalf("Error extracting 'resource' from JSON payload: %s", err)
	} else if res != "1337" {
		t.Fatalf("Error: Expected '1337', but got %s", res)
	}

	res, err = GetStringAtKey(ctx, testObj, "items/0/list_numbers/1")
	if err != nil {
		t.Fatalf("Error extracting 'resource' from JSON payload: %s", err)
	} else if res != "2" {
		t.Fatalf("Error: Expected '2', but got %s", res)
	}
}
