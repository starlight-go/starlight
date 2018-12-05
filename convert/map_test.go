package convert_test

import (
	"reflect"
	"testing"

	"github.com/go-skyhook/skyhook"
)

type assert struct {
	t *testing.T
}

func (a *assert) Eq(expected, got interface{}) {
	if !reflect.DeepEqual(expected, got) {
		a.t.Fatalf("expected %v to be equal to %v", expected, got)
	}
}
func TestMapPop(t *testing.T) {
	x6 := map[string]int{"a": 1, "b": 2}
	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"x6":     x6,
	}

	code := []byte(`
assert.Eq(x6.pop("a"), 1)
assert.Eq(len(x6), 1)
assert.Eq(x6.pop("c", 3), 3)
assert.Eq(x6.pop("c", None), None) # default=None tests an edge case of UnpackArgs
assert.Eq(x6.pop("b"), 2)
assert.Eq(len(x6), 0)
`)
	_, err := skyhook.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(x6) != 0 {
		t.Fatalf("expected len of map to be 0, but was %d", len(x6))
	}

	code = []byte(`
x6.pop("c")
`)
	_, err = skyhook.Eval(code, globals, nil)
	if err == nil {
		t.Fatal("unexpected nil error")
	}
	expected := "pop: missing key"
	if err.Error() != expected {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
