package main

import (
	"log"
	"net/http"

	"github.com/charconstpointer/netlimit"
)

func main() {
	ln, err := netlimit.Listen("tcp", ":8080", 1024, 1024)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	}))
}
