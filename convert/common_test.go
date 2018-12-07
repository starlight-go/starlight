package convert_test

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/starlight-go/starlight"
)

type assert struct {
	t *testing.T
}

func (a *assert) Eq(expected, got interface{}) {
	if !reflect.DeepEqual(expected, got) {
		_, file, line, _ := runtime.Caller(13)
		a.t.Fatalf("\n%v:%v: - expected %#v (%T) to be equal to %#v (%T)\n", file, line, expected, expected, got, got)
	}
}

type fail struct {
	code string
	err  string
}

func expectFails(t *testing.T, tests []fail, globals map[string]interface{}) {
	for _, f := range tests {
		t.Run(f.code, func(t *testing.T) {
			_, err := starlight.Eval([]byte(f.code), globals, nil)
			expectErr(t, err, f.err)
		})
	}
}
