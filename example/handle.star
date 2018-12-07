# Globals are:
# w: the http.ResponseWriter for the request
# r: the *http.request
# Fprintf: fmt.Fprintf

# for loops and if statements need to be in functions in starlark
def main():
  # Query returns a map[string][]string
  
  # this gets a value from a map, with a default it it doesn't exist
  # and then takes the first value in the list.
  repeat = r.URL.Query().get("repeat", ["1"])[0]
  name = r.URL.Query().get("name", ["starlight"])[0]

  for x in range(int(repeat)):
    Fprintf(w, "hello %s\n", name)

  # we can use pythonic truthy statements on the slices returned from the map to
  # check if they're empty.
  if not r.URL.Query().get("repeat") and not r.URL.Query().get("repeat"):
    w.Write("\nadd ?repeat=<int>&name=<string> to the URL to customize this output\n")

  w.Write("\ntry modifying the contents of output.star and see what happens.\n")

main()