package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
)

func main() {
	limitConn := 1 * 100
	limitTotal := 1 * 100
	ln, err := slowerdaddy.Listen("tcp", ":8080", limitTotal, limitConn)
	if err != nil {
		log.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, err := os.Open("lorem")
		if err != nil {
			log.Fatal(err)
		}
		_, _ = io.Copy(w, f)
	})

	http.Serve(ln, handler)
}
