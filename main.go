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
	limitConn  = 100
	limitTotal = 1000
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
		_, _ = io.Copy(w, f)
	})

	go http.Serve(ln, handler)
	// var limits []int
	for {
		// newLimit := rand.Intn(100) + 1
		// limits = append(limits, newLimit)
		// log.Println("updating limit to", newLimit)
		// ln.SetConnLimit(newLimit)
		// avg := 0
		// for _, l := range limits {
		// 	avg += l
		// }
		// log.Println("avg limit", avg/len(limits))
		time.Sleep(time.Second)
	}
}
