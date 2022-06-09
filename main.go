package main

import (
	"io"
	"log"
	"os"
	"time"
)

func main() {
	f, err := os.Create("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString("Hello, World!")
	f.Sync()
	f.Close()

	f, err = os.Open("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	tr := NewThrottledReader(f, 1)
	n, err := io.Copy(os.Stdout, tr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Read", n, "bytes")
}

type ThrottledReader struct {
	io.Reader
	last  time.Time
	limit int64
	rem   int64
}

func NewThrottledReader(reader io.Reader, limit int64) *ThrottledReader {
	return &ThrottledReader{
		Reader: reader,
		limit:  limit,
		rem:    limit,
		last:   time.Now(),
	}
}

func (r *ThrottledReader) Read(p []byte) (n int, err error) {
	ellapsed := time.Since(r.last)
	newTokens := int64(ellapsed) / int64(time.Second)
	allowedTokens := r.limit - r.rem
	if allowedTokens > 0 {
		r.rem += newTokens
	}
	n, err = r.Reader.Read(p[:r.rem])
	if err != nil {
		return n, err
	}
	if n > 0 {
		r.rem -= int64(n)
		r.last = time.Now()
	}
	return n, nil
}
