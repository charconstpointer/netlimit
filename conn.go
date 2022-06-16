package netlimit

import (
	"context"
	"fmt"
	"net"
)

var _ net.Conn = (*Conn)(nil)

type Allocator interface {
	Alloc(ctx context.Context, n int) (int, error)
	SetLimit(limit int) error
}

// Conn is a net.Conn that obeys quota limits managed by Allocator
type Conn struct {
	net.Conn

	// a is the allocator that controls the quota requests and bandwidth allocations for this connection
	a Allocator

	// done is a channel used to signal that the connection is closed and ready to be gc'd
	done chan struct{}
}

// NewConn returns a new Conn
func NewConn(conn net.Conn, a Allocator) (*Conn, error) {
	if a == nil {
		return nil, fmt.Errorf("allocator cannot be nil")
	}
	return &Conn{
		Conn: conn,
		a:    a,
		done: make(chan struct{}, 1),
	}, nil
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
		// TODO: perhaps we could use io.CopyN here
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

// SetLimit sets the limit of the local limiter.
func (c *Conn) SetLimit(limit int) error {
	return c.a.SetLimit(limit)
}

// Close closes the connection.
func (c *Conn) Close() error {
	c.done <- struct{}{}
	return c.Conn.Close()
}
