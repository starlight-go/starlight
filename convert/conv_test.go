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
func("a", 1, foo=1, bar=2)
`)

	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	var goargs []interface{}
	var gokwargs []Kwarg

	fn := func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var err error
		goargs = FromTuple(args)
		gokwargs, err = FromKwargs(kwargs)
		if err != nil {
			return starlark.None, err
		}
		return starlark.None, nil
	}

	globals := map[string]starlark.Value{
		"func": starlark.NewBuiltin("func", fn),
	}
	_, err := starlark.ExecFile(thread, "foo.star", data, globals)
	if err != nil {
		t.Fatal(err)
	}

	expArgs := []interface{}{"a", int64(1)}
	if len(expArgs) != len(goargs) {
		t.Fatalf("expected %d args, but got %d", len(expArgs), len(goargs))
	}
	expKwargs := []Kwarg{{Name: "foo", Value: int64(1)}, {Name: "bar", Value: int64(2)}}

	if !reflect.DeepEqual(expArgs, goargs) {
		t.Errorf("expected args %#v, got args %#v", expArgs, goargs)
	}

	if !reflect.DeepEqual(expKwargs, gokwargs) {
		t.Fatalf("expected kwargs %#v, but got %#v", expKwargs, gokwargs)
	}
}

func TestMakeStarFn(t *testing.T) {
	fn := func(s string, i int64, b bool, f float64) (int, string, error) {
		return 5, "hi!", nil
	}

	skyf := MakeStarFn("boo", fn)
	// Mental note: starlark numbers pop out as int64s
	data := []byte(`
a = boo("a", 1, True, 0.1)
b = 0.1
	`)

	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) { fmt.Println(msg) },
	}

	envs := map[string]starlark.Value{
		"boo": skyf,
	}
	globals, err := starlark.ExecFile(thread, "foo.star", data, envs)
	if err != nil {
		t.Fatal(err)
	}
	v := FromStringDict(globals)
	if !reflect.DeepEqual(v["a"], []interface{}{int64(5), "hi!"}) {
		t.Fatalf(`expected a = [5, "hi!"], but got %#v`, v)
	}
}

func TestStructToValue(t *testing.T) {
	type contact struct {
		Name, Street string
	}
	c := &contact{Name: "bob", Street: "oak"}

	s := NewStruct(c)
	v, err := ToValue(s)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := v.(*GoStruct)
	if !ok {
		t.Fatalf("expected v to be *Struct, but was %T", v)
	}
}

func TestMakeNamedList(t *testing.T) {
	type Strings []string
	v := Strings{"foo", "bar"}
	val, err := ToValue(v)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := val.(*GoSlice)
	if !ok {
		t.Fatalf("value should be *GoSlice but was %T", val)
	}
}

type contact struct {
	Name        string
	age         int
	PhoneNumber string `starlark:"phone"`
	Secret      int    `starlark:"-"`
}

func (c contact) Foo()  {}
func (c *contact) Bar() {}

// reflect can't find non-exported functions... but can find non-exported
// methods ¯\_(ツ)_/¯

func (c *contact) bar() {}
func (c contact) foo()  {}

func TestStructAttrNames(t *testing.T) {
	c := &contact{}
	s := NewStruct(c)
	names := s.AttrNames()
	expected := []string{"Name", "Foo", "phone", "Bar"}
	for _, s := range names {
		if !contains(expected, s) {
			t.Errorf("output contains extra value %q", s)
		}
	}
	for _, s := range expected {
		if !contains(names, s) {
			t.Errorf("output is missing value %q", s)
		}
	}
	t.Logf("%q", names)
}

func contains(list []string, s string) bool {
	for _, l := range list {
		if s == l {
			return true
		}
	}
	return false
}
