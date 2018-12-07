package convert

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"go.starlark.net/starlark"
)

// Much of this code is derived in large part from starlark-go's List
// implementation:
// https://github.com/google/starlark-go/blob/master/starlark/value.go#L666
// Which is Copyright 2017 The Bazel Authors and uses a BSD 3-clause license.

// GoSlice is a wrapper around a Go slice to adapt it for use with starlark.
type GoSlice struct {
	v      reflect.Value
	numIt  int
	frozen bool
}

// NewGoMap wraps the given slice in a new GoSlice.  This function will panic if m
// is not a map.
func NewGoSlice(slice interface{}) *GoSlice {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice || v.Kind() != reflect.Array {
		panic(fmt.Errorf("NewGoSlice expects a slice or array, but got %T", slice))
	}
	return &GoSlice{v: v}
}

// String returns the string representation of the value.
// Starlark string values are quoted as if by Python's repr.
func (g *GoSlice) String() string {
	return fmt.Sprint(g.v.Interface())
}

// Type returns a short string describing the value's type.
func (g *GoSlice) Type() string {
	return fmt.Sprintf("starlight_slice<%T>", g.v.Interface())
}

// Freeze causes the value, and all values transitively
// reachable from it through collections and closures, to be
// marked as frozen.  All subsequent mutations to the data
// structure through this API will fail dynamically, making the
// data structure immutable and safe for publishing to other
// Starlark interpreters running concurrently.
func (g *GoSlice) Freeze() {
	g.frozen = true
}

// Truth returns the truth value of an object.
func (g *GoSlice) Truth() starlark.Bool {
	return g.v.Len() > 0
}

// Hash returns a function of x such that Equals(x, y) => Hash(x) == Hash(y).
// Hash may fail if the value's type is not hashable, or if the value
// contains a non-hashable value.
func (g *GoSlice) Hash() (uint32, error) {
	return 0, errors.New("starlight_slice is not hashable")
}

func (g *GoSlice) Clear() error {
	if err := g.checkMutable("clear"); err != nil {
		return err
	}
	g.v = g.v.Slice(0, 0)
	return nil
}

func (g *GoSlice) Index(i int) starlark.Value {
	v, err := toValue(g.v.Index(i))
	if err != nil {
		panic(err)
	}
	return v
}

func (g *GoSlice) SetIndex(index int, v starlark.Value) error {
	if err := g.checkMutable("assign to"); err != nil {
		return err
	}
	val := conv(v, g.v.Type().Elem())
	g.v.Index(index).Set(val)
	return nil
}

func (g *GoSlice) Slice(start, end, step int) starlark.Value {
	// python slices are copies, so we don't just use .Slice here
	if step == 1 {
		copy := reflect.MakeSlice(g.v.Type(), end-start, end-start)
		reflect.Copy(copy, g.v.Slice(start, end))
		return &GoSlice{v: copy}
	}
	copy := reflect.MakeSlice(g.v.Type().Elem(), 0, 0)
	sign := signOf(step)
	for i := start; signOf(end-i) == sign; i += step {
		copy = reflect.Append(copy, g.v.Index(i))
	}
	return &GoSlice{v: copy}
}

func signOf(i int) int {
	// yeah, sorry, I'm not doing this the hacker way.
	switch {
	case i == 0:
		return 0
	case i < 0:
		return -1
	default:
		return 1
	}
}

func (g *GoSlice) Len() int {
	return g.v.Len()
}

func (g *GoSlice) Iterate() starlark.Iterator {
	g.numIt++
	return &sliceIterator{
		g: g,
	}
}

func (g *GoSlice) Attr(name string) (starlark.Value, error) {
	return sliceAttr(g, name, sliceMethods)
}

func (g *GoSlice) AttrNames() []string {
	return sliceAttrNames(sliceMethods)
}

type builtinSliceMethod func(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error)

var sliceMethods = map[string]builtinSliceMethod{
	"append": list_append,
	"clear":  list_clear,
	"extend": list_extend,
	"index":  list_index,
	"insert": list_insert,
	"pop":    list_pop,
	"remove": list_remove,
}

func sliceAttr(g *GoSlice, name string, methods map[string]builtinSliceMethod) (starlark.Value, error) {
	method := methods[name]
	if method == nil {
		return nil, nil // no such method
	}

	// Allocate a closure over 'method'.
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		return method(b.Name(), g, args, kwargs)
	}
	return starlark.NewBuiltin(name, impl).BindReceiver(g), nil
}

