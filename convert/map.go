package convert

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"go.starlark.net/starlark"
)

// Much of this code is derived in large part from starlark-go's Dict
// implementation:
// https://github.com/google/starlark-go/blob/master/starlark/value.go#L612
// Which is Copyright 2017 The Bazel Authors and uses a BSD 3-clause license.

// GoMap is a wrapper around a Go map that makes it satisfy starlark's
// expectations of a starlark dict.
type GoMap struct {
	v      reflect.Value
	numIt  int
	frozen bool
}

// NewGoMap wraps the given map m in a new GoMap.  This function will panic if m
// is not a map.
func NewGoMap(m interface{}) *GoMap {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		panic(fmt.Errorf("NewGoMap expects a map, but got %T", m))
	}
	return &GoMap{v: v}
}

// SetKey implements starlark.HasSetKey.
func (g *GoMap) SetKey(k, v starlark.Value) (err error) {
	if g.frozen {
		return fmt.Errorf("cannot insert into frozen map")
	}
	if g.numIt > 0 {
		return fmt.Errorf("cannot insert into map during iteration")
	}

	// if you set something funky on the map, it'll panic, so we recover it here.
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("%v", r)
		}
	}()

	key := conv(k, g.v.Type().Key())
	val := conv(v, g.v.Type().Elem())
	g.v.SetMapIndex(key, val)
	return nil
}

// Get implements starlark.Mapping.
func (g *GoMap) Get(in starlark.Value) (out starlark.Value, found bool, err error) {
	v := g.v.MapIndex(conv(in, g.v.Type().Key()))
	if v.Kind() == reflect.Invalid {
		return starlark.None, false, nil
	}
	val, err := toValue(v)
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

// String returns the string representation of the value.
// Starlark string values are quoted as if by Python's repr.
func (g *GoMap) String() string {
	return fmt.Sprint(g.v.Interface())
}

// Type returns a short string describing the value's type.
func (g *GoMap) Type() string {
	return fmt.Sprintf("starlight_map<%T>", g.v.Interface())
}

// Freeze causes the value, and all values transitively
// reachable from it through collections and closures, to be
// marked as frozen.  All subsequent mutations to the data
// structure through this API will fail dynamically, making the
// data structure immutable and safe for publishing to other
// Starlark interpreters running concurrently.
func (g *GoMap) Freeze() {
	g.frozen = true
}

// Truth returns the truth value of an object.
func (g *GoMap) Truth() starlark.Bool {
	return g.v.Len() > 0
}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y).
// Hash may fail if the value's type is not hashable, or if the value
// contains a non-hashable value.
func (g *GoMap) Hash() (uint32, error) {
	return 0, errors.New("starlight_map is not hashable")
}

func (g *GoMap) Clear() error {
	if g.frozen {
		return fmt.Errorf("cannot clear frozen map")
	}
	if g.numIt > 0 {
		return fmt.Errorf("cannot clear map during iteration")
	}
	for _, k := range g.v.MapKeys() {
		g.v.SetMapIndex(k, reflect.Value{})
	}
	return nil
}

func (g *GoMap) Delete(k starlark.Value) (v starlark.Value, found bool, err error) {
	if g.frozen {
		return nil, false, fmt.Errorf("cannot delete from frozen map")
	}
	if g.numIt > 0 {
		return nil, false, fmt.Errorf("cannot delete from map during iteration")
	}
	key := conv(k, g.v.Type().Key())
	return g.delete(key)
}

func (g *GoMap) delete(key reflect.Value) (v starlark.Value, found bool, err error) {
	val := g.v.MapIndex(key)
	if val.Kind() == reflect.Invalid {
		return starlark.None, false, nil
	}
	g.v.SetMapIndex(key, reflect.Value{})

	ret, err := toValue(val)
	if err != nil {
		return starlark.None, true, err
	}
	return ret, true, nil
}

func (g *GoMap) Items() []starlark.Tuple {
	tuples := make([]starlark.Tuple, 0, g.v.Len())
	var err error
	for _, k := range g.v.MapKeys() {
		tuple := make(starlark.Tuple, 2)
		tuple[0], err = toValue(k)
		if err != nil {
			panic(err)
		}
		tuple[1], err = toValue(g.v.MapIndex(k))
		if err != nil {
			panic(err)
		}
		tuples = append(tuples, tuple)
	}
	return tuples
}

func (g *GoMap) Keys() []starlark.Value {
	keys := make([]starlark.Value, 0, g.v.Len())
	for _, k := range g.v.MapKeys() {
		key, err := toValue(k)
		if err != nil {
			panic(err)
		}
		keys = append(keys, key)
	}
	return keys
}

func (g *GoMap) Len() int {
	return g.v.Len()
}

func (g *GoMap) Iterate() starlark.Iterator {
	g.numIt++
	return &mapIterator{
		g:    g,
		keys: g.v.MapKeys(),
	}
}

func (g *GoMap) Attr(name string) (starlark.Value, error) {
	return mapAttr(g, name, dictMethods)
}

func (g *GoMap) AttrNames() []string {
	return mapAttrNames(dictMethods)
}

