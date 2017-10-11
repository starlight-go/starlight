package skyhook

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/skylark"
)

func TestConversion(t *testing.T) {
	data := []byte(`output = input + " world" + bang()`)

	read := func(string) ([]byte, error) { return data, nil }

	bang := func() string { return "!" }

	s := New([]string{"bar"})
	s.readFile = read
	actual, err := s.Run("foo.sky", map[string]interface{}{
		"input": "hello",
		"bang":  bang,
	})
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

	read := func(string) ([]byte, error) { return data, nil }

	s := New([]string{"bar"})
	s.readFile = read
	actual, err := s.Run("foo.sky", map[string]interface{}{"input": "hello"})
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

func TestMakeSkyFn(t *testing.T) {
	fn := func(s string, i int64, b bool, f float64) (int, string, error) {
		return 5, "hi!", nil
	}

	skyf := MakeSkyFn("boo", fn)
	// Mental note: skylark numbers pop out as int64s
	data := []byte(`
a = boo("a", 1, True, 0.1)
b = 0.1
	`)

	thread := &skylark.Thread{
		Print: func(_ *skylark.Thread, msg string) { fmt.Println(msg) },
	}

	globals := map[string]skylark.Value{
		"boo": skyf,
	}
	if err := skylark.ExecFile(thread, "foo.sky", data, globals); err != nil {
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
	// Mental note: skylark numbers pop out as int64s
	data := []byte(`
a = boo("skyhook")
`)

	thread := &skylark.Thread{
		Print: func(_ *skylark.Thread, msg string) { fmt.Println(msg) },
	}

	globals := map[string]skylark.Value{
		"boo": skyf,
	}
	if err := skylark.ExecFile(thread, "foo.sky", data, globals); err != nil {
		t.Fatal(err)
	}
	v := FromStringDict(globals)
	if v["a"] != "hi skyhook" {
		t.Fatalf(`expected a = "hi skyhook", but got %#v`, v["a"])
	}
}

func TestRerun(t *testing.T) {
	data := []byte(`output = input + " world!"`)

	read := func(string) ([]byte, error) { return data, nil }

	s := New([]string{"bar"})
	s.readFile = read
	actual, err := s.Run("foo.sky", map[string]interface{}{
		"input": "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "hello world!" {
		t.Fatalf(`expected "hello world!" but got %q`, actual["output"])
	}

	// change inputs but not script
	actual, err = s.Run("foo.sky", map[string]interface{}{
		"input": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "goodbye world!" {
		t.Fatalf(`expected "goodbye world!" but got %q`, actual["output"])
	}

	// change script, shouldn't change output sicne we cached it
	data = []byte(`output = "hi!"`)
	actual, err = s.Run("foo.sky", map[string]interface{}{
		"input": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "goodbye world!" {
		t.Fatalf(`expected "goodbye world!" but got %q`, actual["output"])
	}

	// remove script, should change output
	s.Forget("foo.sky")
	actual, err = s.Run("foo.sky", map[string]interface{}{
		"input": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "hi!" {
		t.Fatalf(`expected "hi!" but got %q`, actual["output"])
	}

	// reset all, should change output
	s.Reset()
	data = []byte(`output = "bye!"`)
	actual, err = s.Run("foo.sky", map[string]interface{}{
		"input": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "bye!" {
		t.Fatalf(`expected "bye!" but got %q`, actual["output"])
	}

}
