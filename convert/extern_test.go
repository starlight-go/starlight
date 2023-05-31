package convert_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/1set/starlight"
	"github.com/1set/starlight/convert"
	"go.starlark.net/starlark"
)

type ID int

type contact struct {
	ID
	Name    string
	Address struct {
		Street string
		Number int
	}
}

func (c contact) GetAddress(state string) string {
	return fmt.Sprintf("#%d %s %s", c.Address.Number, c.Address.Street, state)
}

func (c *contact) GetName() string {
	return c.Name
}

func TestStructGetField(t *testing.T) {
	foo := contact{
		Name: "bill",
	}
	foo.Address.Street = "oak st"
	foo.Address.Number = 3

	out, err := starlight.Eval([]byte("out = foo.Address.Street"), map[string]interface{}{"foo": foo}, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	s, ok := o.(string)
	if !ok {
		t.Fatalf("out param not string, was %T", o)
	}
	if s != "oak st" {
		t.Fatalf("expected %q, but was %q", foo.Address.Street, s)
	}
}

func TestStructPtrGetField(t *testing.T) {
	foo := &contact{
		Name: "bill",
	}
	foo.Address.Street = "oak st"
	foo.Address.Number = 3

	out, err := starlight.Eval([]byte("out = foo.Address.Street"), map[string]interface{}{"foo": foo}, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	s, ok := o.(string)
	if !ok {
		t.Fatalf("out param not string, was %T", o)
	}
	if s != "oak st" {
		t.Fatalf("expected %q, but was %q", foo.Address.Street, s)
	}
}

func TestStructPtrSetField(t *testing.T) {
	foo := &contact{
		Name: "bill",
	}

	_, err := starlight.Eval([]byte(`foo.Name = "mary"`), map[string]interface{}{"foo": foo}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := "mary"
	if foo.Name != expected {
		t.Fatalf("expected %q, but was %q", expected, foo.Name)
	}
}

func TestStructCallMethod(t *testing.T) {
	foo := contact{
		Name: "bill",
	}
	foo.Address.Street = "oak st"
	foo.Address.Number = 3

	out, err := starlight.Eval([]byte(`out = foo.GetAddress("maine")`), map[string]interface{}{"foo": foo}, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	s, ok := o.(string)
	if !ok {
		t.Fatalf("out param not string, was %T", o)
	}
	expected := "#3 oak st maine"
	if s != expected {
		t.Fatalf("expected %q, but was %q", expected, s)
	}
}

func TestStructPtrCallMethod(t *testing.T) {
	foo := &contact{
		Name: "bill",
	}

	out, err := starlight.Eval([]byte(`out = foo.GetName()`), map[string]interface{}{"foo": foo}, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	s, ok := o.(string)
	if !ok {
		t.Fatalf("out param not string, was %T", o)
	}
	expected := "bill"
	if s != expected {
		t.Fatalf("expected %q, but was %q", expected, s)
	}
}

func TestMap(t *testing.T) {
	m := map[string]*contact{
		"bill": {Name: "bill smith"},
		"mary": {Name: "mary smith"},
	}

	out, err := starlight.Eval([]byte(`out = contacts["bill"].Name`), map[string]interface{}{"contacts": m}, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	s, ok := o.(string)
	if !ok {
		t.Fatalf("out param not string, was %T", o)
	}
	expected := "bill smith"
	if s != expected {
		t.Fatalf("expected %q, but was %q", expected, s)
	}
}

func TestStructPtr(t *testing.T) {
	vals := map[string]interface{}{
		"bill": &contact{ID: 1, Name: "bill smith"},
		"mary": &contact{ID: 2, Name: "mary smith"},
	}

	out, err := starlight.Eval([]byte(`out = bill.ID == mary.ID`), vals, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	b, ok := o.(bool)
	if !ok {
		t.Fatalf("out param not bool, was %T", o)
	}
	expected := false
	if b != expected {
		t.Fatalf("expected %v, but was %v", expected, b)
	}
}
func TestNamedTypeFunc(t *testing.T) {
	id := ID(4)
	var out ID
	f := func(id ID) {
		out = id
	}
	_, err := starlight.Eval([]byte(`f(id)`), map[string]interface{}{"f": f, "id": id}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != id {
		t.Fatalf("expected %v, but got %v", id, out)
	}
}

func TestNamedTypeField(t *testing.T) {
	type foo struct {
		ID
	}
	f := &foo{ID: 5}
	g := &foo{ID: 10}
	_, err := starlight.Eval([]byte(`f.ID = g.ID`), map[string]interface{}{"f": f, "g": g}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.ID != ID(10) {
		t.Fatalf("expected %v, but got %v", ID(10), f.ID)
	}
}

func TestStructPtrMap(t *testing.T) {
	m := map[string]*contact{
		"bill": {Name: "bill smith"},
		"mary": {Name: "mary smith"},
	}

	_, err := starlight.Eval([]byte(`contacts["bill"].Name = "john smith"`), map[string]interface{}{"contacts": m}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := "john smith"
	if m["bill"].Name != expected {
		t.Fatalf("expected %q, but was %q", expected, m["bill"].Name)
	}
}

func TestModMap(t *testing.T) {
	m := map[string]string{
		"bill": "bill smith",
		"mary": "mary smith",
	}

	_, err := starlight.Eval([]byte(`contacts["bill"] = "john smith"`), map[string]interface{}{"contacts": m}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := "john smith"
	if m["bill"] != expected {
		t.Fatalf("expected %q, but was %q", expected, m["bill"])
	}
}

func TestMapItems(t *testing.T) {
	m := map[string]*contact{
		"bill": {Name: "bill smith"},
		"mary": {Name: "mary smith"},
	}
	output := map[string]string{}
	record := func(k, v string) {
		output[k] = v
	}
	code := []byte(`
def do():
	for k, v in contacts.items():
		record(k, v.Name)
do()
	`)
	globals := map[string]interface{}{"contacts": m, "record": record}
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"bill": "bill smith", "mary": "mary smith"}
	if !reflect.DeepEqual(expected, output) {
		t.Fatalf("expected %v, but was %v", expected, output)
	}
}

func TestIndexSliceItems(t *testing.T) {
	slice := []*contact{
		{Name: "bill smith"},
		{Name: "mary smith"},
	}
	code := []byte(`
out = contacts[1].Name
	`)
	globals := map[string]interface{}{"contacts": slice}
	out, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := out["out"]
	if !ok {
		t.Fatal("out param not found")
	}
	s, ok := o.(string)
	if !ok {
		t.Fatalf("out param not string, was %T", o)
	}
	expected := "mary smith"
	if s != expected {
		t.Fatalf("expected %q, but was %q", expected, s)
	}
}

type Named interface {
	Name() string
}

type foo struct {
	name string
}

func (f *foo) Name() string {
	return f.name
}

var resultFuncCall starlark.StringDict
var result string

func BenchmarkFuncCall(b *testing.B) {
	fn := func(s string) {
		result = s
	}

	globals := map[string]interface{}{
		"fn":  fn,
		"foo": &foo{name: "bob"},
	}
	dict, err := convert.MakeStringDict(globals)
	if err != nil {
		b.Fatal(err)
	}
	code := []byte(`fn(foo.Name())`)
	// precompile
	_, p, err := starlark.SourceProgram("foo.star", code, dict.Has)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		dict, err := convert.MakeStringDict(globals)
		if err != nil {
			b.Fatal(err)
		}
		resultFuncCall, err = p.Init(new(starlark.Thread), dict)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestInterfaceAssignment(t *testing.T) {
	var output string
	fn := func(n Named) {
		output = n.Name()
	}
	globals := map[string]interface{}{
		"fn":  fn,
		"foo": &foo{name: "bob"},
	}
	code := []byte(`fn(foo)`)
	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if output != "bob" {
		t.Fatalf("expected %q but got %q", "bob", output)
	}
}

type Celsius float64

func (c Celsius) ToF() Fahrenheit {
	return Fahrenheit(c*9/5 + 32)
}

type Fahrenheit float64

func (f Fahrenheit) ToC() Celsius {
	return Celsius((f - 32) * 5 / 9)
}

func TestGoInterface(t *testing.T) {
	f := Fahrenheit(451)

	globals := map[string]interface{}{
		"f": f,
	}
	code := []byte(`
c = f.ToC()
`)
	output, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := output["c"]
	if !ok {
		t.Fatal("missing value in output")
	}
	c, ok := v.(Celsius)
	if !ok {
		t.Fatalf("expected c to be Celsius but was %T", v)
	}

	if c != f.ToC() {
		t.Fatalf("expected %v but got %v", f.ToC(), c)
	}
}

func TestGoSlice(t *testing.T) {
	vals := []string{"a", "b", "c", "d", "e"}

	globals := map[string]interface{}{
		"vals": vals,
	}
	code := []byte(`
out = vals[1:-1]
`)
	output, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := output["out"]
	if !ok {
		t.Fatal("missing value in output")
	}
	out, ok := v.([]string)
	if !ok {
		t.Fatalf("expected output to be []string but was %T", v)
	}

	expected := vals[1:4]
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("expected %#v but got %#v", expected, out)
	}
}
