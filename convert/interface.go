package convert

import (
	"errors"
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// MakeGoInterface converts the given value into a GoInterface.  This will panic
// if the type is not a bool, string, float kind, int kind, or uint kind .
func MakeGoInterface(v interface{}) *GoInterface {
	val := reflect.ValueOf(v)
	ifc, ok := makeGoInterface(val)
	if !ok {
		panic(fmt.Errorf("value of type %T is not supported by GoInterface", val.Interface()))
	}
	return ifc
}

func makeGoInterface(val reflect.Value) (*GoInterface, bool) {
	// we accept pointers to anything except structs, which should go through GoStruct.
	if val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct {
		return nil, false
	}
	switch val.Kind() {
	case reflect.Ptr,
		reflect.Bool,
		reflect.String,
		reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &GoInterface{v: val}, true
	}
	return nil, false
}

// GoInterface wraps a go value to expose its methods to starlark scripts. Basic
// types will not behave as their base type (you can't add 2 to an ID, even if
// it is an int underneath).
type GoInterface struct {
	v reflect.Value
}

// Attr returns a starlark value that wraps the method or field with the given
// name.
func (g *GoInterface) Attr(name string) (starlark.Value, error) {
	method := g.v.MethodByName(name)
	if method.Kind() != reflect.Invalid {
		return makeStarFn(name, method), nil
	}
	v := g.v
	if g.v.Kind() == reflect.Ptr {
		v = v.Elem()
		method = g.v.MethodByName(name)
		if method.Kind() != reflect.Invalid {
			return makeStarFn(name, method), nil
		}
	}
	return nil, nil
}

// AttrNames returns the list of all fields and methods on this struct.
func (g *GoInterface) AttrNames() []string {
	count := g.v.NumMethod()
	if g.v.Kind() == reflect.Ptr {
		elem := g.v.Elem()
		count += elem.NumMethod()
	}
	names := make([]string, 0, count)
	for i := 0; i < g.v.NumMethod(); i++ {
		names = append(names, g.v.Type().Method(i).Name)
	}
	if g.v.Kind() == reflect.Ptr {
		t := g.v.Elem().Type()
		for i := 0; i < t.NumMethod(); i++ {
			names = append(names, t.Method(i).Name)
		}
	}
	return names
}

// String returns the string representation of the value.
// Starlark string values are quoted as if by Python's repr.
func (g *GoInterface) String() string {
	return fmt.Sprint(g.v.Interface())
}

// Type returns a short string describing the value's type.
func (g *GoInterface) Type() string {
	return fmt.Sprintf("starlight_interface<%T>", g.v.Interface())
}

// Freeze causes the value, and all values transitively
// reachable from it through collections and closures, to be
// marked as frozen.  All subsequent mutations to the data
// structure through this API will fail dynamically, making the
// data structure immutable and safe for publishing to other
// Starlark interpreters running concurrently.
func (g *GoInterface) Freeze() {}

// Truth returns the truth value of an object.
func (g *GoInterface) Truth() starlark.Bool {
	switch g.v.Kind() {
	case reflect.Bool:
		return starlark.Bool(g.v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return g.v.Uint() > 0
	case reflect.Float32, reflect.Float64:
		return g.v.Float() != 0
	case reflect.String:
		return g.v.String() != ""
	}
	// otherwise... I dunno man, sure.
	return true
}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y).
// Hash may fail if the value's type is not hashable, or if the value
// contains a non-hashable value.
func (g *GoInterface) Hash() (uint32, error) {
	return 0, errors.New("starlight_interface is not hashable")
}

// Below are conversion functions, they only work on the appropriate underlying type.
// Note that there is no ToBool because Truth() already serves that purpose.

// ToInt converts the interface value into a starlark int.  This will fail if
// the underlying type is not an int or uint type (including if the underlying
// type is a pointer to an int type).
func (g *GoInterface) ToInt() (starlark.Int, error) {
	switch g.v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return starlark.MakeInt64(g.v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return starlark.MakeUint64(g.v.Uint()), nil
	}
	return starlark.Int{}, fmt.Errorf("can't convert type %T to int64", g.v)
}

// ToString converts the interface value into a starlark string.  This will fail if
// the underlying type is not a string (including if the underlying type is a
// pointer to a string).
func (g *GoInterface) ToString() (starlark.String, error) {
	switch g.v.Kind() {
	case reflect.String:
		return starlark.String(g.v.String()), nil
	}
	return "", fmt.Errorf("can't convert type %T to string", g.v)
}

// ToFloat converts the interface value into a starlark float.  This will fail
// if the underlying type is not a float type (including if the underlying type
// is a pointer to a float).
func (g *GoInterface) ToFloat() (starlark.Float, error) {
	switch g.v.Kind() {
	case reflect.Float32, reflect.Float64:
		return starlark.Float(g.v.Float()), nil
	}
	return 0, fmt.Errorf("can't convert type %T to float64", g.v)
}
