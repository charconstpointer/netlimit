package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/charconstpointer/slowerdaddy/slowerdaddy"
)

func main() {
	limitConn := 100
	limitTotal := 9999
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

	go http.Serve(ln, handler)
	sc := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Enter a number:")
		line, err := sc.ReadString('\n')
		if err != nil {
			log.Println(err)
			continue
		}
		newLimit, err := strconv.Atoi(line[:len(line)-1])
		if err != nil {
			log.Println(err)
			continue
		}
		ln.SetConnLimit(newLimit)
	}
}
