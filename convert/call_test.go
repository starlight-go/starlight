package convert_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/1set/starlight/convert"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// TestCallStarlarkFunctionInGo tests calling a Starlark function in Go with various arguments.
func TestCallStarlarkFunctionInGo(t *testing.T) {
	code := `
def greet(name="John"):
	if name == "null":
		fail("name cannot be 'null'")
	return "Hello, " + name + "!"

greet_func = greet
`
	// run the starlark code
	globals, err := execStarlark(code, nil)
	if err != nil {
		t.Fatalf(`expected no error, but got %v`, err)
	}

	// retrieve the starlark function
	greet, ok := globals["greet_func"].(*starlark.Function)
	if !ok {
		t.Fatalf(`expected greet_func to be a *starlark.Function, but got %T`, globals["greet_func"])
	}
	thread := &starlark.Thread{
		Name:  "test",
		Print: func(_ *starlark.Thread, msg string) { fmt.Println("🌟", msg) },
	}

	// call the starlark function with no arguments
	if res, err := starlark.Call(thread, greet, starlark.Tuple{}, nil); err != nil {
		t.Fatalf(`expected no error while calling greet(), but got %v`, err)
	} else if resStr, ok := res.(starlark.String); !ok {
		t.Fatalf(`expected greet() to return a starlark.String, but got %T`, resStr)
	} else if resStr.GoString() != `Hello, John!` {
		t.Fatalf(`expected greet() to return "Hello, John!", but got %s`, resStr.GoString())
	}

	// call the starlark function with one argument
	jane, _ := convert.ToValue("Jane")
	if res, err := starlark.Call(thread, greet, starlark.Tuple{jane}, nil); err != nil {
		t.Fatalf(`expected no error while calling greet("Jane"), but got %v`, err)
	} else if resStr, ok := res.(starlark.String); !ok {
		t.Fatalf(`expected greet("Jane") to return a starlark.String, but got %T`, resStr)
	} else if resStr.GoString() != `Hello, Jane!` {
		t.Fatalf(`expected greet("Jane") to return "Hello, Jane!", but got %s`, resStr.GoString())
	}

	// call the starlark function with extra arguments
	doe, _ := convert.ToValue("Doe")
	if _, err := starlark.Call(thread, greet, starlark.Tuple{jane, doe}, nil); err == nil {
		t.Fatalf(`expected an error while calling greet("Jane", "Doe"), but got none`)
	}

	// call the starlark function and expect an error
	if _, err := starlark.Call(thread, greet, starlark.Tuple{starlark.String("null")}, nil); err == nil {
		t.Fatalf(`expected an error while calling greet("null"), but got none`)
	}
}

