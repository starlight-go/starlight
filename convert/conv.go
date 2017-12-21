package convert

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/skylark"
	"github.com/google/skylark/resolve"
)

func init() {
	resolve.AllowNestedDef = true // allow def statements within function bodies
	resolve.AllowLambda = true    // allow lambda expressions
	resolve.AllowFloat = true     // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true       // allow the 'set' built-in
}

// ToValue attempts to convert the given value to a skylark.Value.  It supports
// all int, uint, and float numeric types, strings, and bools.  Any
// skylark.Value is passed through as-is.  A []interface{} is converted with
// MakeList, map[interface{}]interface{} is converted with MakeDict, and
// map[interface{}]bool is converted with MakeSet.
func ToValue(v interface{}) (skylark.Value, error) {
	if val, ok := v.(skylark.Value); ok {
		return val, nil
	}
	switch v := v.(type) {
	case int:
		return skylark.MakeInt(v), nil
	case int8:
		return skylark.MakeInt(int(v)), nil
	case int16:
		return skylark.MakeInt(int(v)), nil
	case int32:
		return skylark.MakeInt(int(v)), nil
	case int64:
		return skylark.MakeInt64(v), nil
	case uint:
		return skylark.MakeUint(v), nil
	case uint8:
		return skylark.MakeUint(uint(v)), nil
	case uint16:
		return skylark.MakeUint(uint(v)), nil
	case uint32:
		return skylark.MakeUint(uint(v)), nil
	case uint64:
		return skylark.MakeUint64(v), nil
	case bool:
		return skylark.Bool(v), nil
	case string:
		return skylark.String(v), nil
	case float32:
		return skylark.Float(float64(v)), nil
	case float64:
		return skylark.Float(v), nil
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

	return nil, fmt.Errorf("type %T is not a supported skylark type", v)
}

// FromValue converts a skylark value to a go value.
func FromValue(v skylark.Value) (interface{}, error) {
	switch v := v.(type) {
	case skylark.Bool:
		return bool(v), nil
	case skylark.Int:
		if i, ok := v.Int64(); ok {
			return i, nil
		}
		if i, ok := v.Uint64(); ok {
			return i, nil
		}
		// buh... maybe > maxint64?  Dunno
		return nil, fmt.Errorf("can't convert skylark.Int %q to int", v)
	case skylark.Float:
		return float64(v), nil
	case skylark.String:
		return string(v), nil
	case *skylark.List:
		return FromList(v)
	case skylark.Tuple:
		return FromTuple(v)
	case *skylark.Dict:
		return FromDict(v)
	case *skylark.Set:
		return FromSet(v)
	}
	return nil, fmt.Errorf("type %T is not a supported skylark type", v)
}

// MakeStringDict makes a StringDict from the given arg. The types supported are
// the same as ToValue.
func MakeStringDict(m map[string]interface{}) (skylark.StringDict, error) {
	dict := make(skylark.StringDict, len(m))
	for k, v := range m {
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Func {
			dict[k] = MakeSkyFn(k, v)
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
func FromStringDict(m skylark.StringDict) map[string]interface{} {
	ret := make(map[string]interface{}, len(m))
	for k, v := range m {
		val, err := FromValue(v)
		if err != nil {
			// we just ignore these, since they may be things like skylark
			// functions that we just can't represent.
			continue
		}
		ret[k] = val
	}
	return ret
}

// FromTuple converts a skylark.Tuple into a []interface{}.
func FromTuple(v skylark.Tuple) ([]interface{}, error) {
	vals := []skylark.Value(v)
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
func MakeTuple(v []interface{}) (skylark.Tuple, error) {
	vals := make([]skylark.Value, len(v))
	for i := range v {
		val, err := ToValue(v[i])
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return skylark.Tuple(vals), nil
}

// MakeList makes a list from the given values.  The acceptable values are the
// same as ToValue.
func MakeList(v []interface{}) (*skylark.List, error) {
	vals := make([]skylark.Value, len(v))
	for i := range v {
		val, err := ToValue(v[i])
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return skylark.NewList(vals), nil
}

// FromList creates a go slice from the given skylark list.
func FromList(l *skylark.List) ([]interface{}, error) {
	ret := make([]interface{}, 0, l.Len())
	var v skylark.Value
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
func MakeDict(d map[interface{}]interface{}) (*skylark.Dict, error) {
	dict := skylark.Dict{}
	for k, v := range d {
		key, err := ToValue(k)
		if err != nil {
			return nil, err
		}
		val, err := ToValue(v)
		if err != nil {
			return nil, err
		}
		dict.Set(key, val)
	}
	return &dict, nil
}

// FromDict converts a skylark.Dict to a map[interface{}]interface{}
func FromDict(m *skylark.Dict) (map[interface{}]interface{}, error) {
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
func MakeSet(s map[interface{}]bool) (*skylark.Set, error) {
	set := skylark.Set{}
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

// FromSet converts a skylark.Set to a map[interface{}]bool
func FromSet(s *skylark.Set) (map[interface{}]bool, error) {
	ret := make(map[interface{}]bool, s.Len())
	var v skylark.Value
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
func FromKwargs(kwargs []skylark.Tuple) ([]Kwarg, error) {
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

// MakeSkyFn creates a wrapper around the given function that can be called from
// a skylark script.  Argument support is the same as ToValue. If the last value
// the function returns is an error, it will cause an error to be returned from
// the skylark function.  If there are no other errors, the function will return
// None.  If there's exactly one other value, the function will return the
// skylark equivalent of that value.  If there is more than one return value,
// they'll be returned as a tuple.  MakeSkyFn will panic if you pass it
// something other than a function.
func MakeSkyFn(name string, gofn interface{}) *skylark.Builtin {
	t := reflect.TypeOf(gofn)
	if t.Kind() != reflect.Func {
		panic(errors.New("fn is not a function"))
	}

	return skylark.NewBuiltin(name, func(thread *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
		if len(args) != t.NumIn() {
			return skylark.None, fmt.Errorf("expected %d args but got %d", t.NumIn(), len(args))
		}
		v := reflect.ValueOf(gofn)
		vals, err := FromTuple(args)
		if err != nil {
			return skylark.None, err
		}
		rvs := make([]reflect.Value, 0, len(vals))
		for _, v := range vals {
			rvs = append(rvs, reflect.ValueOf(v))
		}
		out := v.Call(rvs)
		if len(out) == 0 {
			return skylark.None, nil
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
				return skylark.None, err2
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
			return skylark.None, err2
		}
		return tup, err
	})
}
