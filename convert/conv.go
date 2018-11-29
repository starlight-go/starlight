package convert

import (
	"errors"
	"fmt"
	"reflect"

	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func init() {
	resolve.AllowNestedDef = true // allow def statements within function bodies
	resolve.AllowLambda = true    // allow lambda expressions
	resolve.AllowFloat = true     // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true       // allow the 'set' built-in
	resolve.AllowBitwise = true   // allow bitwise operands
}

// ToValue attempts to convert the given value to a starlark.Value.  It supports
// all int, uint, and float numeric types, plus strings and bools.  It supports
// structs, maps, slices, and functions that use the aforementioned.  Any
// starlark.Value is passed through as-is.
func ToValue(v interface{}) (starlark.Value, error) {
	if val, ok := v.(starlark.Value); ok {
		return val, nil
	}
	val := reflect.ValueOf(v)
	kind := val.Kind()
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		kind = val.Elem().Kind()
	}
	switch kind {
	case reflect.Bool:
		return starlark.Bool(val.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return starlark.MakeInt(int(val.Int())), nil
	case reflect.Int64:
		return starlark.MakeInt64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return starlark.MakeUint(uint(val.Uint())), nil
	case reflect.Uint64:
		return starlark.MakeUint64(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return starlark.Float(val.Float()), nil
	case reflect.Func:
		return MakeStarFn("fn", v), nil
	case reflect.Map:
		return MakeDict(v)
	case reflect.String:
		return starlark.String(val.String()), nil
	case reflect.Slice, reflect.Array:
		// There's no way to tell if they want a tuple or a list, so we default
		// to the more permissive list type.
		return MakeList(v)
	case reflect.Struct:
		return NewStruct(v), nil
	}

	return nil, fmt.Errorf("type %T is not a supported starlark type", v)
}

// FromValue converts a starlark value to a go value.
func FromValue(v starlark.Value) (interface{}, error) {
	switch v := v.(type) {
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int:
		// starlark ints can be signed or unsigned
		if i, ok := v.Int64(); ok {
			return i, nil
		}
		if i, ok := v.Uint64(); ok {
			return i, nil
		}

		// buh... maybe > maxint64?  Dunno
		return nil, fmt.Errorf("can't convert starlark.Int %q to int", v)
	case starlark.Float:
		return float64(v), nil
	case starlark.String:
		return string(v), nil
	case *starlark.List:
		return FromList(v)
	case starlark.Tuple:
		return FromTuple(v)
	case *starlark.Dict:
		return FromDict(v)
	case *starlark.Set:
		return FromSet(v)
	}
	return nil, fmt.Errorf("type %T is not a supported starlark type", v)
}

// MakeStringDict makes a StringDict from the given arg. The types supported are
// the same as ToValue.
func MakeStringDict(m map[string]interface{}) (starlark.StringDict, error) {
	dict := make(starlark.StringDict, len(m))
	for k, v := range m {
		val, err := ToValue(v)
		if err != nil {
			return nil, err
		}
		dict[k] = val
	}
	return dict, nil
}

// FromStringDict makes a map[string]interface{} from the given arg.  Any
// unconvertible values are ignored.
func FromStringDict(m starlark.StringDict) map[string]interface{} {
	ret := make(map[string]interface{}, len(m))
	for k, v := range m {
		val, err := FromValue(v)
		if err != nil {
			// we just ignore these, since they may be things like starlark
			// functions that we just can't represent.
			continue
		}
		ret[k] = val
	}
	return ret
}

// FromTuple converts a starlark.Tuple into a []interface{}.
func FromTuple(v starlark.Tuple) ([]interface{}, error) {
	vals := []starlark.Value(v)
	ret := make([]interface{}, len(vals))
	for i := range vals {
		val, err := FromValue(vals[i])
		if err != nil {
			return nil, err
		}
		ret[i] = val
	}
	return ret, nil
}

// MakeTuple makes a tuple from the given values.  The acceptable values are the
// same as ToValue.
func MakeTuple(v []interface{}) (starlark.Tuple, error) {
	vals := make([]starlark.Value, len(v))
	for i := range v {
		val, err := ToValue(v[i])
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return starlark.Tuple(vals), nil
}

// MakeList makes a list from the given slice or array. The acceptable values
// in the list are the same as ToValue.
func MakeList(v interface{}) (*starlark.List, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Slice || val.Kind() != reflect.Array {
		panic(fmt.Errorf("value should be slice or array but was %T", v))
	}
	vals := make([]starlark.Value, val.Len())
	for i := 0; i < val.Len(); i++ {
		val, err := ToValue(val.Index(i))
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return starlark.NewList(vals), nil
}

// FromList creates a go slice from the given starlark list.
func FromList(l *starlark.List) ([]interface{}, error) {
	ret := make([]interface{}, 0, l.Len())
	var v starlark.Value
	i := l.Iterate()
	defer i.Done()
	for i.Next(&v) {
		val, err := FromValue(v)
		if err != nil {
			return nil, err
		}
		ret = append(ret, val)
	}
	return ret, nil
}

// MakeDict makes a Dict from the given map.  The acceptable keys and values are
// the same as ToValue.
func MakeDict(v interface{}) (starlark.Value, error) {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Map {
		panic(fmt.Errorf("can't make map of %T", v))
	}
	dict := starlark.Dict{}
	for _, k := range r.MapKeys() {
		key, err := ToValue(k.Interface())
		if err != nil {
			return nil, err
		}

		val, err := ToValue(r.MapIndex(k).Interface())
		if err != nil {
			return nil, err
		}
		dict.SetKey(key, val)
	}
	return &dict, nil
}

// FromDict converts a starlark.Dict to a map[interface{}]interface{}
func FromDict(m *starlark.Dict) (map[interface{}]interface{}, error) {
	ret := make(map[interface{}]interface{}, m.Len())
	for _, k := range m.Keys() {
		key, err := FromValue(k)
		if err != nil {
			return nil, err
		}
		val, _, err := m.Get(k)
		if err != nil {
			return nil, err
		}
		ret[key] = val
	}
	return ret, nil
}

// MakeSet makes a Set from the given map.  The acceptable keys
// the same as ToValue.
func MakeSet(s map[interface{}]bool) (*starlark.Set, error) {
	set := starlark.Set{}
	for k := range s {
		key, err := ToValue(k)
		if err != nil {
			return nil, err
		}
		if err := set.Insert(key); err != nil {
			return nil, err
		}
	}
	return &set, nil
}

// FromSet converts a starlark.Set to a map[interface{}]bool
func FromSet(s *starlark.Set) (map[interface{}]bool, error) {
	ret := make(map[interface{}]bool, s.Len())
	var v starlark.Value
	i := s.Iterate()
	defer i.Done()
	for i.Next(&v) {
		val, err := FromValue(v)
		if err != nil {
			return nil, err
		}
		ret[val] = true
	}
	return ret, nil
}

// Kwarg is a single instance of a python foo=bar style named argument.
type Kwarg struct {
	Name  string
	Value interface{}
}

// FromKwargs converts a python style name=val, name2=val2 list of tuples into a
// []Kwarg.  It is an error if any tuple is not exactly 2 values,
// or if the first one is not a string.
func FromKwargs(kwargs []starlark.Tuple) ([]Kwarg, error) {
	args := make([]Kwarg, 0, len(kwargs))
	for _, t := range kwargs {
		tup, err := FromTuple(t)
		if err != nil {
			return nil, err
		}
		if len(tup) != 2 {
			return nil, fmt.Errorf("kwarg tuple should have 2 vals, has %v", len(tup))
		}
		s, ok := tup[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected name of kwarg to be string, but was %T (%#v)", tup[0], tup[0])
		}
		args = append(args, Kwarg{Name: s, Value: tup[1]})
	}
	return args, nil
}

var errType = reflect.TypeOf((*error)(nil)).Elem()

// MakeStarFn creates a wrapper around the given function that can be called from
// a starlark script.  Argument support is the same as ToValue. If the last value
// the function returns is an error, it will cause an error to be returned from
// the starlark function.  If there are no other errors, the function will return
// None.  If there's exactly one other value, the function will return the
// starlark equivalent of that value.  If there is more than one return value,
// they'll be returned as a tuple.  MakeStarFn will panic if you pass it
// something other than a function.
func MakeStarFn(name string, gofn interface{}) *starlark.Builtin {
	t := reflect.TypeOf(gofn)
	if t.Kind() != reflect.Func {
		panic(errors.New("fn is not a function"))
	}

	return starlark.NewBuiltin(name, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) != t.NumIn() {
			return starlark.None, fmt.Errorf("expected %d args but got %d", t.NumIn(), len(args))
		}
		v := reflect.ValueOf(gofn)
		vals, err := FromTuple(args)
		if err != nil {
			return starlark.None, err
		}
		rvs := make([]reflect.Value, 0, len(vals))
		for _, v := range vals {
			rvs = append(rvs, reflect.ValueOf(v))
		}
		out := v.Call(rvs)
		if len(out) == 0 {
			return starlark.None, nil
		}
		last := out[len(out)-1]
		err = nil
		if last.Type() == errType {
			if v := last.Interface(); v != nil {
				err = v.(error)
			}
			out = out[:len(out)-1]
		}
		if len(out) == 1 {
			v, err2 := ToValue(out[0].Interface())
			if err2 != nil {
				return starlark.None, err2
			}
			return v, err
		}
		ifcs := make([]interface{}, 0, len(out))
		// tuple-up multple values
		for i := range out {
			ifcs = append(ifcs, out[i].Interface())
		}
		tup, err2 := MakeTuple(ifcs)
		if err != nil {
			return starlark.None, err2
		}
		return tup, err
	})
}

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
	}
	field := v.FieldByName(name)
	if field.Kind() != reflect.Invalid {
		return ToValue(field.Interface())
	}
	method = s.v.MethodByName(name)
	if method.Kind() != reflect.Invalid {
		return MakeStarFn(name, method.Interface()), nil
	}
	return nil, nil
}

// AttrNames returns the list of all fields and methods on this struct.
func (s *Struct) AttrNames() []string {
	names := make([]string, 0, s.t.NumField()+s.t.NumMethod())
	for i := 0; i < s.t.NumField(); i++ {
		names = append(names, s.t.Field(i).Name)
	}
	for i := 0; i < s.t.NumMethod(); i++ {
		names = append(names, s.t.Method(i).Name)
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
