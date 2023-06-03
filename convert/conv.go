// Package convert provides functions for converting data and functions between Go and Starlark.
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

func hasMethods(val reflect.Value) bool {
	if val.NumMethod() > 0 {
		return true
	}
	if val.Kind() == reflect.Ptr && val.Elem().IsValid() && val.Elem().NumMethod() > 0 {
		return true
	}
	return false
}

func toValue(val reflect.Value) (result starlark.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	if val.IsValid() {
		if _, ok := val.Interface().(starlark.Value); ok {
			// let Starlark values pass through, no conversion needed
			return val.Interface().(starlark.Value), nil
		}
		if hasMethods(val) {
			// this handles all basic types with methods (numbers, strings, booleans)
			ifc, ok := makeGoInterface(val)
			if ok {
				return ifc, nil
			}
			// TODO: maps, functions, and slices with methods
		}
	}

	kind := val.Kind()
	if kind == reflect.Ptr {
		if val.Elem().IsValid() {
			kind = val.Elem().Kind()
			// for pointers to basic types, dereference them
			switch kind {
			case reflect.Bool,
				reflect.String,
				reflect.Float32, reflect.Float64,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Slice, reflect.Array, reflect.Map:
				val = val.Elem()
			}
		} else {
			// If the pointer is nil and points to a struct, make a GoInterface for it
			if val.Type().Elem().Kind() == reflect.Struct {
				return &GoInterface{v: val}, nil
			}
		}
	}

	switch kind {
	case reflect.Bool:
		return starlark.Bool(val.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return starlark.MakeInt64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return starlark.MakeUint64(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return starlark.Float(val.Float()), nil
	case reflect.Func:
		return makeStarFn("fn", val), nil
	case reflect.Map:
		return &GoMap{v: val}, nil
	case reflect.String:
		return starlark.String(val.String()), nil
	case reflect.Slice, reflect.Array:
		return &GoSlice{v: val}, nil
	case reflect.Struct:
		return &GoStruct{v: val}, nil
	case reflect.Interface:
		return &GoInterface{v: val}, nil
	case reflect.Invalid:
		return starlark.None, nil
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
		return v.BigInt()
		// buh... maybe > maxint64?  Dunno
		// panic(fmt.Errorf("can't convert starlark.Int %v to int", v))
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
	case starlark.NoneType:
		return nil
	case *GoStruct:
		return v.v.Interface()
	case *GoInterface:
		return v.v.Interface()
	case *GoMap:
		return v.v.Interface()
	case *GoSlice:
		return v.v.Interface()
	default:
		// dunno, hope it's a custom type that the receiver knows how to deal
		// with. This can happen with custom-written go types that implement
		// starlark.Value.
		// maybe it's a Starlark function.
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

// FromStringDict makes a map[string]interface{} from the given arg. Any
// inconvertible values are ignored.
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
		vk, err := toValue(k)
		if err != nil {
			return nil, err
		}

		vv, err := toValue(val.MapIndex(k))
		if err != nil {
			return nil, err
		}
		dict.SetKey(vk, vv)
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
		//ret[key] = val
		ret[key] = FromValue(val)
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

// FromKwargs converts a Python style name=val, name2=val2 list of tuples into a
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
	if gofn.Type().IsVariadic() {
		return makeVariadicStarFn(name, gofn)
	}
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (sv starlark.Value, ef error) {
		defer func() {
			if r := recover(); r != nil {
				sv = starlark.None
				ef = fmt.Errorf("panic in func %s: %v", name, r)
			}
		}()

		if len(args) != gofn.Type().NumIn() {
			return starlark.None, fmt.Errorf("expected %d args but got %d", gofn.Type().NumIn(), len(args))
		}

		// convert all the args, but kwargs are ignored
		vals := FromTuple(args)
		rvs := make([]reflect.Value, 0, len(vals))
		for i, v := range vals {
			val := reflect.ValueOf(v)
			argT := gofn.Type().In(i)

			var err error
			val, err = convertReflectValue(val, argT)
			if err != nil {
				return starlark.None, fmt.Errorf("arg %d: %v", i, err)
			}

			rvs = append(rvs, val)
		}

		out := gofn.Call(rvs)
		return makeOut(out)
	})
}

func makeVariadicStarFn(name string, gofn reflect.Value) *starlark.Builtin {
	return starlark.NewBuiltin(name, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (sv starlark.Value, ef error) {
		defer func() {
			if r := recover(); r != nil {
				sv = starlark.None
				ef = fmt.Errorf("panic in func %s: %v", name, r)
			}
		}()

		minArgs := gofn.Type().NumIn() - 1
		if len(args) < minArgs {
			return starlark.None, fmt.Errorf("expected at least %d args but got %d", minArgs, len(args))
		}

		// convert all the args, but kwargs are ignored
		vals := FromTuple(args)
		rvs := make([]reflect.Value, 0, len(args))

		// grab all the non-variadics first
		for i := 0; i < minArgs; i++ {
			val := reflect.ValueOf(vals[i])
			argT := gofn.Type().In(i)

			var err error
			val, err = convertReflectValue(val, argT)
			if err != nil {
				return starlark.None, fmt.Errorf("arg %d: %v", i, err)
			}

			rvs = append(rvs, val)
		}
		// last "in" type by definition must be a slice of something. We need to
		// know what something, so we can convert things as needed.
		vtype := gofn.Type().In(gofn.Type().NumIn() - 1).Elem()
		// the rest of the args need to be batched into a slice for the variadic
		for i := minArgs; i < len(vals); i++ {
			val := reflect.ValueOf(vals[i])

			var err error
			val, err = convertReflectValue(val, vtype)
			if err != nil {
				return starlark.None, fmt.Errorf("arg %d: %v", i, err)
			}
			rvs = append(rvs, val)
		}
		out := gofn.Call(rvs)
		return makeOut(out)
	})
}

func makeOut(out []reflect.Value) (starlark.Value, error) {
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
	if len(out) == 0 {
		return starlark.None, err
	}
	if len(out) == 1 {
		v, err2 := toValue(out[0])
		if err2 != nil {
			return starlark.None, err2
		}
		return v, err
	}
	// tuple-up multiple values
	res := make([]starlark.Value, 0, len(out))
	for i := range out {
		val, err3 := toValue(out[i])
		if err3 != nil {
			return starlark.None, err3
		}
		res = append(res, val)
	}
	return starlark.Tuple(res), err
}

// convertReflectValue converts a reflect.Value to a given type.
func convertReflectValue(val reflect.Value, argT reflect.Type) (reflect.Value, error) {
	if !val.IsValid() {
		return reflect.Zero(argT), nil
	}
	if val.Type().AssignableTo(argT) {
		return val, nil
	}
	if val.Type().ConvertibleTo(argT) {
		return val.Convert(argT), nil
	}
	if val.Kind() == reflect.Slice && argT.Kind() == reflect.Slice {
		return convertSlice(val, argT)
	}
	if val.Kind() == reflect.Map && argT.Kind() == reflect.Map {
		return convertMap(val, argT)
	}
	return reflect.Value{}, fmt.Errorf("expected type %v got %v", argT, val.Type())
}

func convertSlice(val reflect.Value, argT reflect.Type) (reflect.Value, error) {
	argElem := argT.Elem()
	valLen := val.Len()
	newSlice := reflect.MakeSlice(argT, valLen, valLen)

	for i := 0; i < valLen; i++ {
		elem := val.Index(i)

		if elem.Type().AssignableTo(argElem) {
			newSlice.Index(i).Set(elem)
		} else if elem.Type().ConvertibleTo(argElem) {
			newSlice.Index(i).Set(elem.Convert(argElem))
		} else if elem.Elem().Type().ConvertibleTo(argElem) {
			newSlice.Index(i).Set(elem.Elem().Convert(argElem))
		} else {
			return reflect.Value{}, fmt.Errorf("expected slice element type %v got %v", argElem, elem.Type())
		}
	}

	return newSlice, nil
}

func convertMap(val reflect.Value, argT reflect.Type) (reflect.Value, error) {
	argKey := argT.Key()
	argElem := argT.Elem()
	newMap := reflect.MakeMapWithSize(argT, val.Len())

	for _, key := range val.MapKeys() {
		newKey, err := convertElemValue(key, argKey)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("map key conversion failed: %v", err)
		}

		valElem := val.MapIndex(key)
		newElem, err := convertElemValue(valElem, argElem)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("map value conversion failed: %v", err)
		}

		newMap.SetMapIndex(newKey, newElem)
	}

	return newMap, nil
}

