package skyhook

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/skylark"
)

func TestConversion(t *testing.T) {
	data := []byte(`
output = input + " world"
`)

	actual, err := Skyhook{}.exec("foo.sky", data, map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{
		"input":  "hello",
		"output": "hello world",
	}
	if len(actual) != len(expected) {
		t.Errorf("expected %d items, but got %d", len(expected), len(actual))
	}
	for k, v := range expected {
		act, ok := actual[k]
		if !ok {
			t.Errorf("actual missing key %q", k)
			continue
		}
		if !reflect.DeepEqual(act, v) {
			t.Errorf("actual value for key %q expected to be %v but was %v", k, v, act)
		}
	}
}

func TestDirOrder(t *testing.T) {
	s := New([]string{"testdata", "testdata/later"})
	v, err := s.Run("foo.sky", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	expected, actual := "hello world", v["output"]
	if actual != expected {
		t.Fatalf("expected %q but got %q", expected, actual)
	}
}

func TestAllDirs(t *testing.T) {
	s := New([]string{"testdata", "testdata/later"})
	v, err := s.Run("bar.sky", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	expected, actual := "hello from bar.sky", v["output"]
	if actual != expected {
		t.Fatalf("expected %q but got %q", expected, actual)
	}
}

func TestFunc(t *testing.T) {
	data := []byte(`
def foo():
	return " world!"

output = input + foo()
`)

	actual, err := Skyhook{}.exec("foo.sky", data, map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{
		"input":  "hello",
		"output": "hello world!",
	}
	if len(actual) != len(expected) {
		t.Errorf("expected %d items, but got %d", len(expected), len(actual))
	}
	for k, v := range expected {
		act, ok := actual[k]
		if !ok {
			t.Errorf("actual missing key %q", k)
			continue
		}
		if !reflect.DeepEqual(act, v) {
			t.Errorf("actual value for key %q expected to be %v but was %v", k, v, act)
		}
	}
}

func TestKwargs(t *testing.T) {
	// Mental note: skylark numbers pop out as int64s
	data := []byte(`
func("a", 1, foo=1, foo=2)
`)

	thread := &skylark.Thread{
		Print: func(_ *skylark.Thread, msg string) { fmt.Println(msg) },
	}

	var goargs []interface{}
	var gokwargs []Kwarg

	fn := func(thread *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		var err error
		goargs, err = FromTuple(args)
		if err != nil {
			return skylark.None, err
		}
		gokwargs, err = FromKwargs(kwargs)
		if err != nil {
			return skylark.None, err
		}
		return skylark.None, nil
	}

	globals := map[string]skylark.Value{
		"func": skylark.NewBuiltin("func", fn),
	}
	if err := skylark.ExecFile(thread, "foo.sky", data, globals); err != nil {
		t.Fatal(err)
	}

	expArgs := []interface{}{"a", int64(1)}
	if len(expArgs) != len(goargs) {
		t.Fatalf("expected %d args, but got %d", len(expArgs), len(goargs))
	}
	expKwargs := []Kwarg{{Name: "foo", Value: int64(1)}, {Name: "foo", Value: int64(2)}}

	if !reflect.DeepEqual(expArgs, goargs) {
		t.Errorf("expected args %#v, got args %#v", expArgs, goargs)
	}

	if !reflect.DeepEqual(expKwargs, gokwargs) {
		t.Fatalf("expected kwargs %#v, but got %#v", expKwargs, gokwargs)
	}
}
