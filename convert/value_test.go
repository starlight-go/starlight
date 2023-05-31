package convert

import (
	"math/big"
	"reflect"
	"testing"

	"go.starlark.net/starlark"
)

func TestToValue(t *testing.T) {
	bigVal := big.NewInt(1).Mul(big.NewInt(100000000000000), big.NewInt(100000000000000))
	tests := []struct {
		name    string
		v       interface{}
		want    starlark.Value
		wantErr bool
	}{
		{
			name:    "nil to none",
			v:       nil,
			want:    starlark.None,
			wantErr: false,
		},
		{
			name:    "typed nil to none",
			v:       (*string)(nil),
			wantErr: true,
		},
		{
			name:    "starlark typed nil to none",
			v:       (*starlark.Value)(nil),
			wantErr: true,
		},
		{
			name:    "starlark none value",
			v:       starlark.None,
			want:    starlark.None,
			wantErr: false,
		},
		{
			name:    "starlark string value",
			v:       starlark.String("test"),
			want:    starlark.String("test"),
			wantErr: false,
		},
		{
			name:    "starlark int value",
			v:       starlark.MakeInt(123),
			want:    starlark.MakeInt(123),
			wantErr: false,
		},
		{
			name:    "string to value",
			v:       "test",
			want:    starlark.String("test"),
			wantErr: false,
		},
		{
			name:    "int to value",
			v:       123,
			want:    starlark.MakeInt(123),
			wantErr: false,
		},
		{
			name: "not big int to value",
			v:    big.NewInt(123),
			want: starlark.MakeInt(123),
		},
		{
			name: "big int to value",
			v:    bigVal,
			want: starlark.MakeBigInt(bigVal),
		},
		{
			name:    "bool to value",
			v:       true,
			want:    starlark.Bool(true),
			wantErr: false,
		},
		{
			name:    "float to value",
			v:       123.45,
			want:    starlark.Float(123.45),
			wantErr: false,
		},
		{
			name:    "big float to value",
			v:       big.NewFloat(123.456),
			want:    starlark.Float(123.456),
			wantErr: false,
		},
		{
			name:    "slice to value",
			v:       []int{1, 2, 3},
			want:    &GoSlice{v: reflect.ValueOf([]int{1, 2, 3})},
			wantErr: false,
		},
		{
			name:    "array to value",
			v:       [3]int{1, 2, 3},
			want:    &GoSlice{v: reflect.ValueOf([]int{1, 2, 3})},
			wantErr: false,
		},
		{
			name:    "map to value",
			v:       map[string]int{"one": 1, "two": 2},
			want:    &GoMap{v: reflect.ValueOf(map[string]int{"one": 1, "two": 2})},
			wantErr: false,
		},
		{
			name:    "map slice to value",
			v:       map[string][]int{"one": {1, 2}, "two": {3, 4}},
			want:    &GoMap{v: reflect.ValueOf(map[string][]int{"one": {1, 2}, "two": {3, 4}})},
			wantErr: false,
		},
		{
			name:    "empty struct to value",
			v:       struct{}{},
			want:    &GoStruct{v: reflect.ValueOf(struct{}{})},
			wantErr: false,
		},
		{
			name:    "custom struct to value",
			v:       struct{ Name string }{Name: "test"},
			want:    &GoStruct{v: reflect.ValueOf(struct{ Name string }{Name: "test"})},
			wantErr: false,
		},
		{
			name:    "lib struct to value",
			v:       big.NewRat(1, 3),
			want:    &GoStruct{v: reflect.ValueOf(big.NewRat(1, 3))},
			wantErr: false,
		},
		{
			name:    "function to value",
			v:       func() string { return "test" },
			want:    makeStarFn("fn", reflect.ValueOf(func() string { return "test" })),
			wantErr: false,
		},
		{
			name:    "unsupported type: channel",
			v:       make(chan int),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToValue(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToValue(%v) error = %v, wantErr %v", tt.v, err, tt.wantErr)
				return
			}
			if !(reflect.DeepEqual(got, tt.want) || got.String() == tt.want.String()) {
				t.Errorf("ToValue(%v) got = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestFromValue(t *testing.T) {
	slDict := starlark.NewDict(2)
	slDict.SetKey(starlark.String("a"), starlark.String("b"))

	slSet := starlark.NewSet(2)
	slSet.Insert(starlark.String("a"))
	slSet.Insert(starlark.String("b"))

	testBuiltin := makeStarFn("fn", reflect.ValueOf(func() string { return "test" }))
	testFunction := getSimpleStarlarkFunc()

	bigVal := big.NewInt(1).Mul(big.NewInt(100000000000000), big.NewInt(100000000000000))

	tests := []struct {
		name string
		v    starlark.Value
		want interface{}
	}{
		{
			name: "Bool",
			v:    starlark.Bool(true),
			want: true,
		},
		{
			name: "Int",
			v:    starlark.MakeInt(123),
			want: int64(123),
		},
		{
			name: "BigInt",
			v:    starlark.MakeBigInt(bigVal),
			want: bigVal,
		},
		{
			name: "Float",
			v:    starlark.Float(1.23),
			want: float64(1.23),
		},
		{
			name: "String",
			v:    starlark.String("hello"),
			want: "hello",
		},
		{
			name: "List",
			v:    starlark.NewList([]starlark.Value{starlark.String("a"), starlark.String("b")}),
			want: []interface{}{"a", "b"},
		},
		{
			name: "Tuple",
			v:    starlark.Tuple([]starlark.Value{starlark.String("a"), starlark.String("b")}),
			want: []interface{}{"a", "b"},
		},
		{
			name: "Dict",
			v:    slDict,
			want: map[interface{}]interface{}{"a": "b"},
		},
		{
			name: "Set",
			v:    slSet,
			want: map[interface{}]bool{"a": true, "b": true},
		},
		{
			name: "None",
			v:    starlark.None,
			want: nil,
		},
		// for GoStruct, GoInterface, GoMap, and GoSlice, we're assuming they just hold an interface{}
		{
			name: "GoStruct",
			v:    &GoStruct{v: reflect.ValueOf("hello")},
			want: "hello",
		},
		{
			name: "GoInterface",
			v:    &GoInterface{v: reflect.ValueOf(123)},
			want: 123,
		},
		{
			name: "GoMap",
			v:    &GoMap{v: reflect.ValueOf(map[string]int{"a": 1})},
			want: map[string]int{"a": 1},
		},
		{
			name: "GoSlice",
			v:    &GoSlice{v: reflect.ValueOf([]int{1, 2, 3})},
			want: []int{1, 2, 3},
		},
		{
			name: "Default",
			v:    &customType{}, // assuming customType is a starlark.Value
			want: &customType{}, // assuming FromValue returns the original value if it doesn't know how to convert it
		},
		{
			name: "Builtin",
			v:    testBuiltin,
			want: testBuiltin,
		},
		{
			name: "Function",
			v:    testFunction,
			want: testFunction,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromValue(tt.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromValue(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

// Assuming this is a custom type that implements starlark.Value
type customType struct{}

func (c *customType) String() string        { return "customType" }
func (c *customType) Type() string          { return "customType" }
func (c *customType) Freeze()               {}
func (c *customType) Truth() starlark.Bool  { return starlark.True }
func (c *customType) Hash() (uint32, error) { return 0, nil }

// Assuming this is a custom type that doesn't implement starlark.Callable
type unknownType struct{}

// Generate Starlark Functions

func getSimpleStarlarkFunc() *starlark.Function {
	code := `
def double(x):
	return x*2
`
	thread := &starlark.Thread{Name: "test"}
	globals, err := starlark.ExecFile(thread, "mock.star", code, nil)
	if err != nil {
		panic(err)
	}
	return globals["double"].(*starlark.Function)
}

func TestMakeDict(t *testing.T) {
	sd1 := starlark.NewDict(1)
	_ = sd1.SetKey(starlark.String("a"), starlark.String("b"))

	sd2 := starlark.NewDict(1)
	_ = sd2.SetKey(starlark.String("a"), starlark.MakeInt(1))

	vf3 := 2
	sd3 := starlark.NewDict(1)
	_ = sd3.SetKey(starlark.String("a"), starlark.Float(vf3))

	sd4 := starlark.NewDict(1)
	_ = sd4.SetKey(starlark.String("a"), NewGoSlice([]string{"b", "c"}))

	sd5 := starlark.NewDict(1)
	_ = sd5.SetKey(starlark.String("a"), MakeGoInterface("b"))

	sd6 := starlark.NewDict(1)
	_ = sd6.SetKey(starlark.MakeInt(10), starlark.Tuple{starlark.String("a")})

	tests := []struct {
		name    string
		v       interface{}
		want    starlark.Value
		wantErr bool
	}{
		{
			name: "map[string]string",
			v:    map[string]string{"a": "b"},
			want: sd1,
		},
		{
			name: "map[string]int",
			v:    map[string]int{"a": 1},
			want: sd2,
		},
		{
			name: "map[string]float32",
			v:    map[string]float32{"a": float32(vf3)},
			want: sd3,
		},
		{
			name: "map[string][]string",
			v:    map[string][]string{"a": {"b", "c"}},
			want: sd4,
		},
		{
			name: "map[string]interface{}",
			v:    map[string]interface{}{"a": "b"},
			want: sd5,
		},
		{
			name: "map[starlark.String]starlark.String",
			v:    map[starlark.String]starlark.String{"a": starlark.String("b")},
			want: sd1,
		},
		{
			name: "map[starlark.Int]starlark.Tuple",
			v:    map[starlark.Int]starlark.Tuple{starlark.MakeInt(10): {starlark.String("a")}},
			want: sd6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeDict(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeDict(%v) error = %v, wantErr %v", tt.v, err, tt.wantErr)
				return
			}
			if !(reflect.DeepEqual(got, tt.want) || got.String() == tt.want.String()) {
				t.Errorf("MakeDict(%v) got = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestFromDict(t *testing.T) {
	sd1 := starlark.NewDict(1)
	_ = sd1.SetKey(starlark.String("a"), starlark.String("b"))

	sd2 := starlark.NewDict(1)
	_ = sd2.SetKey(starlark.String("a"), starlark.MakeInt(1))

	vf3 := 2
	sd3 := starlark.NewDict(1)
	_ = sd3.SetKey(starlark.String("a"), starlark.Float(vf3))

	sd4 := starlark.NewDict(1)
	_ = sd4.SetKey(starlark.String("a"), NewGoSlice([]string{"b", "c"}))

	sd5 := starlark.NewDict(1)
	_ = sd5.SetKey(starlark.String("a"), MakeGoInterface("b"))

	sd6 := starlark.NewDict(1)
	_ = sd6.SetKey(starlark.MakeInt(10), starlark.Tuple{starlark.String("a")})

	tests := []struct {
		name string
		v    *starlark.Dict
		want map[interface{}]interface{}
	}{
		{
			name: "map[string]string",
			v:    sd1,
			want: map[interface{}]interface{}{"a": "b"},
		},
		{
			name: "map[string]int64",
			v:    sd2,
			want: map[interface{}]interface{}{"a": int64(1)},
		},
		{
			name: "map[string]float32",
			v:    sd3,
			want: map[interface{}]interface{}{"a": float64(vf3)},
		},
		{
			name: "map[string][]string",
			v:    sd4,
			want: map[interface{}]interface{}{"a": []string{"b", "c"}},
		},
		{
			name: "map[string]interface{}",
			v:    sd5,
			want: map[interface{}]interface{}{"a": "b"},
		},
		{
			name: "map[starlark.String]starlark.String",
			v:    sd1,
			want: map[interface{}]interface{}{"a": "b"},
		},
		{
			name: "map[starlark.Int]starlark.Tuple",
			v:    sd6,
			want: map[interface{}]interface{}{int64(10): []interface{}{"a"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromDict(tt.v)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromDict(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestMakeSet(t *testing.T) {
	s1 := starlark.NewSet(1)
	_ = s1.Insert(starlark.String("a"))

	s2 := starlark.NewSet(1)
	_ = s2.Insert(starlark.MakeInt(1))

	s3 := starlark.NewSet(2)
	_ = s3.Insert(starlark.String("a"))
	_ = s3.Insert(starlark.String("b"))

	tests := []struct {
		name    string
		s       map[interface{}]bool
		want    *starlark.Set
		wantErr bool
	}{
		{
			name: "set[string]",
			s:    map[interface{}]bool{"a": true},
			want: s1,
		},
		{
			name: "set[int]",
			s:    map[interface{}]bool{1: true},
			want: s2,
		},
		{
			name: "set[string,string]",
			s:    map[interface{}]bool{"a": true, "b": true},
			want: s3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeSet(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeSet(%v) error = %v, wantErr %v", tt.s, err, tt.wantErr)
				return
			}
			if eq, err := starlark.Equal(got, tt.want); !eq || err != nil {
				t.Errorf("MakeSet(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestFromSet(t *testing.T) {
	s1 := starlark.NewSet(1)
	_ = s1.Insert(starlark.String("a"))

	s2 := starlark.NewSet(1)
	_ = s2.Insert(starlark.MakeInt(200))

	s3 := starlark.NewSet(2)
	_ = s3.Insert(starlark.String("a"))
	_ = s3.Insert(starlark.String("b"))

	tests := []struct {
		name string
		s    *starlark.Set
		want map[interface{}]bool
	}{
		{
			name: "set[string]",
			s:    s1,
			want: map[interface{}]bool{"a": true},
		},
		{
			name: "set[int]",
			s:    s2,
			want: map[interface{}]bool{int64(200): true},
		},
		{
			name: "set[string,string]",
			s:    s3,
			want: map[interface{}]bool{"a": true, "b": true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromSet(tt.s)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromSet(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestFromTuple(t *testing.T) {
	tuple1 := starlark.Tuple{starlark.String("a")}
	tuple2 := starlark.Tuple{starlark.MakeInt(100)}
	tuple3 := starlark.Tuple{starlark.String("a"), starlark.String("b")}
	tests := []struct {
		name string
		v    starlark.Tuple
		want []interface{}
	}{
		{
			name: "tuple[string]",
			v:    tuple1,
			want: []interface{}{"a"},
		},
		{
			name: "tuple[int]",
			v:    tuple2,
			want: []interface{}{int64(100)},
		},
		{
			name: "tuple[string, string]",
			v:    tuple3,
			want: []interface{}{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromTuple(tt.v)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromTuple(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func TestFromList(t *testing.T) {
	l1 := starlark.NewList([]starlark.Value{starlark.String("a")})
	l2 := starlark.NewList([]starlark.Value{starlark.MakeInt(200)})
	l3 := starlark.NewList([]starlark.Value{starlark.String("a"), starlark.String("b")})
	tests := []struct {
		name string
		l    *starlark.List
		want []interface{}
	}{
		{
			name: "list[string]",
			l:    l1,
			want: []interface{}{"a"},
		},
		{
			name: "list[int]",
			l:    l2,
			want: []interface{}{int64(200)},
		},
		{
			name: "list[string, string]",
			l:    l3,
			want: []interface{}{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromList(tt.l)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromList(%v) = %v, want %v", tt.l, got, tt.want)
			}
		})
	}
}
