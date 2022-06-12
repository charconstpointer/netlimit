package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
)

var (
	limitConn  = 12
	limitTotal = 512
	addr       = ":8080"
	proto      = "tcp"
	fileName   = "lorem"
)

func main() {
	ln, err := slowerdaddy.Listen(proto, addr, limitTotal, limitConn)
	if err != nil {
		log.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			time.Sleep(time.Second * 5)
			ln.SetLocalLimit(144)
		}()
		_, _ = io.Copy(w, f)
	})

	err = http.Serve(ln, handler)
	if err != nil {
		return
	}
}
