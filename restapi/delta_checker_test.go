package restapi

import (
	"fmt"
	"reflect"
	"testing"
)

// Creating a type alias to save some typing in the test cases
type MapAny = map[string]interface{}

type deltaTestCase struct {
	testCase       string
	testId         int
	o1             map[string]interface{}
	o2             map[string]interface{}
	ignoreList     []string
	resultHasDelta bool // True if the compared
}

var deltaTestCases = []deltaTestCase{

	// Various cases where there are no changes
	{
		testCase:       "No change 1",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": "bar"},
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	{
		testCase:       "No change - nested object",
		o1:             MapAny{"foo": "bar", "inner": MapAny{"foo": "bar"}},
		o2:             MapAny{"foo": "bar", "inner": MapAny{"foo": "bar"}},
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	{
		testCase:       "No change - has an array",
		o1:             MapAny{"foo": "bar", "list": []string{"foo", "bar"}},
		o2:             MapAny{"foo": "bar", "list": []string{"foo", "bar"}},
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	{
		testCase:       "No change - more types",
		o1:             MapAny{"bool": true, "int": 4, "null": nil},
		o2:             MapAny{"bool": true, "int": 4, "null": nil},
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	// These test cases come in pairs or sets. Each test in the set will have
	// the same o1 and o2 values. We'll first ensure that any changes by the
	// server are detected without an ignore_list, then we'll test the same
	// values again with one or more ignore_lists.

	// Change a field

	{
		testCase:       "Server changes the value of a field",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": "changed"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes the value of a field (ignored)",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": "changed"},
		ignoreList:     []string{"foo"},
		resultHasDelta: false,
	},

	// Handle nils in data
	{
		testCase:       "Server changes the value of a field (nil provided)",
		o1:             MapAny{"foo": nil},
		o2:             MapAny{"foo": "changed"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes the value of a field (nil returned)",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": nil},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server omits setting the value of a null field",
		o1:             MapAny{"foo": "bar", "baz": nil},
		o2:             MapAny{"foo": "bar"},
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	// Add a field

	{
		testCase:       "Server adds a field",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": "bar", "new": "field"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server adds a field (ignored)",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": "bar", "new": "field"},
		ignoreList:     []string{"new"},
		resultHasDelta: false,
	},

	// Remove a field

	{
		testCase:       "Server removes a field",
		o1:             MapAny{"foo": "bar", "id": "foobar"},
		o2:             MapAny{"foo": "bar"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server removes a field (ignored)",
		o1:             MapAny{"foo": "bar", "id": "foobar"},
		o2:             MapAny{"foo": "bar"},
		ignoreList:     []string{"id"},
		resultHasDelta: false,
	},

	// Deep fields

	{
		testCase:       "Server changes a deep field",
		o1:             MapAny{"outside": MapAny{"change": "a"}},
		o2:             MapAny{"outside": MapAny{"change": "b"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server adds a deep field",
		o1:             MapAny{"outside": MapAny{"change": "a"}},
		o2:             MapAny{"outside": MapAny{"change": "a", "add": "a"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server removes a deep field",
		o1:             MapAny{"outside": MapAny{"change": "a", "remove": "a"}},
		o2:             MapAny{"outside": MapAny{"change": "a"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	// Deep fields (but ignored)

	{
		testCase:       "Server changes a deep field (ignored)",
		o1:             MapAny{"outside": MapAny{"change": "a"}},
		o2:             MapAny{"outside": MapAny{"change": "b"}},
		ignoreList:     []string{"outside.change"},
		resultHasDelta: false,
	},

	{
		testCase:       "Server adds a deep field (ignored)",
		o1:             MapAny{"outside": MapAny{"change": "a"}},
		o2:             MapAny{"outside": MapAny{"change": "a", "add": "a"}},
		ignoreList:     []string{"outside.add"},
		resultHasDelta: false,
	},

	{
		testCase:       "Server removes a deep field (ignored)",
		o1:             MapAny{"outside": MapAny{"change": "a", "remove": "a"}},
		o2:             MapAny{"outside": MapAny{"change": "a"}},
		ignoreList:     []string{"outside.remove"},
		resultHasDelta: false,
	},
	// Similar to 12: make sure we notice a change to a deep field even when we ignore some of them
	{
		testCase:       "Server changes/adds/removes a deep field (ignored 2)",
		o1:             MapAny{"outside": MapAny{"watch": "me", "change": "a", "remove": "a"}},
		o2:             MapAny{"outside": MapAny{"watch": "me_change", "change": "b", "add": "a"}},
		ignoreList:     []string{"outside.change", "outside.add", "outside.remove"},
		resultHasDelta: true,
	},

	// Similar to 12,13 but ignore the whole "outside"
	{
		testCase:       "Server changes/adds/removes a deep field (ignore root field)",
		o1:             MapAny{"outside": MapAny{"watch": "me", "change": "a", "remove": "a"}},
		o2:             MapAny{"outside": MapAny{"watch": "me_change", "change": "b", "add": "a"}},
		ignoreList:     []string{"outside"},
		resultHasDelta: false,
	},

	// Basic List Changes
	// Note: we don't support ignoring specific differences to lists - only ignoring the list as a whole
	{
		testCase:       "Server adds to list",
		o1:             MapAny{"list": []string{"foo", "bar"}},
		o2:             MapAny{"list": []string{"foo", "bar", "baz"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server removes from list",
		o1:             MapAny{"list": []string{"foo", "bar"}},
		o2:             MapAny{"list": []string{"foo"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes an item in the list",
		o1:             MapAny{"list": []string{"foo", "bar"}},
		o2:             MapAny{"list": []string{"foo", "BAR"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server rearranges the list",
		o1:             MapAny{"list": []string{"foo", "bar"}},
		o2:             MapAny{"list": []string{"bar", "foo"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes the list but we ignore the whole list",
		o1:             MapAny{"list": []string{"foo", "bar"}},
		o2:             MapAny{"list": []string{"bar", "foo"}},
		ignoreList:     []string{"list"},
		resultHasDelta: false,
	},

	// We don't currently support ignoring a change like this, but we could in the future with a syntax like `list[].val` similar to jq
	{
		testCase:       "Server changes a sub-value in a list of objects",
		o1:             MapAny{"list": []MapAny{{"key": "foo", "val": "x"}, {"key": "bar", "val": "x"}}},
		o2:             MapAny{"list": []MapAny{{"key": "foo", "val": "Y"}, {"key": "bar", "val": "Z"}}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	// NEW: Test cases for list item ignoring with [] syntax
	{
		testCase:       "Server changes a sub-value in a list of objects (ignored with [])",
		o1:             MapAny{"list": []MapAny{{"key": "foo", "val": "x"}, {"key": "bar", "val": "x"}}},
		o2:             MapAny{"list": []MapAny{{"key": "foo", "val": "Y"}, {"key": "bar", "val": "Z"}}},
		ignoreList:     []string{"list[].val"},
		resultHasDelta: false,
	},

	{
		testCase:       "Server changes a sub-value in a list but we watch another field",
		o1:             MapAny{"list": []MapAny{{"key": "foo", "val": "x"}, {"key": "bar", "val": "x"}}},
		o2:             MapAny{"list": []MapAny{{"key": "FOO", "val": "Y"}, {"key": "BAR", "val": "Z"}}},
		ignoreList:     []string{"list[].val"},
		resultHasDelta: true,
	},

	{
		testCase:       "Server adds a field to list items (ignored with [])",
		o1:             MapAny{"items": []MapAny{{"name": "a"}, {"name": "b"}}},
		o2:             MapAny{"items": []MapAny{{"name": "a", "added": "new"}, {"name": "b", "added": "new"}}},
		ignoreList:     []string{"items[].added"},
		resultHasDelta: false,
	},

	{
		testCase:       "Server changes nested field in list items (ignored with [])",
		o1:             MapAny{"items": []MapAny{{"data": MapAny{"secret": "x"}}, {"data": MapAny{"secret": "y"}}}},
		o2:             MapAny{"items": []MapAny{{"data": MapAny{"secret": "CHANGED"}}, {"data": MapAny{"secret": "CHANGED"}}}},
		ignoreList:     []string{"items[].data.secret"},
		resultHasDelta: false,
	},

	{
		testCase:       "List length changes (not ignoreable)",
		o1:             MapAny{"list": []MapAny{{"val": "x"}}},
		o2:             MapAny{"list": []MapAny{{"val": "x"}, {"val": "y"}}},
		ignoreList:     []string{"list[].val"},
		resultHasDelta: true,
	},

	// NEW: Test cases for keys containing dots
	{
		testCase:       "Ignore a key containing a dot",
		o1:             MapAny{"@odata.etag": "v1", "name": "foo"},
		o2:             MapAny{"@odata.etag": "v2", "name": "foo"},
		ignoreList:     []string{"@odata.etag"},
		resultHasDelta: false,
	},

	{
		testCase:       "Server adds a key containing a dot (ignored)",
		o1:             MapAny{"name": "foo"},
		o2:             MapAny{"name": "foo", "@odata.context": "http://example.com"},
		ignoreList:     []string{"@odata.context"},
		resultHasDelta: false,
	},

	{
		testCase:       "Dotted key not in ignore list should detect changes",
		o1:             MapAny{"@odata.etag": "v1", "name": "foo"},
		o2:             MapAny{"@odata.etag": "v2", "name": "foo"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Nested object with dotted keys (ignored)",
		o1:             MapAny{"metadata": MapAny{"@odata.type": "old"}},
		o2:             MapAny{"metadata": MapAny{"@odata.type": "new"}},
		ignoreList:     []string{"metadata.@odata.type"},
		resultHasDelta: false,
	},

	// Combined test: Azure-like scenario from GitHub issue
	{
		testCase:       "Azure-style response with @odata fields and credentials (ignored)",
		o1:             MapAny{"name": "datasource", "credentials": MapAny{"connectionString": nil}},
		o2:             MapAny{"name": "datasource", "@odata.etag": "\"0x12345\"", "credentials": MapAny{"connectionString": "ResourceId=/sub/..."}},
		ignoreList:     []string{"@odata.etag", "credentials.connectionString"},
		resultHasDelta: false,
	},
}

/*
 * Since I'm not super familiar with Go, and most of the hasDelta code is
 * effectively "untyped", I want to make sure my code doesn't fail when
 * comparing certain type combinations
 */
func generateTypeConversionTests() []deltaTestCase {
	typeValues := MapAny{
		"string":     "foo",
		"number":     42,
		"object":     MapAny{"foo": "bar"},
		"array":      []string{"foo", "bar"},
		"bool_true":  true,
		"bool_false": false,
	}

	tests := make([]deltaTestCase, len(typeValues)*len(typeValues))

	testCounter := 0
	for fromType, fromValue := range typeValues {
		for toType, toValue := range typeValues {
			tests = append(tests, deltaTestCase{
				testCase:       fmt.Sprintf("Type Conversion from [%s] to [%s]", fromType, toType),
				o1:             MapAny{"value": fromValue},
				o2:             MapAny{"value": toValue},
				ignoreList:     []string{},
				resultHasDelta: fromType != toType,
			})

			testCounter += 1
		}
	}

	return tests
}

func TestHasDelta(t *testing.T) {
	// Run the main test cases
	for _, testCase := range deltaTestCases {
		_, result := getDelta(testCase.o1, testCase.o2, testCase.ignoreList)
		if result != testCase.resultHasDelta {
			t.Errorf("delta_checker_test.go: Test Case [%s] wanted [%v] got [%v]", testCase.testCase, testCase.resultHasDelta, result)
		}
	}

	// Test type changes
	for _, testCase := range generateTypeConversionTests() {
		_, result := getDelta(testCase.o1, testCase.o2, testCase.ignoreList)
		if result != testCase.resultHasDelta {
			t.Errorf("delta_checker_test.go: TYPE CONVERSION Test Case [%d:%s] wanted [%v] got [%v]", testCase.testId, testCase.testCase, testCase.resultHasDelta, result)
		}
	}
}

func TestHasDeltaModifiedResource(t *testing.T) {

	// Test modifiedResource return val

	recordedInput := map[string]interface{}{
		"name":  "Joey",
		"color": "tabby",
		"hobbies": map[string]interface{}{
			"hunting": "birds",
			"eating":  "plants",
		},
	}

	actualInput := map[string]interface{}{
		"color":    "tabby",
		"hairball": true,
		"hobbies": map[string]interface{}{
			"hunting":  "birds",
			"eating":   "plants",
			"sleeping": "yep",
		},
	}

	expectedOutput := map[string]interface{}{
		"name":  "Joey",
		"color": "tabby",
		"hobbies": map[string]interface{}{
			"hunting": "birds",
			"eating":  "plants",
		},
	}

	ignoreList := []string{"hairball", "hobbies.sleeping", "name"}

	modified, _ := getDelta(recordedInput, actualInput, ignoreList)
	if !reflect.DeepEqual(expectedOutput, modified) {
		t.Errorf("delta_checker_test.go: Unexpected delta: expected %v but got %v", expectedOutput, modified)
	}
}

func TestSliceTypePreservation(t *testing.T) {
	// Test that slice types are preserved when using [] syntax
	recordedInput := map[string]interface{}{
		"items": []MapAny{{"key": "foo", "val": "x"}, {"key": "bar", "val": "x"}},
	}

	actualInput := map[string]interface{}{
		"items": []MapAny{{"key": "foo", "val": "CHANGED"}, {"key": "bar", "val": "CHANGED"}},
	}

	ignoreList := []string{"items[].val"}

	modified, hasDelta := getDelta(recordedInput, actualInput, ignoreList)

	if hasDelta {
		t.Errorf("Expected no delta when ignoring items[].val, but got hasDelta=true")
	}

	// Verify the type is preserved
	items, ok := modified["items"]
	if !ok {
		t.Errorf("Expected 'items' key in modified resource")
		return
	}

	itemsType := reflect.TypeOf(items)
	expectedType := reflect.TypeOf([]MapAny{})

	if itemsType != expectedType {
		t.Errorf("Slice type not preserved: expected %v but got %v", expectedType, itemsType)
	}
}