// TestUseGoValueInStarlark tests using various Go values in Starlark. It verifies:
// 1. the Go value can be converted to Starlark values as input;
// 2. the converted Starlark values can be used in Starlark code;
func TestUseGoValueInStarlark(t *testing.T) {
	// for common go values, convert them to starlark values and run the starlark code with go assert and starlark test assert
	codeCompareList := `

print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	for i in range(len(exp)):
		if go_value[i] != exp[i]:
			fail('go_value[{}] {}({}) is not equal to {}({})'.format(i, go_value[i],type(go_value[i]), exp[i],type(exp[i])))
		else:
			print('go_value[{}] {}({}) == {}({})'.format(i, go_value[i],type(go_value[i]), exp[i],type(exp[i])))
test()
`
	codeCompareMapDict := `

print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	el = sorted(list(exp.items()))
	al = sorted(list(go_value.items()))
	if el != al:
		fail('go_value {}({}) is not equal to {}({})'.format(go_value,type(go_value), exp,type(exp)))
`

	type testCase struct {
		name        string
		goValue     interface{}
		codeSnippet string
		wantErrConv bool
		wantErrExec bool
	}
	testCases := []testCase{
		{
			name:    "nil",
			goValue: nil,
			codeSnippet: `
assert.Equal(None, go_value)

print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value != None:
		fail('go_value is not None')
test()
`,
		},
		{
			name:    "int",
			goValue: 123,
			codeSnippet: `
assert.Equal(123, go_value)

print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value != 123:
		fail('go_value is not 123')
test()
`,
		},
		{
			name:        "float",
			goValue:     123.456,
			codeSnippet: `assert.Equal(123.456, go_value)`,
		},
		{
			name:        "bigint",
			goValue:     big.NewInt(1234567890),
			codeSnippet: `assert.Equal(1234567890, go_value.Int64())`,
		},
		{
			name:    "string",
			goValue: "aloha",
			codeSnippet: `
assert.Equal('aloha', go_value)

print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value != 'aloha':
		fail('go_value is not "aloha"')
test()
`,
		},
		{
			name:        "slice of interface",
			goValue:     []interface{}{123, "world"},
			codeSnippet: `exp = [123, "world"]` + codeCompareList,
			wantErrExec: true, // for []interface{}, convert to GoSlice+GoInterface
		},
		{
			name:        "complex slice of interface",
			goValue:     []interface{}{123, "world", []int{1, 2, 3}, []string{"hello", "world"}},
			codeSnippet: `exp = [123, "world", [1, 2, 3], ["hello", "world"]]` + codeCompareList,
			wantErrExec: true, // for complex []interface{}, convert to GoSlice+GoInterface
		},
		{
			name:        "slice of int",
			goValue:     []int{123, 456},
			codeSnippet: `exp = [123, 456]` + codeCompareList,
		},
		{
			name:        "slice of string",
			goValue:     []string{"hello", "world"},
			codeSnippet: `exp = ["hello", "world"]` + codeCompareList,
		},
		{
			name:        "slice of bool",
			goValue:     []bool{true, false},
			codeSnippet: `exp = [True, False]` + codeCompareList,
		},
		{
			name:        "array of interface",
			goValue:     [2]interface{}{123, "world"},
			codeSnippet: `exp = [123, "world"]` + codeCompareList,
			wantErrExec: true, // for [2]interface{}, convert to GoSlice+GoInterface
		},
		{
			name:        "complex array of interface",
			goValue:     [4]interface{}{123, "world", []int{1, 2, 3}, []string{"hello", "world"}},
			codeSnippet: `exp = [123, "world", [1, 2, 3], ["hello", "world"]]` + codeCompareList,
			wantErrExec: true, // for complex [4]interface{}, convert to GoSlice+GoInterface
		},
		{
			name:        "array of int",
			goValue:     [2]int{123, 456},
			codeSnippet: `exp = [123, 456]` + codeCompareList,
		},
		{
			name:        "array of string",
			goValue:     [2]string{"hello", "world"},
			codeSnippet: `exp = ["hello", "world"]` + codeCompareList,
		},
		{
			name:        "array of bool",
			goValue:     [2]bool{true, false},
			codeSnippet: `exp = [True, False]` + codeCompareList,
		},
		{
			name:        "map of string to int",
			goValue:     map[string]int{"one": 1, "two": 2},
			codeSnippet: `exp = {"one": 1, "two": 2}` + codeCompareMapDict,
		},
		{
			name:        "map of int to string",
			goValue:     map[int]string{1: "one", 2: "two"},
			codeSnippet: `exp = {1: "one", 2: "two"}` + codeCompareMapDict,
		},
		{
			name:        "map of string to slice of int",
			goValue:     map[string][]int{"one": {1, 2}, "two": {3, 4}},
			codeSnippet: `exp = {"one": [1, 2], "two": [3, 4]}` + codeCompareMapDict,
		},
		{
			name:        "map of string to slice of string",
			goValue:     map[string][]string{"one": {"1", "2"}, "two": {"3", "4"}},
			codeSnippet: `exp = {"one": ["1", "2"], "two": ["3", "4"]}` + codeCompareMapDict,
		},
		{
			name:        "map of string to slice of slice",
			goValue:     map[string][][]int{"one": {{1, 2}, {3, 4}}, "two": {{5, 6}, {7, 8}}},
			codeSnippet: `exp = {"one": [[1, 2], [3, 4]], "two": [[5, 6], [7, 8]]}` + codeCompareMapDict,
		},
		{
			name: "map of custom struct",
			goValue: map[string]customStruct{
				"one": {Name: "John", Value: 42},
			},
			codeSnippet: `
print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value['one'].Name != 'John' or go_value['one'].Value != 42:
		fail('go_value is not "John"/42')
test()
`,
		},
		{
			name: "map of slice of custom struct",
			goValue: map[string][]customStruct{
				"one": {{Name: "John", Value: 42}, {Name: "Jane", Value: 43}},
			},
			codeSnippet: `
print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value['one'][0].Name != 'John' or go_value['one'][0].Value != 42:
		fail('go_value is not "John"/42')
	if go_value['one'][1].Name != 'Jane' or go_value['one'][1].Value != 43:
		fail('go_value is not "Jane"/43')
test()
`,
		},
		{
			name:    "empty struct",
			goValue: struct{}{},
			codeSnippet: `
print('※ go_value: {}({})'.format(go_value, type(go_value)))
assert.Equal({}, go_value)
`,
			wantErrExec: true,
		},
		{
			name: "custom struct",
			goValue: struct {
				Name  string
				Value int
			}{Name: "Hello", Value: 42},
			codeSnippet: `
print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value.Name != 'Hello' or go_value.Value != 42:
		fail('go_value is not "aloha"')
test()
`,
		},
		{
			name: "custom function",
			goValue: func(name string) string {
				return "Hello " + name
			},
			codeSnippet: `
print('※ go_value: {}({})'.format(go_value, type(go_value)))
def test():
	if go_value("World") != 'Hello World':
		fail('go_value is not "Hello"')
test()
`,
		},
		{
			name:    "unsupported type",
			goValue: make(chan bool),
			codeSnippet: `
print('※ go_value: {}({})'.format(go_value, type(go_value)))
`,
			wantErrConv: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			globals := map[string]interface{}{
				"assert":   &assert{t: t},
				"go_value": tc.goValue,
			}

			// convert go values to Starlark values as predefined globals
			env, errConv := convert.MakeStringDict(globals)
			if errConv != nil == !tc.wantErrConv {
				t.Fatalf(`expected no error while converting globals, but got %v`, errConv)
			} else if errConv == nil && tc.wantErrConv {
				t.Fatalf(`expected an error while converting globals, but got none`)
			}
			if errConv != nil {
				return
			}

			// run the Starlark code to test the converted globals
			_, errExec := execStarlark(tc.codeSnippet, env)
			if errExec != nil && !tc.wantErrExec {
				t.Fatalf(`expected no error while executing code snippet, but got %v`, errExec)
			} else if errExec == nil && tc.wantErrExec {
				t.Fatalf(`expected an error while executing code snippet, but got none`)
			}
		})
	}
}

