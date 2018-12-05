package starlight

import (
	"reflect"
	"testing"
)

func TestConversion(t *testing.T) {
	data := []byte(`output = input + " world" + bang()`)

	read := func(string) ([]byte, error) { return data, nil }

	bang := func() string { return "!" }

	s := New([]string{"bar"})
	s.readFile = read
	actual, err := s.Run("foo.star", map[string]interface{}{
		"input": "hello",
		"bang":  bang,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{
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
	v, err := s.Run("foo.star", map[string]interface{}{"input": "hello"})
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
	actual, err := s.Run("foo.star", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	v, ok := actual["output"]
	if !ok {
		t.Fatal("missing output value")
	}
	act, ok := v.(string)
	if !ok {
		t.Fatalf("output should be string but was %T", v)
	}
	if act != "hello world!" {
		t.Fatalf("expected hello world but got %q", act)
	}
}

func TestRerun(t *testing.T) {
	data := []byte(`output = input + " world!"`)

	read := func(string) ([]byte, error) { return data, nil }

	s := New([]string{"bar"})
	s.readFile = read
	actual, err := s.Run("foo.star", map[string]interface{}{
		"input": "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "hello world!" {
		t.Fatalf(`expected "hello world!" but got %q`, actual["output"])
	}

	// change inputs but not script
	actual, err = s.Run("foo.star", map[string]interface{}{
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
	actual, err = s.Run("foo.star", map[string]interface{}{
		"input": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "goodbye world!" {
		t.Fatalf(`expected "goodbye world!" but got %q`, actual["output"])
	}

	// remove script, should change output
	s.Forget("foo.star")
	actual, err = s.Run("foo.star", map[string]interface{}{
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
	actual, err = s.Run("foo.star", map[string]interface{}{
		"input": "goodbye",
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual["output"] != "bye!" {
		t.Fatalf(`expected "bye!" but got %q`, actual["output"])
	}

}

func TestEval(t *testing.T) {
	v, err := Eval(`output = hi()`, map[string]interface{}{
		"hi": func() string { return "hi!" },
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v["output"] != "hi!" {
		t.Fatalf(`expected "hi!" but got %q`, v["output"])
	}
}
