package slowerdaddy

import (
	"errors"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
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
	limiter     *rate.Limiter
	conns       []*Allocator
	localLimit  int
	globalLimit int
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
		Listener:    ln,
		localLimit:  limitConn,
		globalLimit: limitTotal,
		limiter:     limiter,
	}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	alloc := NewAllocator(l.limiter, l.localLimit)
	newConn := NewConn(conn, alloc)

	l.mu.Lock()
	l.conns = append(l.conns, alloc)
	l.mu.Unlock()

	return newConn, nil
}

// SetGlobalLimit sets the limit of the bandwidth of all net.Conn connections currently active combined.
func (l *Listener) SetGlobalLimit(limit int) error {
	l.mu.Lock()
	l.limiter.SetLimit(rate.Limit(limit))
	l.limiter.SetBurst(limit)
	l.globalLimit = limit
	l.mu.Unlock()
	return nil
}

func (l *Listener) SetLocalLimit(newLocalLimit int) error {
	if newLocalLimit > l.globalLimit {
		return ErrLimitGreaterThanTotal
	}
	eg := errgroup.Group{}
	eg.SetLimit(len(l.conns))

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, alloc := range l.conns {
		alloc := alloc
		eg.TryGo(func() error {
			return alloc.SetLimit(newLocalLimit)
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	l.localLimit = newLocalLimit
	return nil
}
