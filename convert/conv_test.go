package convert

import (
	"fmt"
	"reflect"
	"testing"

	"go.starlark.net/starlark"
)

func TestKwargs(t *testing.T) {
	// Mental note: starlark numbers pop out as int64s
	data := []byte(`
func("a", 1, foo=1, foo=2)
`)

	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	var goargs []interface{}
	var gokwargs []Kwarg

	fn := func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var err error
		goargs, err = FromTuple(args)
		if err != nil {
			return starlark.None, err
		}
		gokwargs, err = FromKwargs(kwargs)
		if err != nil {
			return starlark.None, err
		}
		return starlark.None, nil
	}

	globals := map[string]starlark.Value{
		"func": starlark.NewBuiltin("func", fn),
	}
	globals, err := starlark.ExecFile(thread, "foo.star", data, globals)
	if err != nil {
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

func TestMakeSkyFn(t *testing.T) {
	fn := func(s string, i int64, b bool, f float64) (int, string, error) {
		return 5, "hi!", nil
	}

	skyf := MakeSkyFn("boo", fn)
	// Mental note: starlark numbers pop out as int64s
	data := []byte(`
a = boo("a", 1, True, 0.1)
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
	v := FromStringDict(globals)
	if !reflect.DeepEqual(v["a"], []interface{}{int64(5), "hi!"}) {
		t.Fatalf(`expected a = [5, "hi"], but got %#v`, v)
	}
}

func TestMakeSkyFnOneRet(t *testing.T) {
	fn := func(s string) string {
		return "hi " + s
	}

	skyf := MakeSkyFn("boo", fn)
	// Mental note: starlark numbers pop out as int64s
	data := []byte(`
a = boo("skyhook")
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
	v := FromStringDict(globals)
	if v["a"] != "hi skyhook" {
		t.Fatalf(`expected a = "hi skyhook", but got %#v`, v["a"])
	}
}
