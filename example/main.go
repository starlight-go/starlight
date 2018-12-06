package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/starlight-go/starlight"
)

func main() {
	http.HandleFunc("/", handle)
	port := "8080"
	fmt.Printf("running web server on http://localhost:%v\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	_, err := starlight.Eval("output.star", map[string]interface{}{
		"r":       r,
		"w":       w,
		"Fprintf": fmt.Fprintf,
	}, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}
