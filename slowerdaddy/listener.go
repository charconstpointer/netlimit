package slowerdaddy

import (
	"errors"
	"net"

	"golang.org/x/time/rate"
)

var (
	// ErrLimitGreaterThanTotal is returned when the limit is greater than the total limit of the listener.
	ErrLimitGreaterThanTotal = errors.New("limit per conn cannot be greater than total limit")
)

// Listener is a net.Listener that allows to control the bandwidth of the net.Conn connections and the limiter itself.
type Listener struct {
	net.Listener
	// limitConn is the limit of the bandwidth of a single net.Conn.
	limitConn int
	// limitTotal is the limit of the bandwidth of all net.Conn connections currently active combined.
	limitTotal int
}

// Listen returns a Listener that will be bound to addr with the specified limits.
// Listen uses WithLimit to create the Listener.
func Listen(network, addr string, limitTotal, limitConn int) (*Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return WithLimit(ln, limitConn, limitTotal), nil
}

// WithLimit returns a Listener that will be bound to addr with the specified limits.
func WithLimit(l net.Listener, limitConn, limitTotal int) *Listener {
	return &Listener{
		Listener:   l,
		limitConn:  limitConn,
		limitTotal: limitTotal,
	}
}

// Accept waits for and returns the next connection to the listener.
func (l Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &Conn{
		Conn:    conn,
		limit:   l.limitConn,
		limiter: rate.NewLimiter(rate.Limit(l.limitConn), l.limitConn),
	}, nil
}

// SetConnLimit sets the limit of the bandwidth of a single net.Conn.
func (l *Listener) SetConnLimit(limit int) error {
	if limit > l.limitTotal {
		return ErrLimitGreaterThanTotal
	}

	l.limitConn = limit
	return nil
}

// SetTotalLimit sets the limit of the bandwidth of all net.Conn connections currently active combined.
func (l *Listener) SetTotalLimit(limit int) error {
	l.limitTotal = limit
	return nil
}