// TestCallGoFunctionInStarlark tests calling Go functions in Starlark with various types of arguments and return values.
// It verifies:
// 1. Go functions can be converted to Starlark functions, underlying Starlark values will be converted to Go values with specified types;
// 2. Return values of Go functions can be converted to Starlark values;
// 3. Starlark values can be converted to Go values;
func TestCallGoFunctionInStarlark(t *testing.T) {
	type testCase struct {
		name         string
		goFunc       interface{}
		codeSnippet  string
		expectResult interface{}
		wantErrExec  bool
		wantEqual    bool
	}
	testCases := []testCase{
		{
			name: "func() string",
			goFunc: func() string {
				return "Aloha!"
			},
			codeSnippet:  `sl_value = go_func()`,
			expectResult: "Aloha!",
			wantEqual:    true,
		},
		{
			name: "func(string) string",
			goFunc: func(name string) string {
				return "Hello " + name + "!"
			},
			codeSnippet:  `sl_value = go_func("World")`,
			expectResult: "Hello World!",
			wantEqual:    true,
		},
		{
			name: "func(string) int",
			goFunc: func(name string) int {
				return len(name)
			},
			codeSnippet:  `sl_value = go_func("World")`,
			expectResult: int64(5),
			wantEqual:    true,
		},
		{
			name: "func(string) (string, error)",
			goFunc: func(name string) (string, error) {
				return "Hello " + name + "!", nil
			},
			codeSnippet:  `sl_value = go_func("World")`,
			expectResult: "Hello World!",
			wantEqual:    true,
		},
		{
			name: "func(string) (error, error)",
			goFunc: func(name string) (error, error) {
				return fmt.Errorf("need %s", name), nil
			},
			codeSnippet:  `sl_value = go_func("attention")`,
			expectResult: errors.New("need attention"),
			wantEqual:    true,
		},
		{
			name:        "fmt.Errorf",
			goFunc:      fmt.Errorf,
			codeSnippet: `sl_value = go_func("maybe an error")`,
			wantErrExec: true,
		},
		{
			name:         "fmt.Sprintf",
			goFunc:       fmt.Sprintf,
			codeSnippet:  `sl_value = go_func("Hello %s! #%d", "World", 42)`,
			expectResult: `Hello World! #42`,
			wantEqual:    true,
		},
		{
			name:         "strings.Repeat",
			goFunc:       strings.Repeat,
			codeSnippet:  `sl_value = go_func("Hello ", 3)`,
			expectResult: `Hello Hello Hello `,
			wantEqual:    true,
		},
		{
			name: "unsupported func(chan) int",
			goFunc: func(ch chan int) int {
				return <-ch
			},
			codeSnippet: `sl_value = go_func(42)`,
			wantErrExec: true,
		},
		{
			name: "unsupported func(int) chan",
			goFunc: func(size int) chan int {
				return make(chan int, size)
			},
			codeSnippet: `sl_value = go_func(42)`,
			wantErrExec: true,
		},
		{
			name: "mismatched func(int) string",
			goFunc: func(name int) string {
				return fmt.Sprintf("Hello %d!", name)
			},
			codeSnippet: `sl_value = go_func("42")`,
			wantErrExec: true,
		},
		{
			name: "fuzzy func(string) int",
			goFunc: func(name string) int {
				return len(name)
			},
			codeSnippet:  `sl_value = go_func(42)`,
			expectResult: int64(1),
			wantEqual:    true,
		},
		{
			name: "pointer as invalid argument: func(*string) string",
			goFunc: func(name *string) string {
				if name == nil {
					return "Hello World!"
				}
				return "Hello " + *name + "!"
			},
			codeSnippet: `sl_value = go_func("World")`,
			wantErrExec: true,
		},
		{
			name: "pointer as return: func(string) *string",
			goFunc: func(name string) *string {
				return &name
			},
			codeSnippet: `
sl_value = go_func("World")
print('※ sl_value: {}({})'.format(sl_value, type(sl_value)))
`,
			expectResult: "World",
			wantEqual:    true,
		},
		{
			name: "pointer as return for nil: func(string) *string",
			goFunc: func(name string) *string {
				return nil
			},
			codeSnippet: `sl_value = go_func("World")`,
			wantErrExec: true,
		},
		{
			name: "func([]string) (string)",
			goFunc: func(names []string) string {
				return strings.Join(names, ", ")
			},
			codeSnippet:  `sl_value = go_func(["Alice", "Bob", "Carol"])`,
			expectResult: "Alice, Bob, Carol",
			wantEqual:    true,
		},
		{
			name: "func([]int) string",
			goFunc: func(numbers []int8) int16 {
				x := int16(0)
				for _, n := range numbers {
					x += int16(n)
				}
				return x
			},
			codeSnippet:  `sl_value = go_func([1, 2, 3, 4, 5])`,
			expectResult: int64(15),
			wantEqual:    true,
		},
		{
			name: "func([5]int) int",
			goFunc: func(numbers [5]int) int {
				return numbers[0] + numbers[1] + numbers[2] + numbers[3] + numbers[4]
			},
			codeSnippet: `sl_value = go_func([1, 2, 3, 4, 5])`,
			wantErrExec: true, // TODO: support array as input
		},
		{
			name: "func([][]int) int",
			goFunc: func(numbers [][]int) int {
				x := 0
				for _, row := range numbers {
					for _, n := range row {
						x += n
					}
				}
				return x
			},
			codeSnippet: `sl_value = go_func([[1, 2, 3], [4, 5, 6]])`,
			wantErrExec: true, // TODO: support nested slice as input
		},
		{
			name: "func(map[string]int) int",
			goFunc: func(numbers map[string]int) int {
				x := 0
				for _, n := range numbers {
					x += n
				}
				return x
			},
			codeSnippet:  `sl_value = go_func({"a": 1, "b": 2, "c": 3})`,
			expectResult: int64(6),
			wantEqual:    true,
		},
		{
			name: "func(map[string]map[string]int) string",
			goFunc: func(numbers map[string]map[string]int) string {
				x := 0
				for _, row := range numbers {
					for _, n := range row {
						x += n
					}
				}
				return fmt.Sprintf("%d", x)
			},
			codeSnippet: `sl_value = go_func({"a": {"x": 1, "y": 2, "z": 3}, "b": {"x": 4, "y": 5, "z": 6}})`,
			wantErrExec: true, // TODO: support nested map as input
		},
		{
			name: "func(string) custom",
			goFunc: func(name string) customStruct {
				return customStruct{Name: name, Value: 42}
			},
			codeSnippet:  `sl_value = go_func("Alice")`,
			expectResult: customStruct{Name: "Alice", Value: 42},
			wantEqual:    true,
		},
		{
			name: "func(string) *custom",
			goFunc: func(name string) *customStruct {
				return &customStruct{Name: name, Value: 36}
			},
			codeSnippet:  `sl_value = go_func("Bob")`,
			expectResult: &customStruct{Name: "Bob", Value: 36},
			wantEqual:    true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			globals := map[string]interface{}{
				"go_func": tc.goFunc,
			}

			// convert go functions to Starlark values as predefined globals
			env, errConv := convert.MakeStringDict(globals)
			if errConv != nil {
				t.Fatalf(`expected no error while converting funcs, but got %v`, errConv)
			}

			// run the Starlark code to test the converted globals
			res, errExec := execStarlark(tc.codeSnippet, env)
			if errExec != nil && !tc.wantErrExec {
				t.Fatalf(`expected no error while executing code snippet, but got %v`, errExec)
			} else if errExec == nil && tc.wantErrExec {
				t.Fatalf(`expected an error while executing code snippet, but got none`)
			}
			if errExec != nil {
				return
			}

			// result value
			slValue, found := res["sl_value"]
			if !found {
				t.Fatalf(`expected sl_value in globals, but got none`)
			}

			// compare the result
			if gotEqual := reflect.DeepEqual(slValue, tc.expectResult); gotEqual != tc.wantEqual {
				t.Fatalf(`expected sl_value to be %v (%T), but got %v (%T), want equal: %v`, tc.expectResult, tc.expectResult, slValue, slValue, tc.wantEqual)
			}
		})
	}
}

