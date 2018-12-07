package convert_test

import (
	"bytes"
	"fmt"
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

type Foo int

func (f Foo) Foo() string {
	return fmt.Sprintf("Foo: %v", f)
}

func (f *Foo) PFoo() bool {
	return true
}

func toFoo(i int) Foo {
	return Foo(i)
}

func toPFoo(i int) *Foo {
	f := Foo(i)
	return &f
}

func nilPtr() *Foo {
	return nil
}

type Name string

func (n Name) Double() string {
	return string(n + n)
}

func TestInterfaceCall(t *testing.T) {
	globals := map[string]interface{}{
		"toFoo":  toFoo,
		"assert": &assert{t: t},
		"fatal":  t.Fatal,
	}

	code := []byte(`
f = toFoo(1)
assert.Eq("Foo: 1", f.Foo())
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInterfacePtrCall(t *testing.T) {
	globals := map[string]interface{}{
		"toPFoo": toPFoo,
		"assert": &assert{t: t},
		"fatal":  t.Fatal,
	}

	code := []byte(`
f = toPFoo(1)
# assert.Eq(True, f.PFoo())
assert.Eq("Foo: 1", f.Foo())
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInterfaceTruth(t *testing.T) {
	toName := func(s string) Name {
		return Name(s)
	}

	globals := map[string]interface{}{
		"toFoo":  toFoo,
		"toPFoo": toPFoo,
		"assert": &assert{t: t},
		"fatal":  t.Fatal,
		"nilPtr": nilPtr,
		"toName": toName,
	}

	code := []byte(`
def do():
	if not toPFoo(0):
		fatal("expected non-nil pointer type to be true")
	if toFoo(0):
		fatal("expected zero int type to be false")
	if not toFoo(1):
		fatal("expected non-zero int type to be true")
	if nilPtr():
		fatal("expected nil pointer type to be false")
	if toName(""):
		fatal("expected empty string type to be false")
	if not toName("0"):
		fatal("expected non-empty string type to be true")
do()
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

type FFloat float64

func (f FFloat) String() string {
	return fmt.Sprint(float64(f))
}

type OK bool

func (ok OK) String() string {
	return fmt.Sprint(bool(ok))
}

type Size uint16

func (s Size) String() string {
	return fmt.Sprint(uint16(s))
}

func TestInterfaceConvert(t *testing.T) {
	toName := func(s string) Name {
		return Name(s)
	}
	toFFloat := func(f float64) FFloat {
		return FFloat(f)
	}

	toOK := func(b bool) OK {
		return OK(b)
	}

	toSize := func(u uint64) Size {
		return Size(u)
	}
	globals := map[string]interface{}{
		"toFoo":    toFoo,
		"toPFoo":   toPFoo,
		"fatal":    t.Fatal,
		"fatalf":   t.Fatalf,
		"toName":   toName,
		"toFFloat": toFFloat,
		"toOK":     toOK,
		"toSize":   toSize,
	}

	code := []byte(`
def do():
	f = toPFoo(12).toInt()
	if f != 12:
		fatalf("expected typed int to return to int")
	if "phil" != toName("phil").toString():
		fatal("expected typed string to return to string")
	if 2.2 != toFFloat(2.2).toFloat():
		fatal("expected typed float to return to float")
	if True != toOK(True).toBool():
		fatal("expected typed bool to return to bool")
	if 5 != toSize(5).toUint():
		fatal("expected typed uint to return to uint")
do()
`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}
