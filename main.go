package main

import (
	"context"
	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	limitConn  = 50
	limitTotal = 50
	addr       = ":8080"
	proto      = "tcp"
	fileName   = "lorem"
)

func main() {
	ln, err := slowerdaddy.Listen(proto, addr, limitTotal, limitConn)
	if err != nil {
		log.Fatal(err)
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		_, _ = io.Copy(w, f)
	})
	go func() {
		time.Sleep(time.Second * 5)
		err = ln.SetTotalLimit(200)
		if err != nil {
			log.Println(err)
			return
		}

		time.Sleep(time.Second * 3)
		err := ln.SetLocalLimit(context.Background(), 100)
		if err != nil {
			log.Println(err)
			return
		}
	}()
	err = http.Serve(ln, handler)
	if err != nil {
		return
	}
}
