package skyhook

import (
	"reflect"
	"testing"
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
