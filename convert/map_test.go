package convert_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/starlight-go/starlight/convert"

	"github.com/starlight-go/starlight"
	"go.starlark.net/starlark"
)

type assert struct {
	t *testing.T
}

func (a *assert) Eq(expected, got interface{}) {
	if !reflect.DeepEqual(expected, got) {
		a.t.Fatalf("expected %#v (%T) to be equal to %#v (%T)", expected, expected, got, got)
	}
}

// the majority of these tests mimic starlark-go's
// https://github.com/google/starlark-go/blob/master/starlark/testdata/dict.star

func TestMapPop(t *testing.T) {
	x6 := map[string]int{"a": 1, "b": 2}
	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x6":     x6,
	}

	code := []byte(`
assert.Eq(x6.pop("a"), 1)
assert.Eq(len(x6), 1)
assert.Eq(x6["b"], 2)
assert.Eq(x6.pop("c", 3), 3)
assert.Eq(x6.pop("c", None), None) # default=None tests an edge case of UnpackArgs
assert.Eq(x6.pop("b"), 2)
assert.Eq(len(x6), 0)
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(x6) != 0 {
		t.Fatalf("expected len of map to be 0, but was %d", len(x6))
	}

	code = []byte(`
x6.pop("c")
`)
	_, err = starlight.Eval(code, globals, nil)
	expectErr(t, err, "pop: missing key")
}

func TestMapPopItem(t *testing.T) {
	x7 := map[string]int{"a": 1, "b": 2}
	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x7":     x7,
	}

	code := []byte(`
assert.Eq([x7.popitem(), x7.popitem()], [("a", 1), ("b", 2)])
assert.Eq(len(x7), 0)
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(x7) != 0 {
		t.Fatalf("expected len of map to be 0, but was %d", len(x7))
	}

	code = []byte(`
a = x7.popitem()
`)
	_, err = starlight.Eval(code, globals, nil)
	expectErr(t, err, "popitem: empty dict")
}

func TestMapKeysValues(t *testing.T) {
	x8 := map[string]int{"a": 1, "b": 2}
	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x8":     x8,
	}

	code := []byte(`
# dict.keys, dict.values
assert.Eq(x8.keys(), ["a", "b"])
assert.Eq(x8.values(), [1, 2])
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(x8) != 2 {
		t.Fatalf("expected no values to be removed, but contents ended with %v", x8)
	}
}

// toMap converts from a starlark-created map to a map[string]int
func toMap(m map[interface{}]interface{}) (map[string]int, error) {
	out := map[string]int{}
	for k, v := range m {
		s, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, but got %#v", k)
		}
		i, ok := v.(starlark.Int)
		if !ok {
			return nil, fmt.Errorf("expected starlark int val, but got %#v", v)
		}
		val, ok := i.Int64()
		if !ok {
			return nil, fmt.Errorf("starlark int can't be represented as an int64: %s", i)
		}
		out[s] = int(val)
	}
	return out, nil
}

func TestMapIndex(t *testing.T) {
	x9 := map[string]int{}
	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x9":     x9,
		"toMap":  toMap,
	}

	code := []byte(`
a = x9["a"]
`)
	_, err := starlight.Eval(code, globals, nil)
	expectErr(t, err, `key "a" not in starlight_map<map[string]int>`)

	code = []byte(`
x9["a"] = 1
assert.Eq(x9["a"], 1)
assert.Eq(x9, toMap({"a": 1}))
`)

	_, err = starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedMap := map[string]int{"a": 1}
	if !reflect.DeepEqual(x9, expectedMap) {
		t.Fatalf("expected %v, got %v", expectedMap, x9)
	}

	code = []byte(`
def setIndex(d, k, v):
  d[k] = v
setIndex(x9, [], 2)
`)

	_, err = starlight.Eval(code, globals, nil)
	expectErr(t, err, `reflect.Value.Convert: value of type []interface {} cannot be converted to type string`)

	v, err := convert.ToValue(x9)
	if err != nil {
		t.Fatal(err)
	}
	v.Freeze()

	code = []byte(`
x9["a"] = 3
`)

	_, err = starlight.Eval(code, map[string]interface{}{"x9": v}, nil)
	expectErr(t, err, `cannot insert into frozen map`)

}

func expectErr(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Fatal("unexpected nil error")
	}
	if err.Error() != expected {
		t.Fatalf(`expected error "%v", got "%v"`, expected, err)
	}
}

func TestMapGet(t *testing.T) {
	x10 := map[string]int{"a": 1}
	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x10":    x10,
	}

	code := []byte(`
assert.Eq(x10.get("a"), 1)
assert.Eq(x10.get("b"), None)
assert.Eq(x10.get("a", 2), 1)
assert.Eq(x10.get("b", 2), 2)
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMapClear(t *testing.T) {

	x11 := map[string]int{"a": 1}

	var isIn *bool
	record := func(b bool) {
		isIn = &b
	}

	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x11":    x11,
		"record": record,
	}

	code := []byte(`
assert.Eq(x11["a"], 1)
x11.clear()
record("a" not in x11)
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(x11) != 0 {
		t.Errorf("expected map to be empty but was %#v", x11)
	}
	if isIn == nil {
		t.Fatal("isIn not set")
	}
	if *isIn == false {
		t.Fatal(`"not in" should have returned true, but didn't`)
	}

	code = []byte(`
b = x11["a"]
`)
	_, err = starlight.Eval(code, globals, nil)
	expectErr(t, err, `key "a" not in starlight_map<map[string]int>`)

	v, err := convert.ToValue(x11)
	if err != nil {
		t.Fatal(err)
	}
	v.Freeze()

	code = []byte(`
x11.clear()
`)

	_, err = starlight.Eval(code, map[string]interface{}{"x11": v}, nil)
	expectErr(t, err, "cannot clear frozen map")
}

func TestMapSetDefault(t *testing.T) {
	x12 := map[string]int{"a": 1}

	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x12":    x12,
	}

	code := []byte(`
assert.Eq(x12.setdefault("a"), 1)
assert.Eq(x12["a"], 1)
# This test is from starlark tests... but we can't set None as a value in
# a map[string]int
# assert.Eq(x12.setdefault("b"), None)
# assert.Eq(x12["b"], None)
assert.Eq(x12.setdefault("c", 2), 2)
assert.Eq(x12["c"], 2)
assert.Eq(x12.setdefault("c", 3), 2)
assert.Eq(x12["c"], 2)
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}
