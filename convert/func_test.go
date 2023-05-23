package convert_test

import (
	"fmt"
	"testing"

	"github.com/1set/starlight"
	"github.com/1set/starlight/convert"
	"go.starlark.net/starlark"
)

func TestVariadic(t *testing.T) {
	globals := map[string]interface{}{
		"sprint":  fmt.Sprint,
		"fatal":   t.Fatal,
		"sprintf": fmt.Sprintf,
	}

	code := []byte(`
def do(): 
	v = sprint(False)
	if v != "false" :
		fatal("unexpected output: ", v)
	v = sprint(False, 1)
	if v != "false 1" :
		fatal("unexpected output:", v)
	v = sprint(False, 1, " hi ", {"key":"value"})
	if v != 'false 1 hi map[key:"value"]' :
		fatal("unexpected output:", v)
	v = sprintf("this is your %dst formatted message", 1)
	if v != "this is your 1st formatted message":
		fatal("unexpected output:", v)
do()
`)

	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// Helper function to execute a Starlark script with given global functions and data
func execStarlark(script string, globals map[string]starlark.Value) (map[string]interface{}, error) {
	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	data := []byte(script)
	globals, err := starlark.ExecFile(thread, "foo.star", data, globals)
	if err != nil {
		return nil, err
	}

	return convert.FromStringDict(globals), nil
}

// Test a function with no return value
func TestMakeStarFnNoRet(t *testing.T) {
	fn := func(s string) {
		fmt.Println("hi " + s)
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	_, err := execStarlark(`boo("starlight")`, globals)
	if err != nil {
		t.Fatal(err)
	}
}

// Test a function with one non-error return value
func TestMakeStarFnOneRetNonError(t *testing.T) {
	fn := func(s string) string {
		return "hi " + s
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	v, err := execStarlark(`a = boo("starlight")`, globals)
	if err != nil {
		t.Fatal(err)
	}

	if v["a"] != "hi starlight" {
		t.Fatalf(`expected a = "hi starlight", but got %#v`, v["a"])
	}
}

// Test a function with one error return value
func TestMakeStarFnOneRetError(t *testing.T) {
	fn := func(s string) error {
		if s == "error" {
			return fmt.Errorf("error occurred")
		}
		return nil
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	if _, err := execStarlark(`err = boo("wtf")`, globals); err != nil {
		t.Fatalf(`expected no err, but got err: %v`, err)
	}
	if v, err := execStarlark(`err = boo("error")`, globals); err == nil {
		t.Fatalf(`expected err = "error occurred", but got no err: %v`, v)
	}
}

// Test a function with two non-error return values
func TestMakeStarFnTwoRetNonError(t *testing.T) {
	fn := func(s string) (string, string) {
		return "hi " + s, "bye " + s
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	v, err := execStarlark(`a, b = boo("starlight")`, globals)
	if err != nil {
		t.Fatal(err)
	}

	if v["a"] != "hi starlight" || v["b"] != "bye starlight" {
		t.Fatalf(`expected a = "hi starlight", b = "bye starlight", but got a=%#v, b=%#v`, v["a"], v["b"])
	}
}

// Test a function with one non-error return value and one error return value
func TestMakeStarFnOneRetNonErrorAndError(t *testing.T) {
	fn := func(s string) (string, error) {
		if s == "error" {
			return "", fmt.Errorf("error occurred")
		}
		return "hi " + s, nil
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	if v, err := execStarlark(`a = boo("starlight")`, globals); err != nil {
		t.Fatalf(`expected a = "hi starlight", err = nil, but got a=%v, err=%v`, v, err)
	}
	if v, err := execStarlark(`a = boo("error")`, globals); err == nil {
		t.Fatalf(`expected err = "error occurred", but got no err: a=%v`, v)
	}
}
