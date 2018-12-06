package convert_test

import (
	"fmt"
	"testing"

	"github.com/starlight-go/starlight"
	"github.com/starlight-go/starlight/convert"
)

func TestSliceTruth(t *testing.T) {
	empty := []string{}
	full := []bool{false}

	globals := map[string]interface{}{
		"empty": empty,
		"full":  full,
		"fail":  t.Fatal,
	}

	code := []byte(`
def run():
	if empty:
		fail("empty slice should be false")
	if not full:
		fail("non-empty slice should be true")
run()
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSliceIndexing(t *testing.T) {
	abc := []string{
		"a", "b", "c",
	}

	globals := map[string]interface{}{
		"assert": &assert{t: t},
		"abc":    abc,
	}

	code := []byte(`
# indexing, x[i]
assert.Eq(abc[-3], "a")
assert.Eq(abc[-2], "b")
assert.Eq(abc[-1], "c")
assert.Eq(abc[0], "a")
assert.Eq(abc[1], "b")
assert.Eq(abc[2], "c")
`)

	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	tests := []fail{
		{"abc[3]", "starlight_slice<[]string> index 3 out of range [0:3]"},
		{"abc[-4]", "starlight_slice<[]string> index -1 out of range [0:3]"},
	}

	expectFails(t, tests, globals)
}

type fail struct {
	code string
	err  string
}

func expectFails(t *testing.T, tests []fail, globals map[string]interface{}) {
	for _, f := range tests {
		t.Run(f.code, func(t *testing.T) {
			_, err := starlight.Eval([]byte(f.code), globals, nil)
			expectErr(t, err, f.err)
		})
	}
}

func intSlice(vals []interface{}) ([]int, error) {
	ret := make([]int, len(vals))
	for i, v := range vals {
		x, ok := v.(int64)
		if !ok {
			return nil, fmt.Errorf("expected int64 but got %v (%T)", v, v)
		}
		ret[i] = int(x)
	}
	return ret, nil
}

func TestSliceIndexAssign(t *testing.T) {
	x3 := []int{0, 1, 2}

	globals := map[string]interface{}{
		"assert":   &assert{t: t},
		"x3":       x3,
		"intSlice": intSlice,
	}

	code := []byte(`
x3[1] = 2
x3[2] += 3
assert.Eq(x3, intSlice([0, 2, 5]))
`)

	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}

	v, err := convert.ToValue(x3)
	if err != nil {
		t.Fatal(err)
	}
	v.Freeze()

	globals["x3"] = v

	tests := []fail{
		{"x3[3]=4", "starlight_slice<[]int> index 3 out of range [0:3]"},
		{"x3[0]=0", "cannot assign to frozen slice"},
		{"x3.clear()", "cannot clear frozen slice"},
	}
	expectFails(t, tests, globals)
}

// func TestSlicePlus(t *testing.T) {
// 	x := []int{1, 2, 3}

// 	globals := map[string]interface{}{
// 		"x":        x,
// 		"intSlice": intSlice,
// 		"assert":   assert{t: t},
// 	}

// 	code := []byte(`
// y = x + [3, 4, 5]
// assert.Eq(y, intSlice([1, 2, 3, 3, 4, 5]))
// `)
// 	_, err := starlight.Eval(code, globals, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
