package slowerdaddy

import (
	"context"
	"log"
	"net"
)

// Limiter is a rate limiter used to limit the bandwidth of a net.Conn.
type Limiter interface {
	WaitN(context.Context, int) error
}

// Conn is a net.Conn that obeys quota limits set by Listener
type Conn struct {
	net.Conn
	limiter Limiter
	limit   int
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
// Read will obey quota rules set by Listener
func (c *Conn) Read(b []byte) (n int, err error) {
	quota := c.limit
	if len(b) < c.limit {
		quota = len(b)
	}

	if err := c.limiter.WaitN(context.Background(), quota); err != nil {
		return 0, err
	}

	n, err = c.Conn.Read(b[:quota])
	if err != nil {
		return n, err
	}
	log.Println(c.RemoteAddr().String(), "read", n, err)
	return n, err
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
// Write will obey quota rules set by Listener
func (c *Conn) Write(b []byte) (n int, err error) {
	quota := c.limit
	written := 0
	for written < len(b) {
		if len(b[written:]) < c.limit {
			quota = len(b[written:])
		}
		if err := c.limiter.WaitN(context.Background(), quota); err != nil {
			return 0, err
		}
		tail := written + quota
		if tail > len(b) {
			tail = len(b)
		}
		n, err = c.Conn.Write(b[written:tail])
		if err != nil {
			return written, err
		}
		written += n
		log.Println(c.RemoteAddr().String(), "write", n, err)

	}

	return written, err
}