func sliceAttrNames(methods map[string]builtinSliceMethod) []string {
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type sliceIterator struct {
	g *GoSlice
	i int
}

func (it *sliceIterator) Next(p *starlark.Value) bool {
	if it.i < it.g.v.Len() {
		v, err := toValue(it.g.v.Index(it.i))
		if err != nil {
			panic(err)
		}
		*p = v
		it.i++
		return true
	}
	return false
}

// checkMutable reports an error if the slicve should not be mutated.
// verb+" slice" should describe the operation.
func (g *GoSlice) checkMutable(verb string) error {
	if g.frozen {
		return fmt.Errorf("cannot %s frozen slice", verb)
	}
	if g.numIt > 0 {
		return fmt.Errorf("cannot %s slice during iteration", verb)
	}
	return nil
}

func (it *sliceIterator) Done() {
	it.g.numIt--
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·append
func list_append(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("append: got %d arguments, want 1", len(args))
	}
	if err := g.checkMutable("append to"); err != nil {
		return nil, err
	}
	v := conv(args[0], g.v.Type().Elem())
	g.v = reflect.Append(g.v, v)
	return starlark.None, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·clear
func list_clear(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("clear: got %d arguments, want 0", len(args))
	}
	return starlark.None, g.Clear()
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·extend
func list_extend(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("extend: got %d arguments, want 1", len(args))
	}
	if err := g.checkMutable("extend"); err != nil {
		return nil, err
	}
	iterable, ok := args[0].(starlark.Iterable)
	if !ok {
		return nil, fmt.Errorf("argument is not iterable: %#v", args[0])
	}
	var val starlark.Value
	it := iterable.Iterate()
	defer it.Done()
	for it.Next(&val) {
		v := conv(val, g.v.Type().Elem())
		g.v = reflect.Append(g.v, v)
	}

	return starlark.None, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·index
func list_index(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var start_, end_ starlark.Value
	switch len(args) {
	default:
		return nil, fmt.Errorf("index: expected 1-3 args, got %d", len(args))
	case 3:
		end_ = args[2]
		fallthrough
	case 2:
		start_ = args[1]
		fallthrough
	case 1:
		// ok
	}
	value := conv(args[0], g.v.Type().Elem())
	start, end, err := indices(start_, end_, g.v.Len())
	if err != nil {
		return nil, fmt.Errorf("%s: %s", fnname, err)
	}

	for i := start; i < end; i++ {
		if reflect.DeepEqual(g.v.Index(i).Interface(), value.Interface()) {
			return starlark.MakeInt(i), nil
		}
	}
	return nil, fmt.Errorf("index: value %v not in list", value)
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·insert
func list_insert(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("extend: got %d arguments, want 2", len(args))
	}
	if err := g.checkMutable("insert into"); err != nil {
		return nil, err
	}

	index, err := toInt(args[0])
	if err != nil {
		return nil, err
	}
	if index < 0 {
		index += g.v.Len()
	}

	val := conv(args[1], g.v.Type().Elem())
	if index >= g.Len() {
		g.v = reflect.Append(g.v, val)
	} else {
		if index < 0 {
			index = 0 // start
		}
		g.v = reflect.Append(g.v, reflect.Zero(g.v.Type().Elem()))
		reflect.Copy(g.v.Slice(index+1, g.v.Len()), g.v.Slice(index, g.v.Len())) // slide up one
		g.v.Index(index).Set(val)
	}
	return starlark.None, nil
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·remove
func list_remove(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("remove: got %d arguments, want 1", len(args))
	}
	if err := g.checkMutable("remove from"); err != nil {
		return nil, err
	}

	v := conv(args[0], g.v.Type().Elem()).Interface()
	for i := 0; i < g.v.Len(); i++ {
		elem := g.v.Index(i)
		if reflect.DeepEqual(elem.Interface(), v) {
			g.v = reflect.AppendSlice(g.v.Slice(0, i), g.v.Slice(i+1, g.v.Len()))
			return starlark.None, nil
		}
	}
	return nil, fmt.Errorf("remove: element %v not found", v)
}

// https://github.com/google/starlark-go/blob/master/doc/spec.md#list·pop
func list_pop(fnname string, g *GoSlice, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	index := g.v.Len() - 1
	switch len(args) {
	case 0:
		// ok
	case 1:
		var err error
		index, err = toInt(args[0])
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("pop: expected 0 or 1 args, but got %d", len(args))
	}
	if index < 0 || index >= g.v.Len() {
		return nil, fmt.Errorf("pop: index %d is out of range [0:%d]", index, g.v.Len())
	}
	if err := g.checkMutable("pop from"); err != nil {
		return nil, err
	}
	// convert this out before reslicing, otherwise the value changes out from under us.
	res, err := toValue(g.v.Index(index))
	if err != nil {
		return nil, err
	}
	g.v = reflect.AppendSlice(g.v.Slice(0, index), g.v.Slice(index+1, g.v.Len()))
	return res, nil
}

// indices converts start_ and end_ to indices in the range [0:len].
// The start index defaults to 0 and the end index defaults to len.
// An index -len < i < 0 is treated like i+len.
// All other indices outside the range are clamped to the nearest value in the range.
// Beware: start may be greater than end.
// This function is suitable only for slices with positive strides.
func indices(start_, end_ starlark.Value, len int) (start, end int, err error) {
	start = 0
	if err := asIndex(start_, len, &start); err != nil {
		return 0, 0, fmt.Errorf("invalid start index: %s", err)
	}
	// Clamp to [0:len].
	if start < 0 {
		start = 0
	} else if start > len {
		start = len
	}

	end = len
	if err := asIndex(end_, len, &end); err != nil {
		return 0, 0, fmt.Errorf("invalid end index: %s", err)
	}
	// Clamp to [0:len].
	if end < 0 {
		end = 0
	} else if end > len {
		end = len
	}

	return start, end, nil
}

// asIndex sets *result to the integer value of v, adding len to it
// if it is negative.  If v is nil or None, *result is unchanged.
func asIndex(v starlark.Value, len int, result *int) error {
	if v != nil && v != starlark.None {
		var err error
		*result, err = starlark.AsInt32(v)
		if err != nil {
			return fmt.Errorf("got %s, want int", v.Type())
		}
		if *result < 0 {
			*result += len
		}
	}
	return nil
}

func toInt(v starlark.Value) (int, error) {
	idx, ok := v.(starlark.Int)
	if !ok {
		return 0, fmt.Errorf("index must be a number, but was %T", v)
	}
	i, ok := idx.Int64()
	if !ok {
		return 0, fmt.Errorf("index not representable as int64")
	}
	return int(i), nil
}
