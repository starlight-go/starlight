# Skyhook [![GoDoc](https://godoc.org/github.com/hippogryph/skyhook?status.svg)](https://godoc.org/github.com/hippogryph/skyhook)

<p align="center"><img src="https://user-images.githubusercontent.com/3185864/49255912-57317500-f3fb-11e8-9854-f217105a7248.png"/></p>

Skyhook is a wrapper library for google's [starlark](https://github.com/google/starlark-go)
embedded python-like language. Skyhook is intended to give you an easier-to-use
interface for running starlark scripts directly from your Go programs.  Starlark
is a dialect of python, and has a Go native interpreter, so you can let your
users extend your application without any external requirements.

## Example

You can call a script from go thusly:

```go

type contact struct {
    Name string
}
hello := func(s string) string {
    return "hello " + s
}

c := &contact{Name: "Bob"}

out, _ := skyhook.Eval(
    []byte("output = hi(c.Name)"), 
    map[string]interface{}{
        "c":c, 
        "hi":hello,
    })

fmt.Println(out["output"])

// prints "hello bob"
```

Eval expects either a filename, slice of bytes, or io.Reader as its argument containing the code, and then a map of global variables to populate the script with.

## Usage

Skyhook.New creates a plugin cache that will read and compile scripts on the fly.

Skyhook.Eval does all the compilation at call time.

## Inputs and Outputs

Starlark scripts (and skyhook scripts by extension) use global variables in the
script as the input.

Thus if args are `map[string]interface{}{"input":"hello"}`, the script may act
on the variable called input thusly:

```python
output = input + "world!"
```

When run, this script will create a value in the map returned from Run with the
key "output" and with the value "hello world!".

## Types

Skyhook automatically translates go types to starlark types. The types supported
are strings, bools, and any int, uint, or float type.  Also supported are
structs, slices, arrays, and maps that use the aforementioned types. Conversion
out of starlark scripts work in reverse much the same way.  You may also pass in
starlark.Value types directly, in which case they will be passed to the script
as-is (this is useful if you need custom behavior).

## Functions

You can pass go functions that the script can call by passing your function in
with the rest of the globals. Some caveats: starlark ints always come in as
int64, not "int".  The supported types are the same as ToValue.  Positional args
are passed to your function and converted with FromValue. Kwargs passed from
starlark are currently ignored.

## Caching

Since parsing scripts is non-zero work, skyhook caches the scripts it finds
after the first time they get run, so that further runs of the script will not
incur the disk read and parsing overhead. To make skyhook reparse a file
(perhaps because it's changed) use the Forget method for the specific file, or
Reset to remove all cached files.