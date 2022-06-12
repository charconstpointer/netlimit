package slowerdaddy

import (
	"errors"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	// ErrLimitGreaterThanTotal is returned when the limit is greater than the total limit of the listener.
	ErrLimitGreaterThanTotal = errors.New("limit per conn cannot be greater than total limit")
)

var _ net.Listener = (*Listener)(nil)

// Listener is a net.Listener that allows to control the bandwidth of the net.Conn connections and the limiter itself.
type Listener struct {
	mu sync.Mutex
	net.Listener
	limiter    *rate.Limiter
	conns      []*Conn
	limitConn  int
	limitTotal int
}

// Listen returns a Listener that will be bound to addr with the specified limits.
// Listen uses WithLimit to create the Listener.
func Listen(network, addr string, limitTotal, limitConn int) (*Listener, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	limiter := rate.NewLimiter(rate.Limit(limitTotal), limitTotal)

	return &Listener{
		Listener:   ln,
		limitConn:  limitConn,
		limitTotal: limitTotal,
		limiter:    limiter,
	}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	alloc := NewAllocator(l.limiter, l.limitConn)
	newConn := &Conn{
		Conn:  conn,
		alloc: alloc,
	}
	l.conns = append(l.conns, newConn)

	return newConn, nil
}

// SetTotalLimit sets the limit of the bandwidth of all net.Conn connections currently active combined.
func (l *Listener) SetTotalLimit(limit int) error {
	l.mu.Lock()
	l.limiter.AllowN(time.Now(), l.limitTotal)
	l.limiter.SetLimit(rate.Limit(limit))
	l.limiter.SetBurst(limit)
	l.limitTotal = limit
	l.mu.Unlock()
	return nil
}

func (l *Listener) SetLocalLimit(limit int) error {
	if limit > l.limitTotal {
		return ErrLimitGreaterThanTotal
	}

	for _, conn := range l.conns {
		conn.alloc.SetLimit(limit)
	}
	return nil
}
