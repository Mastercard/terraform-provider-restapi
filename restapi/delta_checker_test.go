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
		_, result := getDelta(testCase.o1, testCase.o2, testCase.ignoreList, false)
		if result != testCase.resultHasDelta {
			t.Errorf("delta_checker_test.go: Test Case [%s] wanted [%v] got [%v]", testCase.testCase, testCase.resultHasDelta, result)
		}
	}

	// Test type changes
	for _, testCase := range generateTypeConversionTests() {
		_, result := getDelta(testCase.o1, testCase.o2, testCase.ignoreList, false)
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

	modified, _ := getDelta(recordedInput, actualInput, ignoreList, false)
	if !reflect.DeepEqual(expectedOutput, modified) {
		t.Errorf("delta_checker_test.go: Unexpected delta: expected %v but got %v", expectedOutput, modified)
	}
}

func TestIgnoreServerAdditions(t *testing.T) {
	testCases := []struct {
		name                  string
		recorded              map[string]interface{}
		actual                map[string]interface{}
		ignoreServerAdditions bool
		expectedHasChanges    bool
		expectedResult        map[string]interface{}
	}{
		{
			name:                  "Server adds field - not ignored",
			recorded:              MapAny{"foo": "bar"},
			actual:                MapAny{"foo": "bar", "status": "active"},
			ignoreServerAdditions: false,
			expectedHasChanges:    true,
			expectedResult:        MapAny{"foo": "bar", "status": "active"},
		},
		{
			name:                  "Server adds field - ignored",
			recorded:              MapAny{"foo": "bar"},
			actual:                MapAny{"foo": "bar", "status": "active"},
			ignoreServerAdditions: true,
			expectedHasChanges:    false,
			expectedResult:        MapAny{"foo": "bar"},
		},
		{
			name:                  "Server adds multiple fields - ignored",
			recorded:              MapAny{"name": "test"},
			actual:                MapAny{"name": "test", "created_at": "2025-01-01", "updated_at": "2025-01-02", "status": "active"},
			ignoreServerAdditions: true,
			expectedHasChanges:    false,
			expectedResult:        MapAny{"name": "test"},
		},
		{
			name:                  "Server changes configured field - not affected by ignoreServerAdditions",
			recorded:              MapAny{"name": "original"},
			actual:                MapAny{"name": "changed", "status": "active"},
			ignoreServerAdditions: true,
			expectedHasChanges:    true,
			expectedResult:        MapAny{"name": "changed"},
		},
		{
			name:                  "Server adds nested field - ignored",
			recorded:              MapAny{"config": MapAny{"enabled": true}},
			actual:                MapAny{"config": MapAny{"enabled": true, "timestamp": "2025-01-01"}},
			ignoreServerAdditions: true,
			expectedHasChanges:    false,
			expectedResult:        MapAny{"config": MapAny{"enabled": true}},
		},
		{
			name:                  "Server adds nested field - not ignored",
			recorded:              MapAny{"config": MapAny{"enabled": true}},
			actual:                MapAny{"config": MapAny{"enabled": true, "timestamp": "2025-01-01"}},
			ignoreServerAdditions: false,
			expectedHasChanges:    true,
			expectedResult:        MapAny{"config": MapAny{"enabled": true, "timestamp": "2025-01-01"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, hasChanges := getDelta(tc.recorded, tc.actual, []string{}, tc.ignoreServerAdditions)
			if hasChanges != tc.expectedHasChanges {
				t.Errorf("Expected hasChanges=%v, got %v", tc.expectedHasChanges, hasChanges)
			}
			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Errorf("Expected result=%v, got %v", tc.expectedResult, result)
			}
		})
	}
}
