package restapi

import (
	"testing"
	"fmt"
	// "log"
)

// Creating a type alias to save some typing in the test cases
type MapAny = map[string]interface{}

type deltaTestCase struct {
	testCase       string                 `json:"Test_case"`
	testId         int                    `json:"Id"`
	o1             map[string]interface{} `json:o1`
	o2             map[string]interface{} `json:o2`
	ignoreList     []string
	resultHasDelta bool                   // True if the compared
}

var deltaTestCases = []deltaTestCase{

	// Various cases where there are no changes
	{
		testCase:       "No change 1",
		testId:         1,
		o1:             MapAny{ "foo": "bar" },
		o2:             MapAny{ "foo": "bar" },
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	{
		testCase:       "No change - nested object",
		testId:         2,
		o1:             MapAny{"foo":"bar", "inner": MapAny{"foo":"bar"} },
		o2:             MapAny{"foo":"bar", "inner": MapAny{"foo":"bar"} },
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	{
		testCase:       "No change - has an array",
		testId:         3,
		o1:             MapAny{"foo":"bar", "list": []string{"foo", "bar"} },
		o2:             MapAny{"foo":"bar", "list": []string{"foo", "bar"} },
		ignoreList:     []string{},
		resultHasDelta: false,
	},

	{
		testCase:       "No change - more types",
		testId:         4,
		o1:             MapAny{"bool":true, "int": 4 },
		o2:             MapAny{"bool":true, "int": 4 },
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
		testId:         5,
		o1:             MapAny{"foo":"bar"},
		o2:             MapAny{"foo":"changed"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes the value of a field (ignored)",
		testId:         6,
		o1:             MapAny{"foo":"bar"},
		o2:             MapAny{"foo":"changed"},
		ignoreList:     []string{ "foo" },
		resultHasDelta: false,
	},

	// Add a field

	{
		testCase:       "Server adds a field",
		testId:         7,
		o1:             MapAny{"foo":"bar"},
		o2:             MapAny{"foo":"bar", "new":"field"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server adds a field (ignored)",
		testId:         8,
		o1:             MapAny{"foo":"bar"},
		o2:             MapAny{"foo":"bar", "new":"field"},
		ignoreList:     []string{ "new" },
		resultHasDelta: false,
	},

	// Remove a field

	{
		testCase:       "Server removes a field",
		testId:         9,
		o1:             MapAny{"foo":"bar", "id": "foobar"},
		o2:             MapAny{"foo":"bar"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server removes a field (ignored)",
		testId:         10,
		o1:             MapAny{"foo":"bar", "id": "foobar"},
		o2:             MapAny{"foo":"bar"},
		ignoreList:     []string{ "id" },
		resultHasDelta: false,
	},

	// Deep fields

	{
		testCase:       "Server changes/adds/removes a deep field",
		testId:         11,
		o1:             MapAny{"outside": MapAny{"change":"a", "remove":"a"}},
		o2:             MapAny{"outside": MapAny{"change":"b", "add":"a"}},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes/adds/removes a deep field (ignored)",
		testId:         12,
		o1:             MapAny{"outside": MapAny{"change":"a", "remove":"a"}},
		o2:             MapAny{"outside": MapAny{"change":"b", "add":"a"}},
		ignoreList:     []string{ "outside.change", "outside.add", "outside.remove" },
		resultHasDelta: false,
	},

	// Similar to 12: make sure we notice a change to a deep field even when we ignore some of them
	{
		testCase:       "Server changes/adds/removes a deep field (ignored 2)",
		testId:         13,
		o1:             MapAny{"outside": MapAny{"watch":"me", "change":"a", "remove":"a"}},
		o2:             MapAny{"outside": MapAny{"watch":"me_change","change":"b", "add":"a"}},
		ignoreList:     []string{ "outside.change", "outside.add", "outside.remove" },
		resultHasDelta: true,
	},

	// Similar to 12,13 but ignore the whole "outside"
	{
		testCase:       "Server changes/adds/removes a deep field (ignore root field)",
		testId:         14,
		o1:             MapAny{"outside": MapAny{"watch":"me", "change":"a", "remove":"a"}},
		o2:             MapAny{"outside": MapAny{"watch":"me_change","change":"b", "add":"a"}},
		ignoreList:     []string{ "outside" },
		resultHasDelta: false,
	},


	// Basic List Changes
	// Note: we don't support ignoring specific differences to lists - only ignoring the list as a whole
	{
		testCase:       "Server adds to list",
		testId:         15,
		o1:             MapAny{"list": []string{"foo", "bar"} },
		o2:             MapAny{"list": []string{"foo", "bar", "baz"} },
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server removes from list",
		testId:         16,
		o1:             MapAny{"list": []string{"foo", "bar"} },
		o2:             MapAny{"list": []string{"foo"} },
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes an item in the list",
		testId:         17,
		o1:             MapAny{"list": []string{"foo", "bar"} },
		o2:             MapAny{"list": []string{"foo", "BAR"} },
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server rearranges the list",
		testId:         18,
		o1:             MapAny{"list": []string{"foo", "bar"} },
		o2:             MapAny{"list": []string{"bar", "foo"} },
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server changes the list but we ignore the whole list",
		testId:         19,
		o1:             MapAny{"list": []string{"foo", "bar"} },
		o2:             MapAny{"list": []string{"bar", "foo"} },
		ignoreList:     []string{ "list" },
		resultHasDelta: false,
	},

	// We don't currently support ignoring a change like this, but we could in the future with a syntax like `list[].val` similar to jq
	{
		testCase:       "Server changes a sub-value in a list of objects",
		testId:         19,
		o1:             MapAny{"list": []MapAny{ {"key":"foo", "val":"x"}, {"key":"bar", "val":"x"} } },
		o2:             MapAny{"list": []MapAny{ {"key":"foo", "val":"Y"}, {"key":"bar", "val":"Z"} } },
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
		"string": "foo",
		"number": 42,
		"object": MapAny{"foo":"bar"},
		"array": []string { "foo", "bar" },
		"bool_true": true,
		"bool_false": false,
	}

	tests := make([]deltaTestCase, len(typeValues) * len(typeValues))

	testCounter := 0
	for fromType, fromValue := range typeValues {
		for toType, toValue := range typeValues {
			tests = append(tests, deltaTestCase{
				testCase:       fmt.Sprintf("Type Conversion from [%s] to [%s]", fromType, toType),
				testId:         testCounter,
				o1:             MapAny{"value": fromValue },
				o2:             MapAny{"value": toValue },
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
			t.Errorf("delta_checker_test.go: Test Case [%d:%s] wanted [%v] got [%v]", testCase.testId, testCase.testCase, testCase.resultHasDelta, result)
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
