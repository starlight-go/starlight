package convert_test

import (
	"fmt"
	"testing"

	"github.com/starlight-go/starlight"
)

func TestVariadic(t *testing.T) {
	globals := map[string]interface{}{
		"sprint":  fmt.Sprint,
		"fatal":   t.Fatal,
		"sprintf": fmt.Sprintf,
	}

	code := []byte(`
def do(): 
	v = sprint(False)
	if v != "false" :
		fatal("unexpected output: ", v)
	v = sprint(False, 1)
	if v != "false 1" :
		fatal("unexpected output:", v)
	v = sprint(False, 1, " hi ", {"key":"value"})
	if v != 'false 1 hi map[key:"value"]' :
		fatal("unexpected output:", v)
	v = sprintf("this is your %dst formatted message", 1)
	if v != "this is your 1st formatted message":
		fatal("unexpected output:", v)
do()
`)

	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}
