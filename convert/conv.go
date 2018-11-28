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
// all int, uint, and float numeric types, strings, and bools.  Any
// starlark.Value is passed through as-is.  A []interface{} is converted with
// MakeList, map[interface{}]interface{} is converted with MakeDict, and
// map[interface{}]bool is converted with MakeSet.
func ToValue(v interface{}) (starlark.Value, error) {
	if val, ok := v.(starlark.Value); ok {
		return val, nil
	}
	switch v := v.(type) {
	case int:
		return starlark.MakeInt(v), nil
	case int8:
		return starlark.MakeInt(int(v)), nil
	case int16:
		return starlark.MakeInt(int(v)), nil
	case int32:
		return starlark.MakeInt(int(v)), nil
	case int64:
		return starlark.MakeInt64(v), nil
	case uint:
		return starlark.MakeUint(v), nil
	case uint8:
		return starlark.MakeUint(uint(v)), nil
	case uint16:
		return starlark.MakeUint(uint(v)), nil
	case uint32:
		return starlark.MakeUint(uint(v)), nil
	case uint64:
		return starlark.MakeUint64(v), nil
	case bool:
		return starlark.Bool(v), nil
	case string:
		return starlark.String(v), nil
	case float32:
		return starlark.Float(float64(v)), nil
	case float64:
		return starlark.Float(v), nil
	case []interface{}:
		// There's no way to tell if they want a tuple or a list, so we default
		// to the more permissive list type.
		return MakeList(v)
	case map[interface{}]interface{}:
		// Dict
		return MakeDict(v)
	case map[interface{}]bool:
		// Set
		return MakeSet(v)
	}

	return nil, fmt.Errorf("type %T is not a supported starlark type", v)
}

// FromValue converts a starlark value to a go value.
func FromValue(v starlark.Value) (interface{}, error) {
	switch v := v.(type) {
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int:
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
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Func {
			dict[k] = MakeStarFn(k, v)
			continue
		}

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

// MakeList makes a list from the given values.  The acceptable values are the
// same as ToValue.
func MakeList(v []interface{}) (*starlark.List, error) {
	vals := make([]starlark.Value, len(v))
	for i := range v {
		val, err := ToValue(v[i])
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
func MakeDict(d map[interface{}]interface{}) (*starlark.Dict, error) {
	dict := starlark.Dict{}
	for k, v := range d {
		key, err := ToValue(k)
		if err != nil {
			return nil, err
		}
		val, err := ToValue(v)
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
