package slowerdaddy

import (
	"context"
	"fmt"
	"net"
)

var _ net.Conn = (*Conn)(nil)

type Allocator interface {
	Alloc(ctx context.Context, n int) (int, error)
}

// Conn is a net.Conn that obeys quota limits set by Listener
type Conn struct {
	net.Conn
	a Allocator
}

// NewConn returns a new Conn that obeys quota limits set by Listener
func NewConn(conn net.Conn, a Allocator) *Conn {
	return &Conn{Conn: conn, a: a}
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
// Read will obey quota rules set by Listener
func (c *Conn) Read(b []byte) (n int, err error) {
	ctx := context.Background()
	granted, err := c.a.Alloc(ctx, len(b))
	if err != nil {
		return 0, fmt.Errorf("failed to allocate quota: %w", err)
	}

	return c.Conn.Read(b[:granted])
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
// Write will obey quota rules set by Listener
func (c *Conn) Write(b []byte) (n int, err error) {
	ctx := context.Background()
	granted, err := c.a.Alloc(ctx, len(b))
	if err != nil {
		return 0, fmt.Errorf("failed to allocate quota: %w", err)
	}

	written := 0
	total := len(b)
	for written < total {
		tail := written + granted
		if tail > total {
			tail = total
		}

		n, err = c.Conn.Write(b[written:tail])
		if err != nil {
			return written, err
		}

		written += n
		quotaToRequest := len(b[written:])
		if quotaToRequest == 0 {
			break
		}
		granted, err = c.a.Alloc(ctx, quotaToRequest)
		if err != nil {
			return written, fmt.Errorf("failed to allocate quota: %w", err)
		}
	}
	return written, err
}