type builtinMapMethod func(fnname string, recv *GoMap, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

// stolen from starlark.
var dictMethods = map[string]builtinMapMethod{
	"clear":      dict_clear,
	"get":        dict_get,
	"items":      dict_items,
	"keys":       dict_keys,
	"pop":        dict_pop,
	"popitem":    dict_popitem,
	"setdefault": dict_setdefault,
	"update":     dict_update,
	"values":     dict_values,
}

func mapAttr(recv *GoMap, name string, methods map[string]builtinMapMethod) (starlark.Value, error) {
	method := methods[name]
	if method == nil {
		return nil, nil // no such method
	}

	// Allocate a closure over 'method'.
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return method(b.Name(), recv, args, kwargs)
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(recv), nil
}

func mapAttrNames(methods map[string]builtinMapMethod) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type mapIterator struct {
	g    *GoMap
	i    int
	keys []reflect.Value
}

func (it *mapIterator) Next(p *starlark.Value) bool {
	if it.i < len(it.keys) {
		v, err := toValue(it.keys[it.i])
		if err != nil {
			panic(err)
		}
		*p = v
		it.i++
		return true
	}
	return false
}

func (it *mapIterator) Done() {
	it.g.numIt--
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·get
func dict_get(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 0 || len(args) > 2 {
		return nil, fmt.Errorf("%s: got %d arguments, want 1 or 2", fnname, len(args))
	}
	v, found, err := g.Get(args[0])
	if !found && len(args) > 1 {
		// second arg is a default
		return args[1], nil
	}
	return v, err
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·clear
func dict_clear(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: wanted 0 args, got %d", fnname, len(args))
	}
	return starlark.None, g.Clear()
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·items
func dict_items(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: wanted 0 args, got %d", fnname, len(args))
	}
	items := g.Items()
	res := make([]starlark.Value, len(items))
	for i, item := range items {
		res[i] = item // convert [2]Value to Value
	}
	return starlark.NewList(res), nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·keys
func dict_keys(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: wanted 0 args, got %d", fnname, len(args))
	}
	return starlark.NewList(g.Keys()), nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·pop
func dict_pop(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 0 || len(args) > 2 {
		return nil, fmt.Errorf("%s: got %d arguments, want 1 or 2", fnname, len(args))
	}
	v, found, err := g.Delete(args[0])
	if err != nil {
		return starlark.None, err
	}
	if found {
		return v, nil
	}
	if len(args) > 1 {
		// second arg is a default
		return args[1], nil
	}
	return nil, fmt.Errorf("pop: missing key")
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·popitem
func dict_popitem(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: wanted 0 args, got %d", fnname, len(args))
	}
	keys := g.v.MapKeys()
	if len(keys) == 0 {
		return nil, fmt.Errorf("popitem: empty dict")
	}
	k := keys[0]
	v, _, err := g.delete(k)
	if err != nil {
		return nil, err
	}
	key, err := toValue(k)
	if err != nil {
		return nil, err
	}
	return starlark.Tuple{key, v}, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·setdefault
func dict_setdefault(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) == 0 || len(args) > 2 {
		return nil, fmt.Errorf("%s: got %d arguments, want 1 or 2", fnname, len(args))
	}
	var dflt starlark.Value = starlark.None
	if len(args) > 1 {
		dflt = args[1]
	}
	k := args[0]
	if v, ok, err := g.Get(k); err != nil {
		return nil, err
	} else if ok {
		return v, nil
	} else {
		return dflt, g.SetKey(k, dflt)
	}
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·update
func dict_update(fnname string, g *GoMap, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("update: got %d arguments, want at most 1", len(args))
	}
	if err := updateDict(g, args, kwargs); err != nil {
		return nil, fmt.Errorf("update: %v", err)
	}
	return starlark.None, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#dict·update
func dict_values(fnname string, g *GoMap, args starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("%s: wanted 0 args, got %d", fnname, len(args))
	}
	items := g.Items()
	res := make([]starlark.Value, len(items))
	for i, item := range items {
		res[i] = item[1]
	}
	return starlark.NewList(res), nil
}

// Common implementation of builtin dict function and dict.update method.
// Precondition: len(updates) == 0 or 1.
func updateDict(dict *GoMap, updates starlark.Tuple, kwargs []starlark.Tuple) error {
	if len(updates) == 1 {
		switch updates := updates[0].(type) {
		case starlark.NoneType:
			// no-op
		case *starlark.Dict:
			// Iterate over dict's key/value pairs, not just keys.
			for _, item := range updates.Items() {
				if err := dict.SetKey(item[0], item[1]); err != nil {
					return err // dict is frozen
				}
			}
		default:
			// all other sequences
			iter := starlark.Iterate(updates)
			if iter == nil {
				return fmt.Errorf("got %s, want iterable", updates.Type())
			}
			defer iter.Done()
			var pair starlark.Value
			for i := 0; iter.Next(&pair); i++ {
				iter2 := starlark.Iterate(pair)
				if iter2 == nil {
					return fmt.Errorf("dictionary update sequence element #%d is not iterable (%s)", i, pair.Type())

				}
				defer iter2.Done()
				len := starlark.Len(pair)
				if len < 0 {
					return fmt.Errorf("dictionary update sequence element #%d has unknown length (%s)", i, pair.Type())
				} else if len != 2 {
					return fmt.Errorf("dictionary update sequence element #%d has length %d, want 2", i, len)
				}
				var k, v starlark.Value
				iter2.Next(&k)
				iter2.Next(&v)
				if err := dict.SetKey(k, v); err != nil {
					return err
				}
			}
		}
	}

	// Then add the kwargs.
	for _, pair := range kwargs {
		if err := dict.SetKey(pair[0], pair[1]); err != nil {
			return err // dict is frozen
		}
	}

	return nil
}

// conv tries to convert v to t if v is not assignable to t.
func conv(v starlark.Value, t reflect.Type) reflect.Value {
	out := reflect.ValueOf(FromValue(v))
	if !out.Type().AssignableTo(t) {
		return out.Convert(t)
	}
	return out
}
