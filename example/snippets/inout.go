package main

import (
	"fmt"
	"log"
	"time"

	"github.com/starlight-go/starlight"
)

// Starlight makes it easy to get values in and out of your starlark scripts.
// Just pass in pointers to values that you want changed, or callback functions
// that propagate data.

// In theory, starlight also returns all global variables set by the script, but
// in real programs, you need well-defined outputs for your calling code to act on.
// If I write a script that creates a variable called nate_is_awesome = 1337 ... your
// go code probably isn't going to care that the variable exists.

// The best way to do it is to write a "results" struct that you pass in, just
// as you would for any other function.

type Page struct {
	Name     string
	Date     time.Time
	Contents string
	IsDraft  bool
}

const code = `
def run():
  if "nate" in page.Name:
	  # capitalize words
	  page.Name = page.Name.title()
  page.Name += " " + page.Date.Format("2006/01/02")
  page.IsDraft = False
run()
`

func main() {
	p := &Page{
		Name:     "a story about nate",
		Date:     time.Now(),
		Contents: "I like to write go code.",
		IsDraft:  true,
	}
	globals := map[string]interface{}{
		"page": p,
	}
	_, err := starlight.Eval([]byte(code), globals, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v is draft: %v\n", p.Name, p.IsDraft)
}

// Running it:
// $ go run inout.go
// A Story About Nate 2018/12/07 is draft: false