func convertElemValue(val reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	if val.Type().AssignableTo(targetType) || val.Type().ConvertibleTo(targetType) {
		return val.Convert(targetType), nil
	} else if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return reflect.Value{}, fmt.Errorf("nil value cannot be converted to type %v", targetType)
		}
		if val.Elem().Type().ConvertibleTo(targetType) {
			return val.Elem().Convert(targetType), nil
		} else if val.Type().Kind() == reflect.Interface {
			unwrapped := val.Elem()
			if unwrapped.Type().ConvertibleTo(targetType) {
				return unwrapped.Convert(targetType), nil
			} else if sv, ok := unwrapped.Interface().(starlark.Value); ok {
				// TODO: this path is not reachable in the current test, maybe we can remove it?
				goVal := FromValue(sv)
				goVal = convertNumericTypes(goVal, targetType)
				if reflect.TypeOf(goVal) != targetType {
					return reflect.Value{}, fmt.Errorf("expected type %v got %v", targetType, reflect.TypeOf(goVal))
				}
				return reflect.ValueOf(goVal), nil
			}
		}
	}
	return reflect.Value{}, fmt.Errorf("expected type %v got %v", targetType, val.Type())
}

func convertNumericTypes(value interface{}, targetType reflect.Type) interface{} {
	// If the value is an integer, convert it to the appropriate integer type.
	switch st := value.(type) {
	case int64:
		switch targetType.Kind() {
		case reflect.Int:
			return int(st)
		case reflect.Int32:
			return int32(st)
		case reflect.Int16:
			return int16(st)
		case reflect.Int8:
			return int8(st)
		case reflect.Uint:
			return uint(st)
		case reflect.Uint64:
			return uint64(st)
		case reflect.Uint32:
			return uint32(st)
		case reflect.Uint16:
			return uint16(st)
		case reflect.Uint8:
			return uint8(st)
		}
	// If the value is a float, convert it to the appropriate float type.
	case float64:
		if targetType.Kind() == reflect.Float32 {
			return float32(st)
		}
	}
	return value
}

// tryConv tries to convert starlark.Value v to Go t if v is not assignable to t.
func tryConv(v starlark.Value, t reflect.Type) (reflect.Value, error) {
	if v == starlark.None {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func:
			return reflect.Zero(t), nil
		default:
			return reflect.Value{}, fmt.Errorf("value of type None cannot be converted to non-nullable type %s", t)
		}
	}
	out := reflect.ValueOf(FromValue(v))
	if !out.Type().AssignableTo(t) {
		if out.Type().ConvertibleTo(t) {
			return out.Convert(t), nil
		}
		return reflect.Value{}, fmt.Errorf("value of type %s cannot be converted to type %s", out.Type(), t)
	}
	return out, nil
}
