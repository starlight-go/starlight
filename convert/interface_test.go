package convert_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/starlight-go/starlight"
)

func TestInterfaceStructPtr(t *testing.T) {
	type resp struct {
		Body io.Reader
	}

	r := resp{Body: strings.NewReader("hi!")}

	globals := map[string]interface{}{
		"r":       r,
		"readAll": ioutil.ReadAll,
	}

	code := []byte(`
a = readAll(r.Body)
`)
	out, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := out["a"].([]byte)
	if !ok {
		t.Fatalf("failed to find output: %#v", out)
	}
	expected := []byte("hi!")
	if !bytes.Equal(expected, b) {
		t.Fatalf("expected %q, got %q", expected, b)
	}
}
