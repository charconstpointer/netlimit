package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	limit := 1024
	tln := NewThrottledListener(ln, limit)
	s := http.Server{
		ReadTimeout:  100 * time.Second,
		WriteTimeout: 100 * time.Second,
	}
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, err := os.Open("lorem")
		if err != nil {
			log.Fatal(err)
		}
		_, _ = io.Copy(w, f)
	})
	http.Serve(tln, s.Handler)
}

type ThrottledReaderWriter struct {
	rw io.ReadWriter
}
