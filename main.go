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
		err = ln.SetGlobalLimit(200)
		if err != nil {
			log.Println(err)
			return
		}

		time.Sleep(time.Second * 3)
		err := ln.SetLocalLimit(100)
		if err != nil {
			log.Println(err)
			return
		}
	}()
	go func() {
		err = http.Serve(ln, handler)
		if err != nil {
			return
		}
	}()
	for i := 0; i < 10; i++ {
		go func() {
			// time.Sleep(time.Second * 1)
			now := time.Now()
			res, err := http.Get("http://localhost:8080/")
			if err != nil {
				log.Println(err)
			}
			_ = res.Body.Close()
			ellapsed := time.Since(now)
			log.Printf("ellapsed: %v", ellapsed.Seconds())
		}()
	}
	time.Sleep(time.Hour)
}
