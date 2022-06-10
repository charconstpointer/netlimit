package slowerdaddy

import (
	"context"
	"log"
	"net"

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
	allow := make(chan int)
	requestQuota := len(b)
	c.alloc.quotaReqs <- &QuotaRequest{
		allowCh: allow,
		Value:   requestQuota,
		ConnID:  c.Conn.RemoteAddr().String(),
	}
	allowedQuota := <-allow
	quota := allowedQuota

	log.Println("reading", quota, "bytes")
	n, err = c.Conn.Read(b[:quota])
	if err != nil {
		return n, err
	}
	return n, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
// Write will obey quota rules set by Listener
func (c *Conn) Write(b []byte) (n int, err error) {
	requestQuota := len(b)
	allowedQuota := c.requestQuota(requestQuota)
	quota := allowedQuota
	written := 0
	for written < len(b) {
		tail := written + quota
		if tail > len(b) {
			tail = len(b)
		}
		n, err = c.Conn.Write(b[written:tail])
		if err != nil {
			return written, err
		}
		log.Println("writing", n, "bytes")
		written += n
		quota = len(b[written:])
		allowedQuota := c.requestQuota(quota)
		quota = allowedQuota
	}

	return written, err
}

func (c *Conn) requestQuota(q int) int {
	allow := make(chan int)
	c.alloc.quotaReqs <- &QuotaRequest{
		allowCh: allow,
		Value:   q,
		ConnID:  c.Conn.RemoteAddr().String(),
	}
	return <-allow
}
