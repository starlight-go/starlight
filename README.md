# Skyhook [![GoDoc](https://godoc.org/github.com/natefinch/skyhook?status.svg)](https://godoc.org/github.com/natefinch/skyhook)


Skyhook is a wrapper for google's [skylark](https://github.com/google/skylark)
embedded python-like language. Skyhook is intended to give you an easier-to-use
interface for running skylark scripts directly from your Go programs.  Skylark
is a dialect of python, and has a Go native interpreter, so you can let your
users extend your application without any external requirements.

## Demo

Assume you have this file at plugins/foo.sky:

```python
def foo():
    return " world!"

output = input + foo()
```

You can call this script from go thusly:

```go
sky := skyhook.New([]string{"plugins"})
globals := map[string]interface{}{"input":"hello"}
globals, err := sky.Run("foo.sky", globals)
if err != nil {
    return err
}
fmt.Println(globals["output"])

// prints "hello world!"
```

## Usage

You give skyhook a list of directories to look in, and it will look
for skylark scripts in each of those directories, in order, until it finds a
script with the given name, and then run that script.

## Inputs and Outputs

Skylark scripts (and skyhook scripts by extension) use global variables in the
script as the input and output.  Args in Run are created as global variables in
the script with the given names.

Thus if args are `map[string]interface{}{"input":"hello"}`, the script may act
on the variable called input thusly:

```python
output = input + "world!"
```

When run, this script will create a value in the map returned from Run with the
key "output" and with the value "hello world!".

## Types

Skyhook automatically translates go types in Run's args map to skylark types.
The types supported are any int, uint, or float type, strings,
maps[interface{}]interface{}, map[interface{}]bool, []interface{}.  Where all
interface[] values must be one of the supported types.  Conversion out of
skylark scripts work in reverse much the same way.  You may also pass in
skylark.Value types directly, in which case they will be passed to the script
as-is.

## Functions

You can pass go functions that the script can call by passing your function in
with the rest of the globals. Some caveats: skylark ints always come in as
int64, not "int".  The supported types are the same as ToValue.  Positional args
are passed to your function and converted with FromValue. Kwargs passed from
skylark are currently ignored.

## Caching

Since parsing scripts is non-zero work, skyhook caches the scripts it finds
after the first time they get run, so that further runs of the script will not
incur the disk read and parsing overhead. To make skyhook reparse a file
(perhaps because it's changed) use the Forget method for the specific file, or
Reset to remove all cached files.