// TestUseStarlarkValueInGo tests using various Starlark values in Go. It verifies:
// 1. the Starlark values can be converted to Go values as output;
// 2. the converted Go value can be used in Go code;
func TestUseStarlarkValueInGo(t *testing.T) {
	code := `
# Basic types
none = None
boolean = True
integer = 42
float_num = 3.14159
string = "Hello, Starlark!"

# Containers
tuple_val = (1, 2, 3)
list_val = [4, 5, 6]
dict_val = {"Alice": 1, "Bob": 2, "Charlie": 3}
set_val = set([1, 2, 3, 4, 5])
person = struct(name="John Doe", age=30, tags=["tag1", "tag2", "tag3"])

# Nested
nested_map = {"a": {"x": 1, "y": 2, "z": 3}, "b": {"x": 4, "y": 5, "z": 6}}
nested_list = [[1, 2, 3], [4, 5, 6]]
`
	envs := map[string]starlark.Value{
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
	}
	globals, err := execStarlark(code, envs)
	if err != nil {
		t.Fatalf(`expected no error, but got %v`, err)
	}

	// Basic types
	if none := globals["none"]; none != nil {
		t.Fatalf(`expected None to convert to nil, but got %v`, none)
	}

	if boolean := globals["boolean"].(bool); !boolean {
		t.Fatalf(`expected boolean to convert to true, but got %v`, boolean)
	}

	if integer := globals["integer"].(int64); integer != 42 {
		t.Fatalf(`expected integer to convert to 42, but got %v`, integer)
	}

	if floatNum := globals["float_num"].(float64); math.Abs(floatNum-3.14159) > 1e-5 {
		t.Fatalf(`expected float_num to convert to 3.14159, but got %v`, floatNum)
	}

	if str := globals["string"].(string); str != "Hello, Starlark!" {
		t.Fatalf(`expected string to convert to "Hello, Starlark!", but got %s`, str)
	}

	// Containers
	if tup := globals["tuple_val"].([]interface{}); !reflect.DeepEqual(tup, []interface{}{int64(1), int64(2), int64(3)}) {
		t.Fatalf(`expected tuple_val to convert to [1, 2, 3], but got %v`, tup)
	}

	if list := globals["list_val"].([]interface{}); !reflect.DeepEqual(list, []interface{}{int64(4), int64(5), int64(6)}) {
		t.Fatalf(`expected list_val to convert to [4, 5, 6], but got %v`, list)
	}

	actualDict := globals["dict_val"].(map[interface{}]interface{})
	expectedDict := map[interface{}]interface{}{"Alice": int64(1), "Bob": int64(2), "Charlie": int64(3)}
	if !reflect.DeepEqual(actualDict, expectedDict) {
		t.Fatalf(`expected actualDict to convert to %v, but got %v`, expectedDict, actualDict)
	}

	actualSet := globals["set_val"].(map[interface{}]bool)
	expectedSet := map[interface{}]bool{int64(1): true, int64(2): true, int64(3): true, int64(4): true, int64(5): true}
	if !reflect.DeepEqual(actualSet, expectedSet) {
		t.Fatalf(`expected set_val to convert to %v, but got %v`, expectedSet, actualSet)
	}

	if person := globals["person"].(interface{}); person == nil {
		t.Fatalf(`expected person to convert to a struct, but got nil`)
	} else {
		personStruct := person.(*starlarkstruct.Struct)
		t.Logf(`personStruct: %v`, personStruct)
		if name, _ := personStruct.Attr("name"); name.(starlark.String).GoString() != "John Doe" {
			t.Fatalf(`expected person.name to be "John Doe", but got %v`, name)
		}
	}

	if nestedMap := globals["nested_map"].(map[interface{}]interface{}); !reflect.DeepEqual(nestedMap, map[interface{}]interface{}{"a": map[interface{}]interface{}{"x": int64(1), "y": int64(2), "z": int64(3)}, "b": map[interface{}]interface{}{"x": int64(4), "y": int64(5), "z": int64(6)}}) {
		t.Fatalf(`expected nested_map to convert to {"a": {"x": 1, "y": 2, "z": 3}, "b": {"x": 4, "y": 5, "z": 6}}, but got %v`, nestedMap)
	}

	if nestedList := globals["nested_list"].([]interface{}); !reflect.DeepEqual(nestedList, []interface{}{[]interface{}{int64(1), int64(2), int64(3)}, []interface{}{int64(4), int64(5), int64(6)}}) {
		t.Fatalf(`expected nested_list to convert to [[1, 2, 3], [4, 5, 6]], but got %v`, nestedList)
	}
}

