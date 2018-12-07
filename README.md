# <img src="https://user-images.githubusercontent.com/3185864/49534746-5b90de80-f890-11e8-9fd6-5417cf915c67.png"/> Starlight [![GoDoc](https://godoc.org/github.com/starlight-go/starlight?status.svg)](https://godoc.org/github.com/starlight-go/starlight) [![Build Status](https://travis-ci.org/starlight-go/starlight.svg?branch=master)](https://travis-ci.org/starlight-go/starlight)


<p align="center" style="font-weight:bold">!! Starlight is still a WIP !!<p/>


Starlight is a wrapper library for google's [starlark](https://github.com/google/starlark-go)
embedded python-like language. Starlight is intended to give you an easier-to-use
interface for running starlark scripts directly from your Go programs.  Starlark
is a dialect of python, and has a Go native interpreter, so you can let your
users extend your application without any external requirements.


## Sample

You can call a script from go thusly:

```go

import (
    "fmt"
    "github.com/starlight-go/starlight"
)

type contact struct {
    Name string
}

func main() {
    c := &contact{Name: "Bob"}
    globals := map[string]interface{}{
        "contact":c, 
        "Println":fmt.Println,
    }

    script := []byte(`
contact.Name = "Phil"
Println("hello " + contact.Name)
`)
    // errors will tell you about syntax/runtime errors.
    _, err := starlight.Eval(script, globals, nil)
}

// prints "hello Phil"
// also the value of c's Name field will now be Phil when referenced from Go code as well.
```

Eval expects either a filename, slice of bytes, or io.Reader as its argument
containing the code, and then a map of global variables to populate the script
with.

## Usage

Starlight.New creates a script cache that will read and compile scripts on the fly, caching those it has already run.

Starlight.Eval does all the compilation at call time.

## Inputs and Outputs

Starlark scripts (and starlight scripts by extension) use global variables in the
script as the input.

Thus if args are `map[string]interface{}{"input":"hello"}`, the script may act
on the variable called input thusly:

```python
output = input + "world!"
```

When run, this script will create a value in the map returned with the
key "output" and with the value "hello world!".

## Types

Starlight automatically translates go types to starlark types. Starlight
supports almost every go type except channels.   You may also pass in types that
implement starlark.Value themselves, in which case they will be passed to the
script as-is (this is useful if you need custom behavior).

## Functions

You can pass go functions that the script can call by passing your function in
with the rest of the globals. Positional args are passed to your function and
converted to their appropriate go type if possible. Kwargs passed from starlark
scripts are currently ignored.

## Caching

Since parsing scripts is non-zero work, starlight caches the scripts it finds
after the first time they get run, so that further runs of the script will not
incur the disk read and parsing overhead. To make starlight reparse a file
(perhaps because it has changed) use the Forget method for the specific file, or
Reset to remove all cached files.

## Example

The [example](https://github.com/starlight-go/starlight/tree/master/example)
directory shows an example of using starlight to run scripts that modify the
output of a running web server.

## Why?

**Why not just use starlark-go directly?**

Well, it's actually quite difficult to get go data *into* a starlark-go script. Starlark as written is made more for configuration, so you mostly get data *out* of scripts, and mostly just basic values.

For example, structs just aren't supported, unless you write a complicated wrapper (starlight does that for you). Adapting a function to starlark-go requires a bunch of boilerplate adapter and conversion code, and then if you're using anything other than basic types (int, string, bool, float, etc), you need adapters for those, too (starlight does that for you, too).

**Why embed python in your Go application?**

Because it lets you add flexibility without having to recompile.  It lets users customize your application with just a few lines of python.

Also, Starlark is *safe*.  You can run arbitrary code from third parties without worrying about it blowing up your machine or downloading nasty things from the internet (unless you give scripts the ability to do that).  Starlark code can only call the functions you allow.

## How?

Lots of reflection.  It's not as slow as you think.  Running a small pre-compiled script including the time to process the inputs using reflection, takes less than 2400ns on my 2017 macbook pro (that's 0.0024 milliseconds). 
