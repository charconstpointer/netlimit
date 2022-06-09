package main

import (
	"net"

	"golang.org/x/time/rate"
)

type Listener struct {
	l       net.Listener
	allowed int
}

func NewThrottledListener(l net.Listener, allowed int) net.Listener {
	return Listener{
		l:       l,
		allowed: allowed,
	}
}

// Accept waits for and returns the next connection to the listener.
func (l Listener) Accept() (net.Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}
	return &Conn{
		Conn:    conn,
		allowed: l.allowed,
		limiter: rate.NewLimiter(rate.Limit(l.allowed), l.allowed),
	}, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l Listener) Close() error {
	return l.l.Close()
}

// Addr returns the listener's network address.
func (l Listener) Addr() net.Addr {
	return l.l.Addr()
}
