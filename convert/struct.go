package convert

import (
	"errors"
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// NewStruct makes a new starlark-compatible Struct from the given struct or
// pointer to struct.  This will panic if you pass it anything else.
func NewStruct(s interface{}) *GoStruct {
	val := reflect.ValueOf(s)
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
func (s *GoStruct) Attr(name string) (starlark.Value, error) {
	method := s.v.MethodByName(name)
	if method.Kind() != reflect.Invalid {
		return makeStarFn(name, method), nil
	}
	v := s.v
	if s.v.Kind() == reflect.Ptr {
		v = v.Elem()
		method = s.v.MethodByName(name)
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
func (s *GoStruct) AttrNames() []string {
	count := s.v.NumMethod()
	if s.v.Kind() == reflect.Ptr {
		elem := s.v.Elem()
		count += elem.NumField() + elem.NumMethod()
	} else {
		count += s.v.NumField()
	}
	names := make([]string, 0, count)
	for i := 0; i < s.v.NumMethod(); i++ {
		names = append(names, s.v.Type().Method(i).Name)
	}
	if s.v.Kind() == reflect.Ptr {
		t := s.v.Elem().Type()
		for i := 0; i < t.NumField(); i++ {
			names = append(names, t.Field(i).Name)
		}
		for i := 0; i < t.NumMethod(); i++ {
			names = append(names, t.Method(i).Name)
		}
	} else {
		for i := 0; i < s.v.NumField(); i++ {
			names = append(names, s.v.Type().Field(i).Name)
		}
	}
	return names
}

// SetField sets the struct field with the given name with the given value.
func (s *GoStruct) SetField(name string, val starlark.Value) error {
	i := FromValue(val)
	v := s.v
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(name)
	if field.CanSet() {
		val := reflect.ValueOf(i)
		if field.Type() != val.Type() {
			val = val.Convert(field.Type())
		}
		field.Set(val)
		return nil
	}
	return fmt.Errorf("%s is not a settable field", name)
}

// String returns the string representation of the value.
// Starlark string values are quoted as if by Python's repr.
func (s *GoStruct) String() string {
	return fmt.Sprint(s.v.Interface())
}

// Type returns a short string describing the value's type.
func (s *GoStruct) Type() string {
	return fmt.Sprintf("skyhook_struct<%T>", s.v.Interface())
}

// Freeze causes the value, and all values transitively
// reachable from it through collections and closures, to be
// marked as frozen.  All subsequent mutations to the data
// structure through this API will fail dynamically, making the
// data structure immutable and safe for publishing to other
// Starlark interpreters running concurrently.
func (s *GoStruct) Freeze() {}

// Truth returns the truth value of an object.
func (s *GoStruct) Truth() starlark.Bool {
	return true
}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y).
// Hash may fail if the value's type is not hashable, or if the value
// contains a non-hashable value.
func (s *GoStruct) Hash() (uint32, error) {
	return 0, errors.New("skyhook_struct is not hashable")
}
