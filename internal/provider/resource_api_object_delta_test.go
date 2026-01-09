package provider

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
	ignoreAll      bool
	resultHasDelta bool // True if the compared
}

var deltaTestCases = []deltaTestCase{
	{
		testCase:       "Server changes the value of a field (ignored)",
		o1:             MapAny{"foo": "bar"},
		o2:             MapAny{"foo": "changed"},
		ignoreList:     []string{"foo"},
		resultHasDelta: false,
	},

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
		o1:             MapAny{"foo": "bar", "baz": "foobar"},
		o2:             MapAny{"foo": "bar"},
		ignoreList:     []string{},
		resultHasDelta: true,
	},

	{
		testCase:       "Server removes a field (ignored)",
		o1:             MapAny{"foo": "bar", "baz": "foobar"},
		o2:             MapAny{"foo": "bar"},
		ignoreList:     []string{"baz"},
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

	{
		testCase:       "Ignore all server changes",
		o1:             MapAny{"foo": "bar", "list": []string{"foo", "bar"}},
		o2:             MapAny{"foo": "baz"},
		ignoreList:     []string{},
		ignoreAll:      true,
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

// The below test is disabled in favor of the faster TestHasDelta unit test.
// However, we are keeping it around for now since it does do an end-to-end
// test of the resource with a fake server. When messing with the delta logic,
// particularly if we get into complex changes in ModifyPlan, it can be useful
//  to re-enable this test.
/*
func TestModifyPlan(t *testing.T) {
	debug := false
	ctx := context.Background()

	if debug {
		os.Setenv("TF_LOG", "DEBUG")
	}
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(8083, apiServerObjects, map[string]string{}, true, debug, "")
	os.Setenv("REST_API_URI", "http://127.0.0.1:8083")

	opt := &apiclient.APIClientOpt{
		URI:                 "http://127.0.0.1:8083/",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IDAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
		Debug:               debug,
	}
	client, err := apiclient.NewAPIClient(opt)
	if err != nil {
		t.Fatal(err)
	}

	for _, cs := range deltaTestCases {
		t.Run(cs.testCase, func(t *testing.T) {
			cs.o1["id"] = "1234"
			cs.o2["id"] = "1234"
			resourceData, _ := json.Marshal(cs.o1)
			serverData, _ := json.Marshal(cs.o2)

			resource.UnitTest(t, resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				PreCheck:                 func() { svr.StartInBackground() },
				Steps: []resource.TestStep{
					// Step 1: Create the object with o1, AKA: the resourceData
					{
						PreConfig: func() {
							client.SendRequest(ctx, http.MethodDelete, "/api/objects/1234", "", debug)
						},
						Config: generateTestResource(
							"Foo",
							string(resourceData),
							map[string]interface{}{
								"ignore_changes_to":         cs.ignoreList,
								"ignore_all_server_changes": cs.ignoreAll,
							},
							debug,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccCheckRestapiObjectExists("restapi_object.Foo", "1234", client),
							resource.TestCheckResourceAttr("restapi_object.Foo", "id", "1234"),
						),
					},

					// Step 2: Modify the object on the server to be o2 (AKA: serverData)
					// This simulates the server making changes (or not) and us detecting the drift
					{
						PlanOnly:           true,
						ExpectNonEmptyPlan: cs.resultHasDelta,
						PreConfig: func() {
							client.SendRequest(ctx, http.MethodPut, "/api/objects/1234", string(serverData), debug)
						},
						Config: generateTestResource(
							"Foo",
							string(resourceData),
							map[string]interface{}{
								"ignore_changes_to":         cs.ignoreList,
								"ignore_all_server_changes": cs.ignoreAll,
							},
							debug,
						),
						Check: resource.ComposeTestCheckFunc(
							testAccCheckRestapiObjectExists("restapi_object.Foo", "1234", client),
							resource.TestCheckResourceAttr("restapi_object.Foo", "id", "1234"),
						),
					},
				},
			})
		})
	}

	svr.Shutdown()
}
*/

// generateTypeConversionTests tests many different type combinations
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
		if result != testCase.resultHasDelta && !testCase.ignoreAll {
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
