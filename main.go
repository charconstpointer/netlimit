package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
)

func main() {
	limit := 1 * 100
	ln, err := slowerdaddy.Listen("tcp", ":8080", limit)
	if err != nil {
		log.Fatal(err)
	}

	s := http.Server{}
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, err := os.Open("lorem")
		if err != nil {
			log.Fatal(err)
		}
		_, _ = io.Copy(w, f)
	})
	http.Serve(ln, s.Handler)
}

type ThrottledReaderWriter struct {
	rw io.ReadWriter
}
