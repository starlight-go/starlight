package starlight

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConversion(t *testing.T) {
	dir, cleanup := makeScript(t, "out.star",
		`output = input + " world" + bang()`)

	defer cleanup()

	bang := func() string { return "!" }

	s := New(dir)
	actual, err := s.Run("out.star", map[string]interface{}{
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
	s := New("testdata", "testdata/later")
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
	s := New("testdata", "testdata/later")
	v, err := s.Run("bar.sky", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	expected, actual := "hello from bar.sky", v["output"]
	if actual != expected {
		t.Fatalf("expected %q but got %q", expected, actual)
	}
}

func TestRerun(t *testing.T) {
	dir, cleanup := makeScript(t, "foo.star",
		`output = input + " world!"`)

	defer cleanup()

	s := New(dir)

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
	err = ioutil.WriteFile(filepath.Join(dir, "foo.star"), []byte(`output = "hi!"`), 0600)
	if err != nil {
		t.Fatal(err)
	}

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
	err = ioutil.WriteFile(filepath.Join(dir, "foo.star"), []byte(`output = "bye!"`), 0600)
	if err != nil {
		t.Fatal(err)
	}

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
	v, err := Eval([]byte(`output = hi()`), map[string]interface{}{
		"hi": func() string { return "hi!" },
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v["output"] != "hi!" {
		t.Fatalf(`expected "hi!" but got %q`, v["output"])
	}
}

func makeScript(t *testing.T, name, data string) (dir string, cleanup func()) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(dir, name)
	t.Logf("creating new script at %v", filename)
	err = ioutil.WriteFile(filename, []byte(data), 0600)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}
