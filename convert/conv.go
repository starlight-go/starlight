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
	return toValue(reflect.ValueOf(v))
}

func toValue(val reflect.Value) (starlark.Value, error) {
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
		return makeStarFn("fn", val), nil
	case reflect.Map:
		return makeDict(val)
	case reflect.String:
		return starlark.String(val.String()), nil
	case reflect.Slice, reflect.Array:
		// There's no way to tell if they want a tuple or a list, so we default
		// to the more permissive list type.
		return makeList(val)
	case reflect.Struct:
		return makeStruct(val), nil
	}

	return nil, fmt.Errorf("type %T is not a supported starlark type", val.Interface())
}

// FromValue converts a starlark value to a go value.
func FromValue(v starlark.Value) interface{} {
	switch v := v.(type) {
	case starlark.Bool:
		return bool(v)
	case starlark.Int:
		// starlark ints can be signed or unsigned
		if i, ok := v.Int64(); ok {
			return i
		}
		if i, ok := v.Uint64(); ok {
			return i
		}
		// buh... maybe > maxint64?  Dunno
		panic(fmt.Errorf("can't convert starlark.Int %q to int", v))
	case starlark.Float:
		return float64(v)
	case starlark.String:
		return string(v)
	case *starlark.List:
		return FromList(v)
	case starlark.Tuple:
		return FromTuple(v)
	case *starlark.Dict:
		return FromDict(v)
	case *starlark.Set:
		return FromSet(v)
	case *Struct:
		return v.v.Interface()
	default:
		// dunno, hope it's a custom type that the receiver knows how to deal with.
		return v
	}
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
		ret[k] = FromValue(v)
	}
	return ret
}

// FromTuple converts a starlark.Tuple into a []interface{}.
func FromTuple(v starlark.Tuple) []interface{} {
	ret := make([]interface{}, len(v))
	for i := range v {
		ret[i] = FromValue(v[i])
	}
	return ret
}

// MakeTuple makes a tuple from the given slice.  The acceptable types in the
// slice are the same as ToValue.
func MakeTuple(v interface{}) (starlark.Tuple, error) {
	return makeTuple(reflect.ValueOf(v))
}

func makeTuple(val reflect.Value) (starlark.Tuple, error) {
	vals, err := makeSliceVals(val)
	if err != nil {
		return nil, err
	}
	return starlark.Tuple(vals), nil
}

// MakeList makes a list from the given slice or array. The acceptable values
// in the list are the same as ToValue.
func MakeList(v interface{}) (*starlark.List, error) {
	return makeList(reflect.ValueOf(v))
}

func makeList(val reflect.Value) (*starlark.List, error) {
	vals, err := makeSliceVals(val)
	if err != nil {
		return nil, err
	}
	return starlark.NewList(vals), nil
}

func makeSliceVals(val reflect.Value) ([]starlark.Value, error) {
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		panic(fmt.Errorf("value should be slice or array but was %v, %T", val.Kind(), val.Interface()))
	}
	vals := make([]starlark.Value, val.Len())
	for i := 0; i < val.Len(); i++ {
		val, err := toValue(val.Index(i))
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return vals, nil
}

// FromList creates a go slice from the given starlark list.
func FromList(l *starlark.List) []interface{} {
	ret := make([]interface{}, 0, l.Len())
	var v starlark.Value
	i := l.Iterate()
	defer i.Done()
	for i.Next(&v) {
		val := FromValue(v)
		ret = append(ret, val)
	}
	return ret
}

// MakeDict makes a Dict from the given map.  The acceptable keys and values are
// the same as ToValue.
func MakeDict(v interface{}) (starlark.Value, error) {
	return makeDict(reflect.ValueOf(v))
}

func makeDict(val reflect.Value) (starlark.Value, error) {
	if val.Kind() != reflect.Map {
		panic(fmt.Errorf("can't make map of %T", val.Interface()))
	}
	dict := starlark.Dict{}
	for _, k := range val.MapKeys() {
		key, err := toValue(k)
		if err != nil {
			return nil, err
		}

		val, err := toValue(val.MapIndex(k))
		if err != nil {
			return nil, err
		}
		dict.SetKey(key, val)
	}
	return &dict, nil
}

// FromDict converts a starlark.Dict to a map[interface{}]interface{}
func FromDict(m *starlark.Dict) map[interface{}]interface{} {
	ret := make(map[interface{}]interface{}, m.Len())
	for _, k := range m.Keys() {
		key := FromValue(k)
		// should never be not found or unhashable, so ignore err and found.
		val, _, _ := m.Get(k)
		ret[key] = val
	}
	return ret
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
func FromSet(s *starlark.Set) map[interface{}]bool {
	ret := make(map[interface{}]bool, s.Len())
	var v starlark.Value
	i := s.Iterate()
	defer i.Done()
	for i.Next(&v) {
		val := FromValue(v)
		ret[val] = true
	}
	return ret
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
		tup := FromTuple(t)
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
	v := reflect.ValueOf(gofn)
	if v.Kind() != reflect.Func {
		panic(errors.New("fn is not a function"))
	}
	return makeStarFn(name, v)
}

func makeStarFn(name string, gofn reflect.Value) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) != gofn.Type().NumIn() {
			return starlark.None, fmt.Errorf("expected %d args but got %d", gofn.Type().NumIn(), len(args))
		}
		vals := FromTuple(args)
		rvs := make([]reflect.Value, 0, len(vals))
		for i, v := range vals {
			val := reflect.ValueOf(v)
			argT := gofn.Type().In(i)
			if val.Type() != argT {
				val = val.Convert(argT)
			}
			rvs = append(rvs, val)
		}
		out := gofn.Call(rvs)
		if len(out) == 0 {
			return starlark.None, nil
		}
		last := out[len(out)-1]
		var err error
		if last.Type() == errType {
			if v := last.Interface(); v != nil {
				err = v.(error)
			}
			out = out[:len(out)-1]
		}
		if len(out) == 1 {
			v, err2 := toValue(out[0])
			if err2 != nil {
				return starlark.None, err2
			}
			return v, err
		}
		res := make([]starlark.Value, 0, len(out))
		// tuple-up multple values
		for i := range out {
			val, err := toValue(out[i])
			if err != nil {
				return starlark.None, err
			}
			res = append(res, val)
		}
		return starlark.Tuple(res), nil
	})
}