// TestCustomStruct tests that custom struct can be operated in Starlark.
func TestCustomStructInStarlark(t *testing.T) {
	t.Skip()

	getNewPerson := func() *personStruct {
		a := "Aloha!"
		b := bytes.NewBuffer(nil)
		p := &personStruct{
			Name:     "John Doe",
			Age:      30,
			Anything: []interface{}{false, 1, 2.0, "3", []int{1, 2, 3}, &a},
			Labels:   []string{"tag1", "tag2", "tag3"},
			Profile: map[string]interface{}{
				"email": "john@doe.me",
				"phone": int16(12345),
			},
			Parent: &personStruct{
				Name:      "Old John",
				Age:       58,
				secretKey: "secret_root",
			},
			NestedValues: map[string]map[int][]float32{
				"foo": {
					1: {1.1, 1.2, 1.3},
				},
				"bar": {
					2: {2.1, 2.2, 2.3},
				},
				"baz": {
					3: {3.1, 3.2, 3.3},
				},
				"das": {
					4: {4.1, 4.2, 4.3},
				},
			},
			secretKey: "secret_child",
			Customer: customStruct{
				Name:  "ACME",
				Value: 100,
			},
			CustomerPtr: &customStruct{
				Name:  "BDX",
				Value: 200,
			},
			MessageWriter: b,
			ReadMessage:   b.String,
			NumberChan:    make(chan int, 10),
			StarDict:      starlark.NewDict(10),
		}
		_ = p.StarDict.SetKey(starlark.String("foo"), starlark.String("bar"))
		return p
	}
	noCheck := func(_ *personStruct, _ map[string]interface{}) error {
		return nil
	}
	getInterfaceStringSliceCompare := func(fieldName string, want []string) func(*personStruct, map[string]interface{}) error {
		return func(pn *personStruct, res map[string]interface{}) error {
			if foo, ok := res[fieldName]; !ok {
				return fmt.Errorf(`expected %q to be in globals, but not found`, fieldName)
			} else if fs, ok := foo.([]interface{}); !ok {
				return fmt.Errorf(`expected %q to be a interface list, but got %v`, fieldName, foo)
			} else {
				if len(fs) != len(want) {
					return fmt.Errorf(`expected %q to be a list of %d, but got %d`, fieldName, len(want), len(fs))
				}
				for i, f := range fs {
					if s, ok := f.(string); !ok {
						return fmt.Errorf(`expected %q to be a list of string, but got %v`, fieldName, fs)
					} else if s != want[i] {
						return fmt.Errorf(`expected %q[%d] to be %q, but got %q`, fieldName, i, want[i], s)
					}
				}
				return nil
			}
		}
	}
	getStringSliceCompare := func(fieldName string, want []string) func(*personStruct, map[string]interface{}) error {
		return func(pn *personStruct, res map[string]interface{}) error {
			if foo, ok := res[fieldName]; !ok {
				return fmt.Errorf(`expected %q to be in globals, but not found`, fieldName)
			} else if fs, ok := foo.([]string); !ok {
				return fmt.Errorf(`expected %q to be a string list, but got %v`, fieldName, foo)
			} else {
				if len(fs) != len(want) {
					return fmt.Errorf(`expected %q to be %d elements, but got %d`, fieldName, len(want), len(fs))
				}
				for i, s := range fs {
					if s != want[i] {
						return fmt.Errorf(`expected %q[%d] to be %q, but got %q`, fieldName, i, want[i], s)
					}
				}
				return nil
			}
		}
	}
	getStringCompare := func(fieldName string, want string) func(*personStruct, map[string]interface{}) error {
		return func(_ *personStruct, m map[string]interface{}) error {
			if v, ok := m[fieldName]; !ok {
				return fmt.Errorf(`expected %q to be in globals, but not found`, fieldName)
			} else if n, ok := v.(string); !ok {
				return fmt.Errorf(`expected %q to be a string, but got %T`, fieldName, v)
			} else if n != want {
				return fmt.Errorf(`expected %q to be "%q, but got %q`, fieldName, want, n)
			}
			return nil
		}
	}

	type testCase struct {
		name        string
		codeSnippet string
		wantErrExec bool
		checkEqual  func(*personStruct, map[string]interface{}) error
	}
	testCases := []testCase{
		{
			name:        "noop",
			codeSnippet: `out = pn`,
			checkEqual:  noCheck,
		},
		{
			name:        "read non-exist field",
			codeSnippet: `foo = pn.NonExist`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "read private field",
			codeSnippet: `foo = pn.secretKey`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "write private field",
			codeSnippet: `pn.secretKey = "whatever"`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "access unsupported field",
			codeSnippet: `foo = pn.NumberChan`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "access unsupported field 2",
			codeSnippet: `foo = pn.NilString`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "assign mismatched type",
			codeSnippet: `pn.Parent = 88`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "assign mismatched type 2",
			codeSnippet: `pn.Age = "number"`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "read nil simple custom field",
			codeSnippet: `out = pn; val = pn.NilCustomer`,
			checkEqual: func(_ *personStruct, m map[string]interface{}) error {
				if v, ok := m["val"]; !ok {
					return fmt.Errorf(`expected "val" to be in globals, but not found`)
				} else if p, ok := v.(*customStruct); !ok {
					return fmt.Errorf(`expected "val" to be a pointer to customStruct, but got %T`, v)
				} else if p != nil {
					return fmt.Errorf(`expected "val" to be nil, but got %v`, v)
				}
				return nil
			},
		},
		{
			name:        "read nil custom field with methods",
			codeSnippet: `out = pn; val = pn.NilPerson`,
			checkEqual: func(_ *personStruct, m map[string]interface{}) error {
				if v, ok := m["val"]; !ok {
					return fmt.Errorf(`expected "val" to be in globals, but not found`)
				} else if p, ok := v.(*personStruct); !ok {
					return fmt.Errorf(`expected "val" to be a *personStruct, but got %T`, v)
				} else if p != nil {
					return fmt.Errorf(`expected "val" to be nil, but got %v`, v)
				}
				return nil
			},
		},
		{
			name:        "read public field",
			codeSnippet: `val = pn.Name ; out = pn`,
			checkEqual:  getStringCompare("val", "John Doe"),
		},
		{
			name:        "write public field",
			codeSnippet: `pn.Name = "Whoever"; out = pn`,
			checkEqual: func(pn *personStruct, _ map[string]interface{}) error {
				if pn.Name != "Whoever" {
					return fmt.Errorf(`expected pn.Name to be "Whoever", but got %q`, pn.Name)
				}
				return nil
			},
		},
		{
			name:        "use public method",
			codeSnippet: `pn.Aging(); out = pn`,
			checkEqual: func(pn *personStruct, _ map[string]interface{}) error {
				if pn.Age != 31 {
					return fmt.Errorf(`expected pn.Age to be 31, but got %v`, pn.Age)
				}
				return nil
			},
		},
		{
			name:        "list prop fields",
			codeSnippet: `fields = dir(pn); out = pn`,
			checkEqual:  getInterfaceStringSliceCompare("fields", []string{"Age", "Aging", "Anything", "Customer", "CustomerPtr", "GetSecretKey", "Labels", "MessageWriter", "Name", "NestedValues", "NilCustomer", "NilPerson", "NilString", "Nothing", "NumberChan", "Parent", "Profile", "ReadMessage", "SetCustomer", "SetSecretKey", "StarDict", "String", "buffer", "secretKey"}),
		},
		{
			name:        "read slice of string",
			codeSnippet: `foo = pn.Labels; out = pn`,
			checkEqual:  getStringSliceCompare("foo", []string{"tag1", "tag2", "tag3"}),
		},
		{
			name:        "read element of slice of string",
			codeSnippet: `foo = pn.Labels[1]; out = pn`,
			checkEqual:  getStringCompare("foo", "tag2"),
		},
		{
			name:        "set slice of string for wrong type", // It fails for []interface{} vs []string
			codeSnippet: `pn.Labels = ["foo", "bar"]; out = pn`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "set slice of interface",
			codeSnippet: `pn.Anything = ["foo", "bar"]; out = pn; sl = pn.Anything`,
			checkEqual:  getInterfaceStringSliceCompare("sl", []string{"foo", "bar"}),
		},
		{
			name:        "read element of slice of interface",
			codeSnippet: `foo = pn.Anything[3]; out = pn`,
			checkEqual:  getStringCompare("foo", "3"),
		},
		{
			name:        "change slice field",
			codeSnippet: `pn.Labels[0] = "bird"; out = pn`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				fieldName := ".Labels"
				want := []string{"bird", "tag2", "tag3"}
				if len(p.Labels) != len(want) {
					return fmt.Errorf(`expected %q to have %d elements, but got %d`, fieldName, len(want), len(p.Labels))
				}
				for i, s := range p.Labels {
					if s != want[i] {
						return fmt.Errorf(`expected %q[%d] to be %q, but got %q`, fieldName, i, want[i], s)
					}
				}
				return nil
			},
		},
		{
			name: "append slice field -- workaround", // It's a known issue that append() ops doesn't work on original slice field, since the new slice struct won't be set back.
			codeSnippet: `
l = pn.Labels
l.append("cat")
l.extend(["dog", "fish"])
l.pop()
l[0] = "bird"
l.insert(2, "whale")
pn.Labels = l

out = pn
`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				fieldName := ".Labels"
				want := []string{"bird", "tag2", "whale", "tag3", "cat", "dog"}
				if len(p.Labels) != len(want) {
					return fmt.Errorf(`expected %q to have %d elements, but got %d`, fieldName, len(want), len(p.Labels))
				}
				for i, s := range p.Labels {
					if s != want[i] {
						return fmt.Errorf(`expected %q[%d] to be %q, but got %q`, fieldName, i, want[i], s)
					}
				}
				return nil
			},
		},
		{
			name:        "read non-exist map field",
			codeSnippet: `foo = pn.Profile["name"]; out = pn`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "read number map field",
			codeSnippet: `foo = pn.Profile["phone"]; out = pn`,
			checkEqual: func(_ *personStruct, m map[string]interface{}) error {
				if v, ok := m["foo"]; !ok {
					return fmt.Errorf(`expected "foo" to be in globals, but not found`)
				} else if n, ok := v.(int16); !ok {
					return fmt.Errorf(`expected "foo" to be an int16, but got %T`, v)
				} else if n != 12345 {
					return fmt.Errorf(`expected "foo" to be 12345, but got %v`, n)
				}
				return nil
			},
		},
		{
			name:        "read string map field",
			codeSnippet: `foo = pn.Profile["email"]; out = pn`,
			checkEqual:  getStringCompare("foo", "john@doe.me"),
		},
		{
			name:        "add new map field",
			codeSnippet: `pn.Profile["foo"] = "bar"; out = pn`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				if v, ok := p.Profile["foo"]; !ok {
					return fmt.Errorf(`expected "foo" to be in Profile, but not found`)
				} else if n, ok := v.(string); !ok {
					return fmt.Errorf(`expected "foo" to be a string, but got %T`, v)
				} else if n != "bar" {
					return fmt.Errorf(`expected "foo" to be "bar", but got %v`, n)
				}
				return nil
			},
		},
		{
			name:        "change map field",
			codeSnippet: `pn.Profile["name"] = "Jane"; out = pn`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				if v, ok := p.Profile["name"]; !ok {
					return fmt.Errorf(`expected "name" to be in Profile, but not found`)
				} else if n, ok := v.(string); !ok {
					return fmt.Errorf(`expected "name" to be a string, but got %T`, v)
				} else if n != "Jane" {
					return fmt.Errorf(`expected "name" to be "Jane", but got %v`, n)
				}
				return nil
			},
		},
		{
			name:        "delete map field",
			codeSnippet: `pn.Profile.pop("email"); out = pn`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				if _, ok := p.Profile["email"]; ok {
					return fmt.Errorf(`expected "email" to be deleted from Profile, but still found`)
				}
				return nil
			},
		},
		{
			name:        "delete non-exist map field",
			codeSnippet: `pn.Profile.pop("name"); out = pn`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "read like javascript",
			codeSnippet: `out = pn; val = pn.Profile.email`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "read nested map field",
			codeSnippet: `out = pn; val = pn.NestedValues["foo"][1][2]`,
			checkEqual: func(_ *personStruct, m map[string]interface{}) error {
				if v, ok := m["val"]; !ok {
					return fmt.Errorf(`expected "val" to be in globals, but not found`)
				} else if n, ok := v.(float64); !ok {
					return fmt.Errorf(`expected "val" to be a float32, but got %T`, v)
				} else if math.Abs(n-1.3) > 0.0001 {
					return fmt.Errorf(`expected "val" to be 1.3, but got %v`, n)
				}
				return nil
			},
		},
		{
			name:        "change nested map field",
			codeSnippet: `out = pn; pn.NestedValues["foo"][1][1] = 111`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				v := p.NestedValues["foo"][1][1]
				if math.Abs(float64(v-111)) > 0.0001 {
					return fmt.Errorf(`expected "foo" to be 111, but got %v`, v)
				}
				return nil
			},
		},
		{
			name: "append nested map field workaround",
			codeSnippet: `
nv = pn.NestedValues["foo"][1]
nv.append(123)
pn.NestedValues["foo"][1] = nv
out = pn
`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				v := p.NestedValues["foo"][1][3]
				if math.Abs(float64(v-123)) > 0.0001 {
					return fmt.Errorf(`expected "foo" to be 111, but got %v`, v)
				}
				return nil
			},
		},
		{
			name:        "set nested map field for wrong type",
			codeSnippet: `pn.NestedValues["foo"][1] = [1, 2, 3]; out = pn`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "read nested struct",
			codeSnippet: `out = pn; val = pn.Parent.Name`,
			checkEqual:  getStringCompare("val", "Old John"),
		},
		{
			name:        "change nested struct",
			codeSnippet: `out = pn; pn.Parent.Name = "New John"`,
			checkEqual: func(p *personStruct, _ map[string]interface{}) error {
				if p.Parent.Name != "New John" {
					return fmt.Errorf(`expected "Name" to be "New John", but got %v`, p.Parent.Name)
				}
				return nil
			},
		},
		{
			name:        "read nested struct nil field",
			codeSnippet: `out = pn; val = pn.Parent.Parent`,
			checkEqual: func(_ *personStruct, m map[string]interface{}) error {
				if v, ok := m["val"]; !ok {
					return fmt.Errorf(`expected "val" to be in globals, but not found`)
				} else if p, ok := v.(*personStruct); !ok {
					return fmt.Errorf(`expected "val" to be a *personStruct, but got %T`, v)
				} else if p != nil {
					return fmt.Errorf(`expected "val" to be nil, but got %v`, p)
				}
				return nil
			},
		},
		{
			name:        "access to nil person method",
			codeSnippet: `out = pn; val = pn.NilPerson.Nothing()`,
			checkEqual:  getStringCompare("val", "nothing"),
		},
		{
			name:        "invalid access to nil person method",
			codeSnippet: `out = pn; val = pn.NilPerson.Aging()`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "invalid access to nil simple custom field",
			codeSnippet: `out = pn; val = pn.NilCustomer.Name`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "invalid access to nil person field",
			codeSnippet: `out = pn; val = pn.NilPerson.Name`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name:        "invalid access to nested struct nil field",
			codeSnippet: `out = pn; val = pn.Parent.Parent.Name`,
			checkEqual:  noCheck,
			wantErrExec: true,
		},
		{
			name: "call interface method",
			codeSnippet: `
out = pn
num = pn.MessageWriter.Write("Mahalo!")
print("wrote", num, "bytes")
val = pn.ReadMessage()
`,
			checkEqual: getStringCompare("val", "Mahalo!"),
		},
		{
			name:        "use original starlark dict",
			codeSnippet: `out = pn; pn.StarDict["Hello"] = [42, 1000]; val = pn.StarDict["Hello"]`,
			checkEqual: func(p *personStruct, m map[string]interface{}) error {
				// check exported value
				if v, ok := m["val"]; !ok {
					return fmt.Errorf(`missing "val" in globals`)
				} else if p, ok := v.([]interface{}); !ok {
					return fmt.Errorf("mistype for val, want: []interface{}, got: %T", p)
				} else {
					vals := []int64{42, 1000}
					if len(vals) != len(p) {
						return fmt.Errorf("diff length of slice: want: %d, got: %d", len(vals), len(p))
					}
					for i := range vals {
						e := vals[i]
						a := p[i].(int64)
						if e != a {
							return fmt.Errorf("diff value of element[%d]: want: %d, got: %d", i, e, a)
						}
					}
				}

				// check original value
				name := "Hello"
				if v, found, err := p.StarDict.Get(starlark.String(name)); err != nil {
					return fmt.Errorf("fail to get value %q from dict: %v", name, err)
				} else if !found {
					return fmt.Errorf("target value %q is missing from dict", name)
				} else if v.Type() != "list" {
					return fmt.Errorf("got wrong value type: want: list, got: %s", v.Type())
				}
				return nil
			},
		},
		{
			name: "Test!!!!",
			codeSnippet: `
print(pn)
print(dir(pn))
out = pn
`,
			checkEqual: func(pn *personStruct, _ map[string]interface{}) error {
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare the environment
			raw := getNewPerson()
			envs, err := convert.MakeStringDict(map[string]interface{}{
				"pn": raw,
			})
			if err != nil {
				t.Fatalf(`failed to make string dict: %v`, err)
			}

			// Execute the code and check error
			globals, err := execStarlark(tc.codeSnippet, envs)
			if tc.wantErrExec {
				if err == nil {
					t.Fatalf(`expected error, but got nil`)
				}
				return
			}
			if err != nil {
				t.Fatalf(`failed to exec starlark: %v`, err)
			}

			// Check the result
			if pn := globals["out"].(*personStruct); pn == nil {
				t.Fatalf(`expected pn to convert to a struct, but got nil`)
			} else {
				if pn != raw {
					t.Fatalf(`expected pn to be equal to the original personStruct`)
				}
				if err := tc.checkEqual(pn, globals); err != nil {
					t.Fatalf(`expected pn to be equal to the original personStruct, but got error: %v`, err)
				}
			}
		})
	}
}

