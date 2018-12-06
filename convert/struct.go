package convert

import (
	"errors"
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// NewStruct makes a new starlark-compatible Struct from the given struct or
// pointer to struct.  This will panic if you pass it anything else.
func NewStruct(strct interface{}) *GoStruct {
	val := reflect.ValueOf(strct)
	if val.Kind() == reflect.Struct || (val.Kind() == reflect.Ptr && val.Elem().Kind() == reflect.Struct) {
		return &GoStruct{v: val}
	}
	panic(fmt.Errorf("value must be a struct or pointer to a struct, but was %T", val.Interface()))
}

// GoStruct is a wrapper around a Go struct to let it be manipulated by starlark
// scripts.
type GoStruct struct {
	v reflect.Value
}

// Attr returns a starlark value that wraps the method or field with the given
// name.
func (g *GoStruct) Attr(name string) (starlark.Value, error) {
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
	field := v.FieldByName(name)
	if field.Kind() != reflect.Invalid {
		return toValue(field)
	}
	return nil, nil
}

// AttrNames returns the list of all fields and methods on this struct.
func (g *GoStruct) AttrNames() []string {
	count := g.v.NumMethod()
	if g.v.Kind() == reflect.Ptr {
		elem := g.v.Elem()
		count += elem.NumField() + elem.NumMethod()
	} else {
		count += g.v.NumField()
	}
	names := make([]string, 0, count)
	for i := 0; i < g.v.NumMethod(); i++ {
		names = append(names, g.v.Type().Method(i).Name)
	}
	if g.v.Kind() == reflect.Ptr {
		t := g.v.Elem().Type()
		for i := 0; i < t.NumField(); i++ {
			names = append(names, t.Field(i).Name)
		}
		for i := 0; i < t.NumMethod(); i++ {
			names = append(names, t.Method(i).Name)
		}
	} else {
		for i := 0; i < g.v.NumField(); i++ {
			names = append(names, g.v.Type().Field(i).Name)
		}
	}
	return names
}

// SetField sets the struct field with the given name with the given value.
func (g *GoStruct) SetField(name string, val starlark.Value) error {
	v := g.v
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(name)
	if field.CanSet() {
		val := conv(val, field.Type())
		field.Set(val)
		return nil
	}
	return fmt.Errorf("%s is not a settable field", name)
}

// String returns the string representation of the value.
// Starlark string values are quoted as if by Python's repr.
func (g *GoStruct) String() string {
	return fmt.Sprint(g.v.Interface())
}

// Type returns a short string describing the value's type.
func (g *GoStruct) Type() string {
	return fmt.Sprintf("starlight_struct<%T>", g.v.Interface())
}

// Freeze causes the value, and all values transitively
// reachable from it through collections and closures, to be
// marked as frozen.  All subsequent mutations to the data
// structure through this API will fail dynamically, making the
// data structure immutable and safe for publishing to other
// Starlark interpreters running concurrently.
func (g *GoStruct) Freeze() {}

// Truth returns the truth value of an object.
func (g *GoStruct) Truth() starlark.Bool {
	return true
}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y).
// Hash may fail if the value's type is not hashable, or if the value
// contains a non-hashable value.
func (g *GoStruct) Hash() (uint32, error) {
	return 0, errors.New("starlight_struct is not hashable")
}
