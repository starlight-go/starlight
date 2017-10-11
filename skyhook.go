// Package skyhook provides a convenience wrapper around github.com/google/skylark.
package skyhook

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/google/skylark"
	"github.com/google/skylark/resolve"
	"github.com/google/skylark/syntax"
	"github.com/hippogryph/skyhook/convert"
)

func init() {
	resolve.AllowNestedDef = true // allow def statements within function bodies
	resolve.AllowLambda = true    // allow lambda expressions
	resolve.AllowFloat = true     // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowFreeze = true    // allow the 'freeze' built-in
	resolve.AllowSet = true       // allow the 'set' built-in
}

// Eval evaluates the skylark source with the given global variables. The type
// of the argument for the src parameter must be string, []byte, or io.Reader.
func Eval(src interface{}, globals map[string]interface{}) (map[string]interface{}, error) {
	dict, err := convert.MakeStringDict(globals)
	if err != nil {
		return nil, err
	}
	if err := skylark.ExecFile(new(skylark.Thread), "eval.sky", src, dict); err != nil {
		return nil, err
	}
	return convert.FromStringDict(dict), nil
}

// Skyhook is a script/plugin runner.
type Skyhook struct {
	dirs     []string
	readFile func(filename string) ([]byte, error)

	mu      *sync.Mutex
	plugins map[string]*syntax.File
}

func parseFile(name string, b []byte, dict skylark.StringDict) (*syntax.File, error) {
	f, err := syntax.Parse(name, b)
	if err != nil {
		return nil, err
	}

	isPredeclaredGlobal := func(name string) bool { _, ok := dict[name]; return ok } // x, but not y
	isBuiltin := func(name string) bool { return skylark.Universe[name] != nil }

	if err := resolve.File(f, isPredeclaredGlobal, isBuiltin); err != nil {
		return nil, err
	}
	return f, nil
}

func runFile(f *syntax.File, globals skylark.StringDict) (map[string]interface{}, error) {
	thread := new(skylark.Thread)
	fr := thread.Push(globals, len(f.Locals))
	defer thread.Pop()
	if err := fr.ExecStmts(f.Stmts); err != nil {
		return nil, err
	}
	return convert.FromStringDict(globals), nil
}

// New returns a Skyhook that looks in the given directories for plugin files to
// run.  The directories are searched in order for files when Run is called.
func New(dirs []string) *Skyhook {
	return &Skyhook{
		dirs:     dirs,
		plugins:  map[string]*syntax.File{},
		readFile: ioutil.ReadFile,
		mu:       &sync.Mutex{},
	}
}

// Run looks for a file with the given filename, and runs it with the given globals
// passed to the script's global namespace. The return value is all convertible
// global variables from the script, which may include the passed-in globals.
func (s *Skyhook) Run(filename string, globals map[string]interface{}) (map[string]interface{}, error) {
	dict, err := convert.MakeStringDict(globals)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if f, ok := s.plugins[filename]; ok {
		s.mu.Unlock()
		return runFile(f, dict)
	}
	s.mu.Unlock()

	for _, d := range s.dirs {
		b, err := s.readFile(filepath.Join(d, filename))
		if err == nil {
			f, err := parseFile(filename, b, dict)
			if err != nil {
				return nil, err
			}
			s.mu.Lock()
			s.plugins[filename] = f
			s.mu.Unlock()
			return runFile(f, dict)
		}
	}
	return nil, fmt.Errorf("cannot find plugin file %q in any plugin directoy", filename)
}

// Reset clears all cached scripts.
func (s *Skyhook) Reset() {
	s.mu.Lock()
	s.plugins = map[string]*syntax.File{}
	s.mu.Unlock()
}

// Forget clears the cached script for the given filename.
func (s *Skyhook) Forget(filename string) {
	s.mu.Lock()
	delete(s.plugins, filename)
	s.mu.Unlock()
}
