package convert

import (
	"errors"
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// ensure a *Struct is a valid starlark.Value
var _ starlark.Value = (*Struct)(nil)

// NewStruct makes a new Struct from the given struct or pointer to struct.
func NewStruct(v interface{}) *Struct {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Struct || (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct) {
		return &Struct{
			i: v,
			v: reflect.ValueOf(v),
			t: t,
		}
	}
	panic(fmt.Errorf("value must be a struct or pointer to a struct, but was %T", v))
}

// Struct is a wrapper around a Go struct to let it be manipulated by starlark
// scripts.
type Struct struct {
	i interface{}
	v reflect.Value
	t reflect.Type
}

// Attr returns a starlark value that wraps the method or field with the given
// name.
func (s *Struct) Attr(name string) (starlark.Value, error) {
	method := s.v.MethodByName(name)
	if method.Kind() != reflect.Invalid {
		return MakeStarFn(name, method.Interface()), nil
	}
	v := s.v
	if s.v.Kind() == reflect.Ptr {
		v = v.Elem()
		method = s.v.MethodByName(name)
		if method.Kind() != reflect.Invalid {
			return MakeStarFn(name, method.Interface()), nil
		}
	}
	field := v.FieldByName(name)
	if field.Kind() != reflect.Invalid {
		return ToValue(field.Interface())
	}
	return nil, nil
}

// AttrNames returns the list of all fields and methods on this struct.
func (s *Struct) AttrNames() []string {
	count := s.t.NumMethod()
	if s.v.Kind() == reflect.Ptr {
		elem := s.v.Elem()
		count += elem.NumField() + elem.NumMethod()
	} else {
		count += s.t.NumField()
	}
	names := make([]string, 0, count)
	for i := 0; i < s.t.NumMethod(); i++ {
		names = append(names, s.t.Method(i).Name)
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
		for i := 0; i < s.t.NumField(); i++ {
			names = append(names, s.t.Field(i).Name)
		}
	}
	return names
}

// SetField sets the struct field with the given name with the given value.
func (s *Struct) SetField(name string, val starlark.Value) error {
	i, err := FromValue(val)
	if err != nil {
		return err
	}
	v := s.v
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(name)
	if field.CanSet() {
		field.Set(reflect.ValueOf(i))
		return nil
	}
	return fmt.Errorf("%s is not a settable field", name)
}

// String returns the string representation of the value.
// Starlark string values are quoted as if by Python's repr.
func (s *Struct) String() string {
	return fmt.Sprint(s.i)
}

// Type returns a short string describing the value's type.
func (s *Struct) Type() string {
	return fmt.Sprintf("%T", s.i)
}

// Freeze causes the value, and all values transitively
// reachable from it through collections and closures, to be
// marked as frozen.  All subsequent mutations to the data
// structure through this API will fail dynamically, making the
// data structure immutable and safe for publishing to other
// Starlark interpreters running concurrently.
func (s *Struct) Freeze() {}

// Truth returns the truth value of an object.
func (s *Struct) Truth() starlark.Bool {
	return true
}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y).
// Hash may fail if the value's type is not hashable, or if the value
// contains a non-hashable value.
func (s *Struct) Hash() (uint32, error) {
	return 0, errors.New("not hashable")
}
