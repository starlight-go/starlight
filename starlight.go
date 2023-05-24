// Package starlight provides a convenience wrapper around go.starlark.net/starlark.
package starlight

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/1set/starlight/convert"
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
	filename, ok := src.(string)
	if ok {
		dict, err = starlark.ExecFile(thread, filename, nil, dict)
	} else {
		dict, err = starlark.ExecFile(thread, "eval.sky", src, dict)
	}
	if err != nil {
		return nil, err
	}
	return convert.FromStringDict(dict), nil
}

// Cache is a cache of scripts to avoid re-reading files and reparsing them.
type Cache struct {
	dirs  []string
	cache *cache

	mu      sync.Mutex
	scripts map[string]*starlark.Program
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

// New returns a Starlight Cache that looks in the given directories for plugin
// files to run.  The directories are searched in order for files when Run is
// called.  Calls to the script function load() will also look in these
// directories. This function will panic if you give it no directories.
func New(dirs ...string) *Cache {
	if len(dirs) == 0 {
		panic(fmt.Errorf("no directories given"))
	}
	return newCache(dirs, nil)
}

// WithGlobals returns a new Starlight cache that passes the listed global
// values to scripts loaded with the load() script function.  Note that these
// globals will *not* be passed to individual scripts you run unless you
// explicitly pass them in the Run call.
func WithGlobals(globals map[string]interface{}, dirs ...string) (*Cache, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("no directories given")
	}
	g, err := convert.MakeStringDict(globals)
	if err != nil {
		return nil, err
	}
	return newCache(dirs, g), nil
}

func newCache(dirs []string, globals starlark.StringDict) *Cache {
	c := &Cache{
		dirs:    dirs,
		scripts: map[string]*starlark.Program{},
	}
	c.cache = &cache{
		cache:    make(map[string]*entry),
		readFile: c.readFile,
		globals:  globals,
	}
	return c
}

// Run looks for a file with the given filename, and runs it with the given globals
// passed to the script's global namespace. The return value is all convertible
// global variables from the script, which may include the passed-in globals.
func (c *Cache) Run(filename string, globals map[string]interface{}) (map[string]interface{}, error) {
	dict, err := convert.MakeStringDict(globals)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if p, ok := c.scripts[filename]; ok {
		c.mu.Unlock()
		return run(p, globals, c.load)
	}
	c.mu.Unlock()

	b, err := c.readFile(filename)
	if err != nil {
		return nil, err
	}
	_, p, err := starlark.SourceProgram(filename, b, dict.Has)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.scripts[filename] = p
	c.mu.Unlock()
	return run(p, globals, c.load)
}

func (c *Cache) load(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	return c.cache.Load(module)
}

func (c *Cache) readFile(filename string) ([]byte, error) {
	var err error
	var b []byte
	for _, d := range c.dirs {
		b, err = ioutil.ReadFile(filepath.Join(d, filename))
		if err == nil {
			return b, nil
		}
	}
	// guaranteed to have at least one directory, so there should be at least
	// not found error here.
	return nil, fmt.Errorf("cannot find file %q in any of the configured directories %q", filename, c.dirs)
}

// Reset clears all cached scripts.
func (c *Cache) Reset() {
	c.mu.Lock()
	c.scripts = map[string]*starlark.Program{}
	c.cache.reset()
	c.mu.Unlock()
}

// Forget clears the cached script for the given filename.
func (c *Cache) Forget(filename string) {
	c.mu.Lock()
	c.cache.remove(filename)
	delete(c.scripts, filename)
	c.mu.Unlock()
}
