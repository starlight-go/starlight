package convert_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/starlight-go/starlight"
)

type mega struct {
	Bool   bool
	Int    int
	Int64  int64
	Body   io.Reader
	String string
	Map    map[string]string
	Time   time.Time
	Now    func() time.Time
	Bytes  []byte
}

func (m *mega) GetTime() time.Time {
	return m.Time
}

func (m *mega) getBool() bool {
	return m.Bool
}

func TestStructs(t *testing.T) {
	m := &mega{
		Bool:  true,
		Int:   1,
		Int64: 2,
		Body:  strings.NewReader("hi!"),
		Map:   map[string]string{"foo": "bar"},
		Time:  time.Now(),
		Now:   time.Now,
		Bytes: []byte("hi!"),
	}
	globals := map[string]interface{}{
		"m":          m,
		"assert":     &assert{t: t},
		"bytesEqual": bytes.Equal,
		"readAll":    ioutil.ReadAll,
	}

	code := []byte(`
assert.Eq(m.Bool, True)
assert.Eq(m.Int, 1)
assert.Eq(m.Int64, 2)
assert.Eq(m.Map["foo"], "bar")
assert.Eq(m.Time.Year(), m.Now().Year())
assert.Eq(m.GetTime().Year(), m.Now().Year())
assert.Eq(True, bytesEqual(readAll(m.Body), m.Bytes))
`)

	_, err := starlight.Eval(code, globals, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCantCallUnexported(t *testing.T) {
	code := []byte(`
a = m.getBool()
`)
	globals := map[string]interface{}{
		"m": &mega{},
	}
	_, err := starlight.Eval(code, globals, nil)
	expectErr(t, err, "starlight_struct<*convert_test.mega> has no .getBool field or method")
}
