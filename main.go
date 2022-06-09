package main

import (
	"io"
	"log"
	"net"
	"net/http"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	limit := 256 //bps
	tln := NewThrottledListener(ln, limit)
	http.Serve(tln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	}))
}

type ThrottledReaderWriter struct {
	rw io.ReadWriter
}
