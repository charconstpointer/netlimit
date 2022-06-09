package main

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
	if err := c.limiter.WaitN(context.Background(), c.allowed); err != nil {
		return 0, err
	}

	readN := c.allowed
	if len(b) < c.allowed {
		readN = len(b)
	}

	n, err = c.Conn.Read(b[:readN])
	if err != nil {
		return n, err
	}
	log.Println("read", n, err)
	return n, err
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	if err := c.limiter.WaitN(context.Background(), c.allowed); err != nil {
		return 0, err
	}

	n, err = c.Conn.Write(b[:c.allowed])
	log.Println("write", n, err)
	return n, err
}