type customStruct struct {
	Name  string
	Value int
}

//func (c *customStruct) String() string {
//	return fmt.Sprintf("Custom[%s|%d]", c.Name, c.Value)
//}

type personStruct struct {
	Name          string                       `starlark:"name"`
	Age           int                          `starlark:"age"`
	Anything      []interface{}                `starlark:"anything"`
	Labels        []string                     `starlark:"tags"`
	Profile       map[string]interface{}       `starlark:"profile"`
	Parent        *personStruct                `starlark:"parent"`
	NestedValues  map[string]map[int][]float32 `starlark:"nested_values"`
	secretKey     string                       // unexported field
	Customer      customStruct                 `starlark:"customer"`
	CustomerPtr   *customStruct                `starlark:"customer_ptr"`
	MessageWriter io.Writer                    `starlark:"message_writer"`
	ReadMessage   func() string                `starlark:"read_message"`
	NumberChan    chan int                     `starlark:"number_chan"`
	NilString     *string                      `starlark:"nil_string"`
	NilCustomer   *customStruct                `starlark:"nil_custom"`
	NilPerson     *personStruct                `starlark:"nil_person"`
	buffer        bytes.Buffer                 `starlark:"-"`
	StarDict      *starlark.Dict               `starlark:"dict"`
}

func (p *personStruct) String() string {
	return fmt.Sprintf("Person[%s,%d]<%s>", p.Name, p.Age, p.secretKey)
}

func (p *personStruct) secretMethod() string {
	return p.secretKey
}

func (p *personStruct) SetSecretKey(key string) {
	p.secretKey = key
}

func (p *personStruct) GetSecretKey() string {
	return p.secretKey
}

func (p *personStruct) SetCustomer(customer customStruct) {
	p.Customer = customer
}

func (p *personStruct) Aging() {
	p.Age++
}

func (p *personStruct) Nothing() string {
	return "nothing"
}
