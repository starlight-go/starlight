package convert_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hippogryph/skyhook"
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

	out, err := skyhook.Eval([]byte("out = foo.Address.Street"), map[string]interface{}{"foo": foo})
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

	out, err := skyhook.Eval([]byte("out = foo.Address.Street"), map[string]interface{}{"foo": foo})
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

	_, err := skyhook.Eval([]byte(`foo.Name = "mary"`), map[string]interface{}{"foo": foo})
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

	out, err := skyhook.Eval([]byte(`out = foo.GetAddress("maine")`), map[string]interface{}{"foo": foo})
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

	out, err := skyhook.Eval([]byte(`out = foo.GetName()`), map[string]interface{}{"foo": foo})
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
		"bill": &contact{Name: "bill smith"},
		"mary": &contact{Name: "mary smith"},
	}

	out, err := skyhook.Eval([]byte(`out = contacts["bill"].Name`), map[string]interface{}{"contacts": m})
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

func TestNamedType(t *testing.T) {
	m := map[string]*contact{
		"bill": &contact{ID: 1, Name: "bill smith"},
		"mary": &contact{ID: 2, Name: "mary smith"},
	}

	out, err := skyhook.Eval([]byte(`out = contacts["bill"].ID == contacts["mary"].ID`), map[string]interface{}{"contacts": m})
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

func TestModMap(t *testing.T) {
	m := map[string]*contact{
		"bill": &contact{Name: "bill smith"},
		"mary": &contact{Name: "mary smith"},
	}

	_, err := skyhook.Eval([]byte(`contacts["bill"].Name = "john smith"`), map[string]interface{}{"contacts": m})
	if err != nil {
		t.Fatal(err)
	}
	expected := "john smith"
	if m["bill"].Name != expected {
		t.Fatalf("expected %q, but was %q", expected, m["bill"].Name)
	}
}

func TestMapItems(t *testing.T) {
	m := map[string]*contact{
		"bill": &contact{Name: "bill smith"},
		"mary": &contact{Name: "mary smith"},
	}
	output := map[string]string{}
	record := func(k, v string) {
		output[k] = v
	}
	_, err := skyhook.Eval([]byte(`
def do():
	for k, v in contacts.items():
		record(k, v.Name)
do()
`), map[string]interface{}{"contacts": m, "record": record})
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
		&contact{Name: "bill smith"},
		&contact{Name: "mary smith"},
	}
	out, err := skyhook.Eval([]byte(`
out = contacts[1].Name
`), map[string]interface{}{"contacts": slice})
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
