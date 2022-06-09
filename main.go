package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	limit := 123 //kbps
	tln := NewThrottledListener(ln, limit)
	s := http.Server{
		ReadTimeout:  100 * time.Second,
		WriteTimeout: 100 * time.Second,
	}
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b := make([]byte, 9999999)
		sb := strings.Builder{}
		for i := 0; i < len(b); i++ {
			sb.WriteString("x")
		}
		fmt.Fprint(w, sb.String())
	})
	http.Serve(tln, s.Handler)
}

type ThrottledReaderWriter struct {
	rw io.ReadWriter
}
