// Package skyhook provides a convenience wrapper around github.com/google/starlark.
package skyhook

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/go-skyhook/skyhook/convert"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func init() {
	resolve.AllowNestedDef = true // allow def statements within function bodies
	resolve.AllowLambda = true    // allow lambda expressions
	resolve.AllowFloat = true     // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true       // allow the 'set' built-in
	resolve.AllowBitwise = true   // allow bitwise operations
}

// LoadFunc is a function that tells starlark how to find and load other scripts
// using the load() function.  If you don't use load() in your scripts, you can pass in nil.
type LoadFunc func(thread *starlark.Thread, module string) (starlark.StringDict, error)

// Eval evaluates the starlark source with the given global variables. The type
// of the argument for the src parameter must be string (filename), []byte, or io.Reader.
func Eval(src interface{}, globals map[string]interface{}, load LoadFunc) (map[string]interface{}, error) {
	dict, err := convert.MakeStringDict(globals)
	if err != nil {
		return nil, err
	}
	thread := &starlark.Thread{
		Load: load,
	}
	dict, err = starlark.ExecFile(thread, "eval.sky", src, dict)
	if err != nil {
		return nil, err
	}
	return convert.FromStringDict(dict), nil
}

// Skyhook is a script/plugin runner.
type Skyhook struct {
	dirs     []string
	readFile func(filename string) ([]byte, error)
	load     LoadFunc

	mu      *sync.Mutex
	plugins map[string]*starlark.Program
}

func run(p *starlark.Program, globals map[string]interface{}, load LoadFunc) (map[string]interface{}, error) {
	g, err := convert.MakeStringDict(globals)
	if err != nil {
		return nil, err
	}
	ret, err := p.Init(&starlark.Thread{Load: load}, g)
	if err != nil {
		return nil, err
	}
	return convert.FromStringDict(ret), nil
}

// New returns a Skyhook that looks in the given directories for plugin files to
// run.  The directories are searched in order for files when Run is called.
func New(dirs []string) *Skyhook {
	return &Skyhook{
		// TODO: make a load function here that works
		load:     nil,
		dirs:     dirs,
		plugins:  map[string]*starlark.Program{},
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
	if p, ok := s.plugins[filename]; ok {
		s.mu.Unlock()
		return run(p, globals, s.load)
	}
	s.mu.Unlock()

	for _, d := range s.dirs {
		b, err := s.readFile(filepath.Join(d, filename))
		if err == nil {
			_, p, err := starlark.SourceProgram(filename, b, dict.Has)
			if err != nil {
				return nil, err
			}
			s.mu.Lock()
			s.plugins[filename] = p
			s.mu.Unlock()
			return run(p, globals, s.load)
		}
	}
	return nil, fmt.Errorf("cannot find plugin file %q in any plugin directoy", filename)
}

// Reset clears all cached scripts.
func (s *Skyhook) Reset() {
	s.mu.Lock()
	s.plugins = map[string]*starlark.Program{}
	s.mu.Unlock()
}

// Forget clears the cached script for the given filename.
func (s *Skyhook) Forget(filename string) {
	s.mu.Lock()
	delete(s.plugins, filename)
	s.mu.Unlock()
}
