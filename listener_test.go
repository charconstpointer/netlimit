package netlimit_test

import (
	"net"
	"testing"

	"github.com/charconstpointer/netlimit"
)

func TestSetLocalLimit(t *testing.T) {
	limits := []int{1, 5, 7}
	ln, err := netlimit.Listen("tcp", ":8080", 10, 1)
	if err != nil {
		t.Errorf("Listen() error = %v", err)
	}
	done := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-done:
				ln.Close()
				t.Logf("Listener closed")
				return
			default:
			}
			c, err := ln.Accept()
			if err != nil {
				t.Errorf("Accept() error = %v", err)
				return
			}
			for _, limit := range limits {
				b := make([]byte, limit)
				_, err := c.Read(b)
				if err != nil {
					t.Errorf("Read() error = %v", err)
				}
			}
		}
	}()
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		t.Errorf("Dial() error = %v", err)
	}

	for _, limit := range limits {
		limit := limit
		ln.SetLocalLimit(limit)
		b := make([]byte, limit)
		n, err := conn.Write(b)
		if err != nil {
			t.Errorf("Write() error = %v", err)
		}
		if n != limit {
			t.Errorf("Write() = %v, want %v", n, limit)
		}
	}
	select {
	case done <- struct{}{}:
	default:
		t.Fatalf("cannot clean up test")
	}
}
