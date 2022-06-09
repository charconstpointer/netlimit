package slowerdaddy

import (
	"context"
	"log"
	"net"
)

type Limiter interface {
	WaitN(context.Context, int) error
}

type Conn struct {
	net.Conn
	limiter Limiter
	allowed int
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	quota := c.allowed
	if len(b) < c.allowed {
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
func (c *Conn) Write(b []byte) (n int, err error) {
	quota := c.allowed
	written := 0
	for written < len(b) {
		if len(b[written:]) < c.allowed {
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
