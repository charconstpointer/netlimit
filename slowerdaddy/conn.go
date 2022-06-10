package slowerdaddy

import (
	"context"
	"log"
	"net"
	"time"

	"golang.org/x/time/rate"
)

// Limiter is a rate limiter used to limit the bandwidth of a net.Conn.
type Limiter interface {
	WaitN(context.Context, int) error
	SetLimit(rate.Limit)
	SetBurst(int)
}

// Conn is a net.Conn that obeys quota limits set by Listener
type Conn struct {
	net.Conn
	alloc *Allocator
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
// Read will obey quota rules set by Listener
func (c *Conn) Read(b []byte) (n int, err error) {
	quotaToRequest := len(b)
	if quotaToRequest > c.alloc.limit {
		quotaToRequest = c.alloc.limit
	}
	quota, ok := c.alloc.TryAlloc(quotaToRequest)
	for !ok {
		quota, ok = c.alloc.TryAlloc(quotaToRequest)
	}

	return c.Conn.Read(b[:quota])
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
// Write will obey quota rules set by Listener
func (c *Conn) Write(b []byte) (n int, err error) {
	quotaToRequest := len(b)
	var ok bool
	if quotaToRequest > c.alloc.limit {
		quotaToRequest = c.alloc.limit
	}

	grantedQuota, ok := c.alloc.TryAlloc(quotaToRequest)
	for !ok {
		grantedQuota, ok = c.alloc.TryAlloc(quotaToRequest)
		time.Sleep(time.Second)
	}

	written := 0
	total := len(b)
	for written < total {
		tail := written + grantedQuota
		if tail > total {
			tail = total
		}

		n, err = c.Conn.Write(b[written:tail])
		if err != nil {
			return written, err
		}
		log.Println("wrote", n, "bytes")

		written += n
		quotaToRequest = len(b[written:])
		grantedQuota, ok = c.alloc.TryAlloc(quotaToRequest)
		for !ok {
			grantedQuota, ok = c.alloc.TryAlloc(quotaToRequest)
			time.Sleep(time.Second)
		}
	}

	return written, err
}
