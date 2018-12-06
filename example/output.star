# Globals are:
# w, the http.ResponseWriter for the request
# r, the *http.request
# FPrintf, fmt.Printf

w.Write("hello from starlight!\n")
Fprintf(w, "Method: %v, URL: %s\n", r.Method, r.URL)
