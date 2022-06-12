package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
)

var (
	limitConn  = 512
	limitTotal = 1024 // 1GB
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
			fmt.Println("starting to read file")
			<-time.After(time.Second * 1)
			fmt.Println("new limit")
			ln.SetLocalLimit(64)
			<-time.After(time.Second * 1)
			fmt.Println("new limit")
			ln.SetLocalLimit(700)
		}()
		_, _ = io.Copy(w, f)
	})

	err = http.Serve(ln, handler)
	if err != nil {
		return
	}
}
