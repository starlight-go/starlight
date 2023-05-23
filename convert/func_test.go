package convert_test

import (
	"errors"
	"fmt"
	"reflect"
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

func TestMakeStarFnOneRet(t *testing.T) {
	fn := func(s string) string {
		return "hi " + s
	}

	skyf := convert.MakeStarFn("boo", fn)
	// Mental note: starlark numbers pop out as int64s
	data := []byte(`
a = boo("starlight")
`)

	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	globals := map[string]starlark.Value{
		"boo": skyf,
	}
	globals, err := starlark.ExecFile(thread, "foo.star", data, globals)
	if err != nil {
		t.Fatal(err)
	}
	v := convert.FromStringDict(globals)
	if v["a"] != "hi starlight" {
		t.Fatalf(`expected a = "hi starlight", but got %#v`, v["a"])
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

func TestMakeStarFnOneRetErrorAndNonError(t *testing.T) {
	fn := func(s string) (error, string) {
		if s == "" {
			return errors.New("input is empty"), ""
		}
		return nil, "hi " + s
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	if v, err := execStarlark(`e, a = boo("")`, globals); err != nil {
		t.Fatalf(`expected a = "", err = "input is empty", but got a=%v, err=%v`, v, err)
	} else if e := v["e"]; e == nil {
		t.Fatalf(`expected e = "input is empty", but got e=nil`)
	}
}

func TestMakeStarFnOneRetTwoNonErrorAndError(t *testing.T) {
	fn := func(s string, n int) (string, int, error) {
		if s == "" {
			return "", 0, errors.New("input is empty")
		}
		return "hi " + s, n + 5, nil
	}

	skyf := convert.MakeStarFn("boo", fn)

	globals := map[string]starlark.Value{
		"boo": skyf,
	}

	if v, err := execStarlark(`a, b = boo("", 5)`, globals); err == nil {
		t.Fatalf(`expected a = "", b = 0, err = "input is empty", but got a=%v, b=%v, err=%v`, v["a"], v["b"], err)
	}
	if v, err := execStarlark(`a, b = boo("starlight", 5)`, globals); err != nil {
		t.Fatalf(`expected a = "hi starlight", b = 10, err = nil, but got a=%v, b=%v, err=%v`, v["a"], v["b"], err)
	}
}

func TestMakeStarFnSlice(t *testing.T) {
	fn := func(s1 []string, s2 []int) (int, string, error) {
		cnt := 10
		if len(s1) != 2 || s1[0] != "hello" || s1[1] != "world" {
			return 0, "", errors.New("incorrect slice input1")
		}
		if len(s2) != 2 || s2[0] != 1 || s2[1] != 2 {
			return 0, "", errors.New("incorrect slice input2")
		}

		// TODO: nested slice like [["slice", "test"], ["hello", "world"]], [[[1, 2]]]) is not supported yet
		return cnt, "hey!", nil
	}

	skyf := convert.MakeStarFn("boo", fn)

	data := []byte(`
a = boo(["hello", "world"], [1, 2])
b = 0.1
    `)

	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	globals := map[string]starlark.Value{
		"boo": skyf,
	}
	globals, err := starlark.ExecFile(thread, "foo.star", data, globals)
	if err != nil {
		t.Fatal(err)
	}
	v := convert.FromStringDict(globals)
	if !reflect.DeepEqual(v["a"], []interface{}{int64(10), "hey!"}) {
		t.Fatalf(`expected a = [10, "hey!"], but got %#v`, v)
	}
}

func TestMakeStarFnMap(t *testing.T) {
	fn := func(m1 map[string]int32, m2 map[string]int, m3 map[string]float32, m4 map[uint8]uint64, m5 map[int16]int8) (int, string, error) {
		cnt := int32(0)
		for k, v := range m1 {
			if k == "hello" && v == 1 {
				cnt += v
			} else if k == "world" && v == 2 {
				cnt += v
			} else {
				return 0, "", errors.New("incorrect map input1")
			}
		}
		for range m2 {
			cnt += 1
		}
		for range m3 {
			cnt += 1
		}
		for range m4 {
			cnt += 1
		}
		for range m5 {
			cnt += 1
		}

		// TODO: nested map like map[int16][]int8 {1000: [1, 2, 3]} is not supported yet
		return int(cnt), "hey!", nil
	}

	skyf := convert.MakeStarFn("boo", fn)

	data := []byte(`
a = boo({"hello": 1, "world": 2}, {"int": 100}, {"float32": 0.1}, {10: 5}, {1000: 100})
b = 0.1
    `)

	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	globals := map[string]starlark.Value{
		"boo": skyf,
	}
	globals, err := starlark.ExecFile(thread, "foo.star", data, globals)
	if err != nil {
		t.Fatal(err)
	}
	v := convert.FromStringDict(globals)
	if !reflect.DeepEqual(v["a"], []interface{}{int64(7), "hey!"}) {
		t.Fatalf(`expected a = [7, "hey!"], but got %#v`, v)
	}
}
