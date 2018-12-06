package convert_test

import (
	"testing"

	"github.com/starlight-go/starlight"
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
