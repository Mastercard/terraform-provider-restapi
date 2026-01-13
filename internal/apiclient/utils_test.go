package apiclient

import (
	"context"
	"encoding/json"
	"testing"

	jsonpatch "github.com/evanphx/json-patch/v5"
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

func TestApplyJSONPatch(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		patch       string
		expected    string
		expectError bool
	}{
		{
			name:  "copy_operation",
			input: `{"data": {"name": "John", "age": 30}, "id": "123"}`,
			patch: `[
				{"op": "copy", "from": "/data/name", "path": "/name"},
				{"op": "copy", "from": "/data/age", "path": "/age"}
			]`,
			expected:    `{"data": {"name": "John", "age": 30}, "id": "123", "name": "John", "age": 30}`,
			expectError: false,
		},
		{
			name:  "remove_operation",
			input: `{"id": "123", "name": "John", "metadata": {"created": "2024-01-01"}}`,
			patch: `[
				{"op": "remove", "path": "/metadata"}
			]`,
			expected:    `{"id": "123", "name": "John"}`,
			expectError: false,
		},
		{
			name:  "unwrap_data",
			input: `{"data": {"id": "123", "name": "John"}, "metadata": {}}`,
			patch: `[
				{"op": "copy", "from": "/data/id", "path": "/id"},
				{"op": "copy", "from": "/data/name", "path": "/name"},
				{"op": "remove", "path": "/data"},
				{"op": "remove", "path": "/metadata"}
			]`,
			expected:    `{"id": "123", "name": "John"}`,
			expectError: false,
		},
		{
			name:  "move_operation",
			input: `{"old_field": "value", "id": "123"}`,
			patch: `[
				{"op": "move", "from": "/old_field", "path": "/new_field"}
			]`,
			expected:    `{"id": "123", "new_field": "value"}`,
			expectError: false,
		},
		{
			name:        "nil_patch",
			input:       `{"id": "123", "name": "John"}`,
			patch:       "",
			expected:    `{"id": "123", "name": "John"}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputObj map[string]interface{}
			if err := json.Unmarshal([]byte(tt.input), &inputObj); err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			var patch jsonpatch.Patch
			if tt.patch != "" {
				var err error
				patch, err = jsonpatch.DecodePatch([]byte(tt.patch))
				if err != nil {
					t.Fatalf("Failed to decode patch: %v", err)
				}
			}

			result, err := ApplyJSONPatch(ctx, inputObj, patch)

			if tt.expectError && err == nil {
				t.Fatal("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				var expectedObj map[string]interface{}
				if err := json.Unmarshal([]byte(tt.expected), &expectedObj); err != nil {
					t.Fatalf("Failed to parse expected: %v", err)
				}

				resultJSON, _ := json.Marshal(result)
				expectedJSON, _ := json.Marshal(expectedObj)

				if string(resultJSON) != string(expectedJSON) {
					t.Errorf("Result mismatch:\nGot:      %s\nExpected: %s", string(resultJSON), string(expectedJSON))
				}
			}
		})
	}
}
