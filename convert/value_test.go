package convert

import (
	"reflect"
	"testing"

	"go.starlark.net/starlark"
)

func TestToValue(t *testing.T) {
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
			want:    starlark.None,
			wantErr: false,
		},
		{
			name:    "starlark typed nil to none",
			v:       (*starlark.Value)(nil),
			want:    starlark.None,
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
			name:    "slice to value",
			v:       []int{1, 2, 3},
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
			name:    "struct to value",
			v:       struct{ Name string }{Name: "test"},
			want:    &GoStruct{v: reflect.ValueOf(struct{ Name string }{Name: "test"})},
			wantErr: false,
		},
		{
			name:    "function to value",
			v:       func() string { return "test" },
			want:    makeStarFn("fn", reflect.ValueOf(func() string { return "test" })),
			wantErr: false,
		},
		{
			name:    "unsupported type",
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
		//{
		//	name: "Dict",
		//	v:    slDict,
		//	want: map[interface{}]interface{}{"a": "b"},
		//},
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
			// The last test case is not completed. Let's complete it.
			want: &customType{}, // assuming FromValue returns the original value if it doesn't know how to convert it
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
